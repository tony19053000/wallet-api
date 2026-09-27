package bank_test

import (
	"database/sql"
	"testing"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/bank"
	"github.com/tony19053000/wallet-api/internal/db"
)

func setupTest(t *testing.T) (*sql.DB, *auth.Service, *bank.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authSvc := auth.NewService(database)
	bankSvc := bank.NewService(database)
	return database, authSvc, bankSvc
}

func TestAddAndListBankAccounts(t *testing.T) {
	_, authSvc, bankSvc := setupTest(t)

	u, _ := authSvc.Signup(auth.SignupRequest{Name: "User", Email: "u@pocketa.money", Password: "p", Handle: "@u"})

	acc, err := bankSvc.AddBankAccount(u.User.ID, bank.CreateBankAccountRequest{
		BankName:      "HDFC Bank",
		AccountHolder: "User Name",
		AccountNumber: "501004821948",
		IFSC:          "HDFC0000240",
	})
	if err != nil {
		t.Fatalf("AddBankAccount failed: %v", err)
	}

	if acc.AccountLast4 != "1948" || !acc.IsPrimary {
		t.Errorf("Unexpected bank account data: %+v", acc)
	}

	list, err := bankSvc.ListBankAccounts(u.User.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListBankAccounts failed: %v", err)
	}

	err = bankSvc.DeleteBankAccount(u.User.ID, acc.ID)
	if err != nil {
		t.Fatalf("DeleteBankAccount failed: %v", err)
	}

	listAfter, _ := bankSvc.ListBankAccounts(u.User.ID)
	if len(listAfter) != 0 {
		t.Errorf("Expected 0 bank accounts after delete, got %d", len(listAfter))
	}
}
