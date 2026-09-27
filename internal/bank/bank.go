package bank

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
)

type BankAccount struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	BankName      string    `json:"bank_name"`
	AccountHolder string    `json:"account_holder"`
	AccountLast4  string    `json:"account_last4"`
	IFSC          string    `json:"ifsc"`
	IsPrimary     bool      `json:"is_primary"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateBankAccountRequest struct {
	BankName      string `json:"bank_name"`
	AccountHolder string `json:"account_holder"`
	AccountNumber string `json:"account_number"`
	IFSC          string `json:"ifsc"`
}

type Service struct {
	db *sql.DB
}

func NewService(database *sql.DB) *Service {
	return &Service{db: database}
}

func (s *Service) ListBankAccounts(userID string) ([]BankAccount, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, bank_name, account_holder, account_last4, ifsc, is_primary, created_at
		FROM bank_accounts
		WHERE user_id = ?
		ORDER BY is_primary DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []BankAccount
	for rows.Next() {
		var ba BankAccount
		err := rows.Scan(
			&ba.ID, &ba.UserID, &ba.BankName, &ba.AccountHolder, &ba.AccountLast4,
			&ba.IFSC, &ba.IsPrimary, &ba.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, ba)
	}

	if accounts == nil {
		accounts = []BankAccount{}
	}

	return accounts, nil
}

func (s *Service) AddBankAccount(userID string, req CreateBankAccountRequest) (*BankAccount, error) {
	req.BankName = strings.TrimSpace(req.BankName)
	req.AccountHolder = strings.TrimSpace(req.AccountHolder)
	req.AccountNumber = strings.TrimSpace(req.AccountNumber)
	req.IFSC = strings.ToUpper(strings.TrimSpace(req.IFSC))

	if req.BankName == "" || req.AccountHolder == "" || req.AccountNumber == "" || req.IFSC == "" {
		return nil, &httpx.CustomError{Code: "invalid_bank_details", Message: "All bank account details are required", Status: http.StatusBadRequest}
	}

	if len(req.AccountNumber) < 4 {
		return nil, &httpx.CustomError{Code: "invalid_account_number", Message: "Account number must be at least 4 digits", Status: http.StatusBadRequest}
	}

	last4 := req.AccountNumber[len(req.AccountNumber)-4:]
	id := db.NewID("bnk_")
	now := time.Now().UTC()

	var count int
	_ = s.db.QueryRow("SELECT COUNT(1) FROM bank_accounts WHERE user_id = ?", userID).Scan(&count)
	isPrimary := count == 0

	_, err := s.db.Exec(`
		INSERT INTO bank_accounts (id, user_id, bank_name, account_holder, account_last4, ifsc, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, userID, req.BankName, req.AccountHolder, last4, req.IFSC, isPrimary, now)
	if err != nil {
		return nil, err
	}

	return &BankAccount{
		ID:            id,
		UserID:        userID,
		BankName:      req.BankName,
		AccountHolder: req.AccountHolder,
		AccountLast4:  last4,
		IFSC:          req.IFSC,
		IsPrimary:     isPrimary,
		CreatedAt:     now,
	}, nil
}

func (s *Service) DeleteBankAccount(userID, bankAccountID string) error {
	var ownerID string
	err := s.db.QueryRow("SELECT user_id FROM bank_accounts WHERE id = ?", bankAccountID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &httpx.CustomError{Code: "bank_account_not_found", Message: "Bank account not found", Status: http.StatusNotFound}
		}
		return err
	}

	if ownerID != userID {
		return &httpx.CustomError{Code: "forbidden", Message: "You can only delete your own bank accounts", Status: http.StatusForbidden}
	}

	_, err = s.db.Exec("DELETE FROM bank_accounts WHERE id = ?", bankAccountID)
	return err
}
