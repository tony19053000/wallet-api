package withdrawal

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Withdrawal struct {
	ID            string    `json:"id"`
	WalletID      string    `json:"wallet_id"`
	BankAccountID string    `json:"bank_account_id"`
	BankName      string    `json:"bank_name,omitempty"`
	AccountLast4  string    `json:"account_last4,omitempty"`
	AmountPaise   int64     `json:"amount_paise"`
	FeePaise      int64     `json:"fee_paise"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateWithdrawalRequest struct {
	BankAccountID string `json:"bank_account_id"`
	AmountPaise   int64  `json:"amount_paise"`
}

type Service struct {
	db *sql.DB
}

func NewService(database *sql.DB) *Service {
	return &Service{db: database}
}

func (s *Service) Withdraw(userID string, req CreateWithdrawalRequest) (*Withdrawal, error) {
	if req.AmountPaise <= 0 {
		return nil, &httpx.CustomError{Code: "invalid_amount", Message: "Withdrawal amount must be greater than zero", Status: http.StatusBadRequest}
	}

	// Verify bank account ownership
	var bankName, accountLast4 string
	err := s.db.QueryRow(`
		SELECT bank_name, account_last4 FROM bank_accounts WHERE id = ? AND user_id = ?
	`, req.BankAccountID, userID).Scan(&bankName, &accountLast4)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "bank_account_not_found", Message: "Bank account not found or does not belong to user", Status: http.StatusNotFound}
		}
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var w auth.Wallet
	err = tx.QueryRow(`
		SELECT id, user_id, balance_paise, status, daily_send_limit_paise, created_at
		FROM wallets WHERE user_id = ?
	`, userID).Scan(&w.ID, &w.UserID, &w.BalancePaise, &w.Status, &w.DailySendLimitPaise, &w.CreatedAt)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
	}

	if w.Status != "active" {
		return nil, &httpx.CustomError{Code: "wallet_frozen", Message: "Wallet is frozen and cannot perform withdrawals", Status: http.StatusForbidden}
	}

	// Balance check
	if w.BalancePaise < req.AmountPaise {
		return nil, &httpx.CustomError{Code: "insufficient_funds", Message: "Insufficient wallet balance", Status: http.StatusBadRequest}
	}

	feePaise := int64(200) // flat ₹2
	newBalance := w.BalancePaise - (req.AmountPaise + feePaise)
	wdrID := db.NewID("wdr_")
	refWdrID := db.NewID("ref_")
	refFeeID := db.NewID("ref_")
	txnWdrID := db.NewID("txn_")
	txnFeeID := db.NewID("txn_")
	now := time.Now().UTC()

	_, err = tx.Exec(`UPDATE wallets SET balance_paise = ? WHERE id = ?`, newBalance, w.ID)
	if err != nil {
		return nil, err
	}

	wdrNote := fmt.Sprintf("Bank withdrawal to %s •••• %s", bankName, accountLast4)
	balAfterWdr := w.BalancePaise - req.AmountPaise

	_, err = tx.Exec(`
		INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
		VALUES (?, ?, 'withdrawal', ?, ?, ?, NULL, ?, ?)
	`, txnWdrID, w.ID, -req.AmountPaise, balAfterWdr, refWdrID, wdrNote, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
		VALUES (?, ?, 'fee', ?, ?, ?, NULL, 'Withdrawal fee', ?)
	`, txnFeeID, w.ID, -feePaise, newBalance, refFeeID, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		INSERT INTO withdrawals (id, wallet_id, bank_account_id, amount_paise, fee_paise, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'completed', ?)
	`, wdrID, w.ID, req.BankAccountID, req.AmountPaise, feePaise, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Withdrawal{
		ID:            wdrID,
		WalletID:      w.ID,
		BankAccountID: req.BankAccountID,
		BankName:      bankName,
		AccountLast4:  accountLast4,
		AmountPaise:   req.AmountPaise,
		FeePaise:      feePaise,
		Status:        "completed",
		CreatedAt:     now,
	}, nil
}

func (s *Service) ListWithdrawals(userID string) ([]Withdrawal, error) {
	var walletID string
	err := s.db.QueryRow("SELECT id FROM wallets WHERE user_id = ?", userID).Scan(&walletID)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
	}

	rows, err := s.db.Query(`
		SELECT w.id, w.wallet_id, w.bank_account_id, b.bank_name, b.account_last4,
		       w.amount_paise, w.fee_paise, w.status, w.created_at
		FROM withdrawals w
		JOIN bank_accounts b ON w.bank_account_id = b.id
		WHERE w.wallet_id = ?
		ORDER BY w.created_at DESC
	`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []Withdrawal
	for rows.Next() {
		var wdr Withdrawal
		err := rows.Scan(
			&wdr.ID, &wdr.WalletID, &wdr.BankAccountID, &wdr.BankName, &wdr.AccountLast4,
			&wdr.AmountPaise, &wdr.FeePaise, &wdr.Status, &wdr.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, wdr)
	}

	if withdrawals == nil {
		withdrawals = []Withdrawal{}
	}

	return withdrawals, nil
}
