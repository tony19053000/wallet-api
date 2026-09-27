package admin_test

import (
	"database/sql"
	"testing"

	"github.com/tony19053000/wallet-api/internal/admin"
	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/wallet"
)

func setupTest(t *testing.T) (*sql.DB, *auth.Service, *wallet.Service, *admin.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authSvc := auth.NewService(database)
	walletSvc := wallet.NewService(database)
	adminSvc := admin.NewService(database)
	return database, authSvc, walletSvc, adminSvc
}

func TestAdminMetricsAndFreeze(t *testing.T) {
	_, authSvc, walletSvc, adminSvc := setupTest(t)

	u, _ := authSvc.Signup(auth.SignupRequest{Name: "User Admin Test", Email: "uat@pocketa.money", Password: "p", Handle: "@uat"})
	walletSvc.TopUp(u.User.ID, 100000, "upi")

	wallets, err := adminSvc.ListWallets("")
	if err != nil || len(wallets) != 1 {
		t.Fatalf("ListWallets failed: %v", err)
	}

	wID := wallets[0].ID

	// Test Freeze
	if err := adminSvc.SetWalletStatus(wID, "frozen"); err != nil {
		t.Fatalf("Freeze failed: %v", err)
	}

	detail, err := adminSvc.GetWalletDetail(wID)
	if err != nil || detail.Wallet.Status != "frozen" {
		t.Errorf("GetWalletDetail expected status frozen, got %v", detail)
	}

	// Test Unfreeze
	if err := adminSvc.SetWalletStatus(wID, "active"); err != nil {
		t.Fatalf("Unfreeze failed: %v", err)
	}

	// Test Metrics
	metrics, err := adminSvc.GetMetrics()
	if err != nil || metrics.TotalBalancesHeldPaise != 100000 {
		t.Errorf("GetMetrics failed: %+v", metrics)
	}

	// Test Ledger Check
	chk, err := adminSvc.CheckLedger()
	if err != nil || !chk.Matches {
		t.Errorf("CheckLedger expected matches true, got %+v", chk)
	}
}
