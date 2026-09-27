package wallet

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Transaction struct {
	ID                   string    `json:"id"`
	WalletID             string    `json:"wallet_id"`
	Type                 string    `json:"type"`
	AmountPaise          int64     `json:"amount_paise"`
	BalanceAfterPaise    int64     `json:"balance_after_paise"`
	ReferenceID          string    `json:"reference_id"`
	CounterpartyWalletID *string   `json:"counterparty_wallet_id,omitempty"`
	CounterpartyName     *string   `json:"counterparty_name,omitempty"`
	CounterpartyHandle   *string   `json:"counterparty_handle,omitempty"`
	Note                 string    `json:"note"`
	CreatedAt            time.Time `json:"created_at"`
}

type WalletResponse struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	BalancePaise        int64     `json:"balance_paise"`
	Status              string    `json:"status"`
	DailySendLimitPaise int64     `json:"daily_send_limit_paise"`
	SentTodayPaise      int64     `json:"sent_today_paise"`
	CreatedAt           time.Time `json:"created_at"`
}

type Service struct {
	db *sql.DB
}

func NewService(database *sql.DB) *Service {
	return &Service{db: database}
}

func (s *Service) GetWalletResponse(userID string) (*WalletResponse, error) {
	var w auth.Wallet
	err := s.db.QueryRow(`
		SELECT id, user_id, balance_paise, status, daily_send_limit_paise, created_at
		FROM wallets WHERE user_id = ?
	`, userID).Scan(&w.ID, &w.UserID, &w.BalancePaise, &w.Status, &w.DailySendLimitPaise, &w.CreatedAt)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
	}

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	var sentToday int64
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(amount_paise), 0)
		FROM transfers
		WHERE from_wallet_id = ? AND status = 'completed' AND created_at >= ?
	`, w.ID, startOfDay).Scan(&sentToday)
	if err != nil {
		sentToday = 0
	}

	return &WalletResponse{
		ID:                  w.ID,
		UserID:              w.UserID,
		BalancePaise:        w.BalancePaise,
		Status:              w.Status,
		DailySendLimitPaise: w.DailySendLimitPaise,
		SentTodayPaise:      sentToday,
		CreatedAt:           w.CreatedAt,
	}, nil
}

type TopUpRequest struct {
	AmountPaise int64  `json:"amount_paise"`
	Source      string `json:"source"`
}

type TopUpResponse struct {
	Wallet      *WalletResponse `json:"wallet"`
	Transaction *Transaction    `json:"transaction"`
}

func (s *Service) TopUp(userID string, amountPaise int64, source string) (*TopUpResponse, error) {
	if amountPaise < 100 || amountPaise > 5000000 { // ₹1 to ₹50,000
		return nil, &httpx.CustomError{Code: "invalid_amount", Message: "Top-up amount must be between ₹1.00 and ₹50,000.00", Status: http.StatusBadRequest}
	}

	source = strings.ToLower(source)
	if source != "upi" && source != "card" && source != "netbanking" {
		source = "upi"
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
		return nil, &httpx.CustomError{Code: "wallet_frozen", Message: "Wallet is frozen and cannot accept top-ups", Status: http.StatusForbidden}
	}

	newBalance := w.BalancePaise + amountPaise
	now := time.Now().UTC()
	txnID := db.NewID("txn_")
	refID := db.NewID("ref_")
	note := fmt.Sprintf("Top-up via %s", strings.ToUpper(source))

	_, err = tx.Exec(`
		UPDATE wallets SET balance_paise = ? WHERE id = ?
	`, newBalance, w.ID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
		VALUES (?, ?, 'top_up', ?, ?, ?, NULL, ?, ?)
	`, txnID, w.ID, amountPaise, newBalance, refID, note, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	walletRes, err := s.GetWalletResponse(userID)
	if err != nil {
		return nil, err
	}

	t := &Transaction{
		ID:                txnID,
		WalletID:          w.ID,
		Type:              "top_up",
		AmountPaise:       amountPaise,
		BalanceAfterPaise: newBalance,
		ReferenceID:       refID,
		Note:              note,
		CreatedAt:         now,
	}

	return &TopUpResponse{
		Wallet:      walletRes,
		Transaction: t,
	}, nil
}

type TransactionFilter struct {
	Type   string
	From   string
	To     string
	Limit  int
	Cursor string
}

func (s *Service) GetTransactions(userID string, filter TransactionFilter) ([]Transaction, error) {
	var walletID string
	err := s.db.QueryRow("SELECT id FROM wallets WHERE user_id = ?", userID).Scan(&walletID)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
	}

	query := `
		SELECT t.id, t.wallet_id, t.type, t.amount_paise, t.balance_after_paise, t.reference_id, 
		       t.counterparty_wallet_id, u.name, u.handle, t.note, t.created_at
		FROM transactions t
		LEFT JOIN wallets cw ON t.counterparty_wallet_id = cw.id
		LEFT JOIN users u ON cw.user_id = u.id
		WHERE t.wallet_id = ?
	`
	args := []any{walletID}

	if filter.Type != "" {
		query += " AND t.type = ?"
		args = append(args, filter.Type)
	}
	if filter.From != "" {
		if t, err := time.Parse("2006-01-02", filter.From); err == nil {
			query += " AND t.created_at >= ?"
			args = append(args, t.UTC())
		}
	}
	if filter.To != "" {
		if t, err := time.Parse("2006-01-02", filter.To); err == nil {
			// Include entire end day
			query += " AND t.created_at <= ?"
			args = append(args, t.Add(24*time.Hour).UTC())
		}
	}

	query += " ORDER BY t.created_at DESC"

	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	query += " LIMIT ?"
	args = append(args, filter.Limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var t Transaction
		var cpWalletID, cpName, cpHandle sql.NullString
		err := rows.Scan(
			&t.ID, &t.WalletID, &t.Type, &t.AmountPaise, &t.BalanceAfterPaise, &t.ReferenceID,
			&cpWalletID, &cpName, &cpHandle, &t.Note, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if cpWalletID.Valid {
			t.CounterpartyWalletID = &cpWalletID.String
		}
		if cpName.Valid {
			t.CounterpartyName = &cpName.String
		}
		if cpHandle.Valid {
			t.CounterpartyHandle = &cpHandle.String
		}
		txns = append(txns, t)
	}

	if txns == nil {
		txns = []Transaction{}
	}

	return txns, nil
}

func (s *Service) GenerateStatementCSV(userID string, from, to string) ([]byte, error) {
	txns, err := s.GetTransactions(userID, TransactionFilter{From: from, To: to, Limit: 1000})
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// Write header
	_ = writer.Write([]string{"Transaction ID", "Date", "Type", "Amount", "Balance After", "Counterparty", "Note", "Reference ID"})

	for _, t := range txns {
		cp := ""
		if t.CounterpartyName != nil && t.CounterpartyHandle != nil {
			cp = fmt.Sprintf("%s (%s)", *t.CounterpartyName, *t.CounterpartyHandle)
		}
		row := []string{
			t.ID,
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			t.Type,
			httpx.FormatPaise(t.AmountPaise),
			httpx.FormatPaise(t.BalanceAfterPaise),
			cp,
			t.Note,
			t.ReferenceID,
		}
		_ = writer.Write(row)
	}

	writer.Flush()
	return buf.Bytes(), nil
}
