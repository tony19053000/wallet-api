package wallet_test

import (
	"database/sql"
	"testing"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/wallet"
)

func setupTestDB(t *testing.T) (*sql.DB, *auth.Service, *wallet.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authSvc := auth.NewService(database)
	walletSvc := wallet.NewService(database)
	return database, authSvc, walletSvc
}

func TestTopUpAndTransactions(t *testing.T) {
	_, authSvc, walletSvc := setupTestDB(t)

	sRes, err := authSvc.Signup(auth.SignupRequest{
		Name:     "Amit Patel",
		Email:    "amit@pocketa.money",
		Password: "password",
		Handle:   "@amit.p",
	})
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Test Top Up ₹1,000 = 100000 paise
	topUpRes, err := walletSvc.TopUp(sRes.User.ID, 100000, "upi")
	if err != nil {
		t.Fatalf("TopUp failed: %v", err)
	}

	if topUpRes.Wallet.BalancePaise != 100000 {
		t.Errorf("Expected balance 100000, got %d", topUpRes.Wallet.BalancePaise)
	}
	if topUpRes.Transaction.Type != "top_up" || topUpRes.Transaction.AmountPaise != 100000 {
		t.Errorf("Unexpected transaction: %+v", topUpRes.Transaction)
	}

	// Test Get Transactions
	txns, err := walletSvc.GetTransactions(sRes.User.ID, wallet.TransactionFilter{})
	if err != nil {
		t.Fatalf("GetTransactions failed: %v", err)
	}
	if len(txns) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(txns))
	}

	// Test Generate Statement CSV
	csvBytes, err := walletSvc.GenerateStatementCSV(sRes.User.ID, "", "")
	if err != nil || len(csvBytes) == 0 {
		t.Fatalf("GenerateStatementCSV failed or empty: %v", err)
	}
}

func TestTopUpInvalidAmount(t *testing.T) {
	_, authSvc, walletSvc := setupTestDB(t)

	sRes, _ := authSvc.Signup(auth.SignupRequest{
		Name:     "Test User",
		Email:    "test@pocketa.money",
		Password: "password",
		Handle:   "@test",
	})

	// Less than min ₹1.00 (100 paise)
	_, err := walletSvc.TopUp(sRes.User.ID, 50, "upi")
	if err == nil {
		t.Error("Expected error for amount < 100 paise")
	}

	// Greater than max ₹50,000 (5,000,000 paise)
	_, err = walletSvc.TopUp(sRes.User.ID, 6000000, "upi")
	if err == nil {
		t.Error("Expected error for amount > 50,000 rupees")
	}
}
