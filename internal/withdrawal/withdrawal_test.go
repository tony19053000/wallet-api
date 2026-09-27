package withdrawal_test

import (
	"database/sql"
	"testing"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/bank"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/wallet"
	"github.com/tony19053000/wallet-api/internal/withdrawal"
)

func setupTest(t *testing.T) (*sql.DB, *auth.Service, *wallet.Service, *bank.Service, *withdrawal.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authSvc := auth.NewService(database)
	walletSvc := wallet.NewService(database)
	bankSvc := bank.NewService(database)
	wdrSvc := withdrawal.NewService(database)
	return database, authSvc, walletSvc, bankSvc, wdrSvc
}

func TestWithdrawalSuccess(t *testing.T) {
	_, authSvc, walletSvc, bankSvc, wdrSvc := setupTest(t)

	u, _ := authSvc.Signup(auth.SignupRequest{Name: "Wdr User", Email: "wdr@pocketa.money", Password: "p", Handle: "@wdr"})
	walletSvc.TopUp(u.User.ID, 200000, "upi") // ₹2,000 balance

	acc, _ := bankSvc.AddBankAccount(u.User.ID, bank.CreateBankAccountRequest{
		BankName:      "ICICI Bank",
		AccountHolder: "Wdr User",
		AccountNumber: "000401583920",
		IFSC:          "ICIC0000004",
	})

	// Withdraw ₹500 = 50000 paise
	wdr, err := wdrSvc.Withdraw(u.User.ID, withdrawal.CreateWithdrawalRequest{
		BankAccountID: acc.ID,
		AmountPaise:   50000,
	})
	if err != nil {
		t.Fatalf("Withdraw failed: %v", err)
	}

	if wdr.Status != "completed" || wdr.FeePaise != 200 {
		t.Errorf("Unexpected withdrawal data: %+v", wdr)
	}

	// Balance after ₹500 withdrawal + ₹2 fee: ₹2,000 - ₹502 = ₹1,498 (149800 paise)
	wRes, _ := walletSvc.GetWalletResponse(u.User.ID)
	if wRes.BalancePaise != 149800 {
		t.Errorf("Expected balance 149800, got %d", wRes.BalancePaise)
	}
}
