package transfer

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Transfer struct {
	ID                 string    `json:"id"`
	FromWalletID       string    `json:"from_wallet_id"`
	FromUserName       string    `json:"from_user_name,omitempty"`
	FromUserHandle     string    `json:"from_user_handle,omitempty"`
	ToWalletID         string    `json:"to_wallet_id"`
	ToUserName         string    `json:"to_user_name,omitempty"`
	ToUserHandle       string    `json:"to_user_handle,omitempty"`
	AmountPaise        int64     `json:"amount_paise"`
	Note               string    `json:"note"`
	Status             string    `json:"status"`
	FailureReason      *string   `json:"failure_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type UserLookupResponse struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Handle   string `json:"handle"`
	WalletID string `json:"wallet_id"`
}

type Service struct {
	db *sql.DB
}

func NewService(database *sql.DB) *Service {
	return &Service{db: database}
}

func (s *Service) LookupUserByHandle(handle string) (*UserLookupResponse, error) {
	handle = auth.FormatHandle(handle)
	var res UserLookupResponse
	err := s.db.QueryRow(`
		SELECT u.id, u.name, u.handle, w.id
		FROM users u
		JOIN wallets w ON w.user_id = u.id
		WHERE u.handle = ?
	`, handle).Scan(&res.UserID, &res.Name, &res.Handle, &res.WalletID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "user_not_found", Message: "User handle not found", Status: http.StatusNotFound}
		}
		return nil, err
	}
	return &res, nil
}

type CreateTransferRequest struct {
	ToWalletID string `json:"to_wallet_id"`
	ToHandle   string `json:"to_handle"`
	AmountPaise int64 `json:"amount_paise"`
	Note       string `json:"note"`
}

func (s *Service) Transfer(fromUserID string, req CreateTransferRequest) (*Transfer, error) {
	if req.AmountPaise <= 0 {
		return nil, &httpx.CustomError{Code: "invalid_amount", Message: "Transfer amount must be greater than zero", Status: http.StatusBadRequest}
	}

	// 1. Resolve target wallet
	targetWalletID := req.ToWalletID
	if targetWalletID == "" && req.ToHandle != "" {
		lu, err := s.LookupUserByHandle(req.ToHandle)
		if err != nil {
			return nil, err
		}
		targetWalletID = lu.WalletID
	}
	if targetWalletID == "" {
		return nil, &httpx.CustomError{Code: "invalid_recipient", Message: "Target wallet or handle is required", Status: http.StatusBadRequest}
	}

	// 2. Load sender wallet
	var senderWallet auth.Wallet
	err := s.db.QueryRow(`
		SELECT id, user_id, balance_paise, status, daily_send_limit_paise, created_at
		FROM wallets WHERE user_id = ?
	`, fromUserID).Scan(&senderWallet.ID, &senderWallet.UserID, &senderWallet.BalancePaise, &senderWallet.Status, &senderWallet.DailySendLimitPaise, &senderWallet.CreatedAt)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Sender wallet not found", Status: http.StatusNotFound}
	}

	if senderWallet.Status != "active" {
		return nil, &httpx.CustomError{Code: "wallet_frozen", Message: "Sender wallet is frozen", Status: http.StatusForbidden}
	}

	// Step (a): Validate balance
	if senderWallet.BalancePaise < req.AmountPaise {
		return nil, &httpx.CustomError{Code: "insufficient_funds", Message: "Insufficient wallet balance", Status: http.StatusBadRequest}
	}

	transferID := db.NewID("trf_")
	refOutID := db.NewID("ref_")
	refInID := db.NewID("ref_")
	txnOutID := db.NewID("txn_")
	txnInID := db.NewID("txn_")
	now := time.Now().UTC()

	// Step (b): Debit the sender and write transfer_out
	senderWallet.BalancePaise -= req.AmountPaise
	_, err = s.db.Exec(`UPDATE wallets SET balance_paise = ? WHERE id = ?`, senderWallet.BalancePaise, senderWallet.ID)
	if err != nil {
		return nil, err
	}

	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = "Money transfer"
	}

	_, err = s.db.Exec(`
		INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
		VALUES (?, ?, 'transfer_out', ?, ?, ?, ?, ?, ?)
	`, txnOutID, senderWallet.ID, -req.AmountPaise, senderWallet.BalancePaise, refOutID, targetWalletID, note, now)
	if err != nil {
		return nil, err
	}

	// Insert initial transfer record
	_, err = s.db.Exec(`
		INSERT INTO transfers (id, from_wallet_id, to_wallet_id, amount_paise, note, status, failure_reason, created_at)
		VALUES (?, ?, ?, ?, ?, 'completed', NULL, ?)
	`, transferID, senderWallet.ID, targetWalletID, req.AmountPaise, note, now)
	if err != nil {
		return nil, err
	}

	// Step (c): Load receiver wallet and check active status & daily limit
	var receiverWallet auth.Wallet
	err = s.db.QueryRow(`
		SELECT id, user_id, balance_paise, status, daily_send_limit_paise, created_at
		FROM wallets WHERE id = ?
	`, targetWalletID).Scan(&receiverWallet.ID, &receiverWallet.UserID, &receiverWallet.BalancePaise, &receiverWallet.Status, &receiverWallet.DailySendLimitPaise, &receiverWallet.CreatedAt)

	if err != nil || receiverWallet.Status != "active" {
		failReason := "recipient_unavailable"
		_, _ = s.db.Exec(`UPDATE transfers SET status = 'failed', failure_reason = ? WHERE id = ?`, failReason, transferID)
		return nil, &httpx.CustomError{Code: "recipient_unavailable", Message: "Recipient wallet is unavailable or frozen", Status: http.StatusUnprocessableEntity}
	}

	// Check daily limit: calculate sent today before this transfer + amount
	startOfDay := now.Truncate(24 * time.Hour)
	var sentToday int64
	_ = s.db.QueryRow(`
		SELECT COALESCE(SUM(amount_paise), 0)
		FROM transfers
		WHERE from_wallet_id = ? AND status = 'completed' AND created_at >= ? AND id != ?
	`, senderWallet.ID, startOfDay, transferID).Scan(&sentToday)

	if sentToday+req.AmountPaise > senderWallet.DailySendLimitPaise {
		failReason := "daily_limit_exceeded"
		_, _ = s.db.Exec(`UPDATE transfers SET status = 'failed', failure_reason = ? WHERE id = ?`, failReason, transferID)
		return nil, &httpx.CustomError{Code: "daily_limit_exceeded", Message: "Daily transfer limit exceeded", Status: http.StatusUnprocessableEntity}
	}

	// Step (d): Credit the receiver
	receiverWallet.BalancePaise += req.AmountPaise
	_, err = s.db.Exec(`UPDATE wallets SET balance_paise = ? WHERE id = ?`, receiverWallet.BalancePaise, receiverWallet.ID)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
		VALUES (?, ?, 'transfer_in', ?, ?, ?, ?, ?, ?)
	`, txnInID, receiverWallet.ID, req.AmountPaise, receiverWallet.BalancePaise, refInID, senderWallet.ID, note, now)
	if err != nil {
		return nil, err
	}

	return s.GetTransferByID(transferID, fromUserID)
}

