package transfer_test

import (
	"database/sql"
	"testing"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/transfer"
	"github.com/tony19053000/wallet-api/internal/wallet"
)

func setupTest(t *testing.T) (*sql.DB, *auth.Service, *wallet.Service, *transfer.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authSvc := auth.NewService(database)
	walletSvc := wallet.NewService(database)
	transferSvc := transfer.NewService(database)
	return database, authSvc, walletSvc, transferSvc
}

func TestNormalTransferSuccess(t *testing.T) {
	_, authSvc, walletSvc, transferSvc := setupTest(t)

	// Create Sender
	u1, _ := authSvc.Signup(auth.SignupRequest{
		Name:     "Riya",
		Email:    "riya@pocketa.money",
		Password: "password",
		Handle:   "@riya.s",
	})
	walletSvc.TopUp(u1.User.ID, 500000, "upi") // ₹5,000

	// Create Receiver
	u2, _ := authSvc.Signup(auth.SignupRequest{
		Name:     "Arjun",
		Email:    "arjun@pocketa.money",
		Password: "password",
		Handle:   "@arjun.m",
	})

	// Test Lookup User
	lu, err := transferSvc.LookupUserByHandle("@arjun.m")
	if err != nil || lu.UserID != u2.User.ID {
		t.Fatalf("LookupUserByHandle failed: %v", err)
	}

	// Perform Transfer ₹1,500 = 150000 paise
	tr, err := transferSvc.Transfer(u1.User.ID, transfer.CreateTransferRequest{
		ToHandle:    "@arjun.m",
		AmountPaise: 150000,
		Note:        "Lunch share",
	})
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	if tr.Status != "completed" {
		t.Errorf("Expected status completed, got %s", tr.Status)
	}

	// Verify Sender Balance (₹5,000 - ₹1,500 = ₹3,500)
	w1, _ := walletSvc.GetWalletResponse(u1.User.ID)
	if w1.BalancePaise != 350000 {
		t.Errorf("Expected sender balance 350000, got %d", w1.BalancePaise)
	}

	// Verify Receiver Balance (₹1,500)
	w2, _ := walletSvc.GetWalletResponse(u2.User.ID)
	if w2.BalancePaise != 150000 {
		t.Errorf("Expected receiver balance 150000, got %d", w2.BalancePaise)
	}
}

func TestTransferInsufficientFunds(t *testing.T) {
	_, authSvc, walletSvc, transferSvc := setupTest(t)

	u1, _ := authSvc.Signup(auth.SignupRequest{Name: "U1", Email: "u1@pocketa.money", Password: "p", Handle: "@u1"})
	walletSvc.TopUp(u1.User.ID, 50000, "upi") // ₹500 balance

	authSvc.Signup(auth.SignupRequest{Name: "U2", Email: "u2@pocketa.money", Password: "p", Handle: "@u2"})

	// Attempt transfer ₹1,000 (100000 paise) > ₹500 balance
	_, err := transferSvc.Transfer(u1.User.ID, transfer.CreateTransferRequest{
		ToHandle:    "@u2",
		AmountPaise: 100000,
	})
	if err == nil {
		t.Error("Expected insufficient funds error")
	}
}

func TestTransferFrozenSenderRejected(t *testing.T) {
	database, authSvc, walletSvc, transferSvc := setupTest(t)

	u1, _ := authSvc.Signup(auth.SignupRequest{Name: "U1", Email: "u1@pocketa.money", Password: "p", Handle: "@u1"})
	walletSvc.TopUp(u1.User.ID, 500000, "upi")

	_, _ = authSvc.Signup(auth.SignupRequest{Name: "U2", Email: "u2@pocketa.money", Password: "p", Handle: "@u2"})

	// Freeze sender wallet
	database.Exec("UPDATE wallets SET status = 'frozen' WHERE user_id = ?", u1.User.ID)

	_, err := transferSvc.Transfer(u1.User.ID, transfer.CreateTransferRequest{
		ToHandle:    "@u2",
		AmountPaise: 100000,
	})
	if err == nil {
		t.Error("Expected frozen wallet error")
	}
}