func (s *Service) GetTransferByID(id string, userID string) (*Transfer, error) {
	var tr Transfer
	var failReason sql.NullString
	err := s.db.QueryRow(`
		SELECT t.id, t.from_wallet_id, fu.name, fu.handle, t.to_wallet_id, tu.name, tu.handle,
		       t.amount_paise, t.note, t.status, t.failure_reason, t.created_at
		FROM transfers t
		JOIN wallets fw ON t.from_wallet_id = fw.id
		JOIN users fu ON fw.user_id = fu.id
		JOIN wallets tw ON t.to_wallet_id = tw.id
		JOIN users tu ON tw.user_id = tu.id
		WHERE t.id = ?
	`, id).Scan(
		&tr.ID, &tr.FromWalletID, &tr.FromUserName, &tr.FromUserHandle,
		&tr.ToWalletID, &tr.ToUserName, &tr.ToUserHandle,
		&tr.AmountPaise, &tr.Note, &tr.Status, &failReason, &tr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "transfer_not_found", Message: "Transfer not found", Status: http.StatusNotFound}
		}
		return nil, err
	}

	if failReason.Valid {
		tr.FailureReason = &failReason.String
	}

	return &tr, nil
}

func (s *Service) ListTransfers(userID string) ([]Transfer, error) {
	var walletID string
	err := s.db.QueryRow("SELECT id FROM wallets WHERE user_id = ?", userID).Scan(&walletID)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
	}

	rows, err := s.db.Query(`
		SELECT t.id, t.from_wallet_id, fu.name, fu.handle, t.to_wallet_id, tu.name, tu.handle,
		       t.amount_paise, t.note, t.status, t.failure_reason, t.created_at
		FROM transfers t
		JOIN wallets fw ON t.from_wallet_id = fw.id
		JOIN users fu ON fw.user_id = fu.id
		JOIN wallets tw ON t.to_wallet_id = tw.id
		JOIN users tu ON tw.user_id = tu.id
		WHERE t.from_wallet_id = ? OR t.to_wallet_id = ?
		ORDER BY t.created_at DESC
		LIMIT 100
	`, walletID, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []Transfer
	for rows.Next() {
		var tr Transfer
		var failReason sql.NullString
		err := rows.Scan(
			&tr.ID, &tr.FromWalletID, &tr.FromUserName, &tr.FromUserHandle,
			&tr.ToWalletID, &tr.ToUserName, &tr.ToUserHandle,
			&tr.AmountPaise, &tr.Note, &tr.Status, &failReason, &tr.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if failReason.Valid {
			tr.FailureReason = &failReason.String
		}
		transfers = append(transfers, tr)
	}

	if transfers == nil {
		transfers = []Transfer{}
	}

	return transfers, nil
}
