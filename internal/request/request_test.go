package request_test

import (
	"database/sql"
	"testing"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/request"
	"github.com/tony19053000/wallet-api/internal/transfer"
	"github.com/tony19053000/wallet-api/internal/wallet"
)

func setupTest(t *testing.T) (*sql.DB, *auth.Service, *wallet.Service, *transfer.Service, *request.Service) {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	authSvc := auth.NewService(database)
	walletSvc := wallet.NewService(database)
	transferSvc := transfer.NewService(database)
	requestSvc := request.NewService(database, transferSvc)
	return database, authSvc, walletSvc, transferSvc, requestSvc
}

func TestRequestAndPayFlow(t *testing.T) {
	_, authSvc, walletSvc, _, requestSvc := setupTest(t)

	// User 1 requests money from User 2
	u1, _ := authSvc.Signup(auth.SignupRequest{Name: "Requester", Email: "req@pocketa.money", Password: "p", Handle: "@req"})
	u2, _ := authSvc.Signup(auth.SignupRequest{Name: "Payer", Email: "pay@pocketa.money", Password: "p", Handle: "@pay"})

	walletSvc.TopUp(u2.User.ID, 300000, "upi") // Payer has ₹3,000 balance

	// Create Request ₹1,000 = 100000 paise
	req, err := requestSvc.CreateRequest(u1.User.ID, request.CreateRequest{
		ToHandle:    "@pay",
		AmountPaise: 100000,
		Note:        "Dinner split",
	})
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}
	if req.Status != "pending" {
		t.Errorf("Expected status pending, got %s", req.Status)
	}

	// List incoming requests for Payer
	inc, err := requestSvc.ListRequests(u2.User.ID, "incoming")
	if err != nil || len(inc) != 1 {
		t.Fatalf("ListRequests incoming failed: %v", err)
	}

	// Payer pays request
	paidReq, err := requestSvc.PayRequest(req.ID, u2.User.ID)
	if err != nil {
		t.Fatalf("PayRequest failed: %v", err)
	}
	if paidReq.Status != "paid" {
		t.Errorf("Expected status paid, got %s", paidReq.Status)
	}

	// Check updated balances
	w1, _ := walletSvc.GetWalletResponse(u1.User.ID)
	if w1.BalancePaise != 100000 {
		t.Errorf("Expected requester balance 100000, got %d", w1.BalancePaise)
	}
}

func TestDeclineAndCancelRequest(t *testing.T) {
	_, authSvc, _, _, requestSvc := setupTest(t)

	u1, _ := authSvc.Signup(auth.SignupRequest{Name: "U1", Email: "u1@pocketa.money", Password: "p", Handle: "@u1"})
	u2, _ := authSvc.Signup(auth.SignupRequest{Name: "U2", Email: "u2@pocketa.money", Password: "p", Handle: "@u2"})

	// Decline test
	req1, _ := requestSvc.CreateRequest(u1.User.ID, request.CreateRequest{ToHandle: "@u2", AmountPaise: 50000, Note: "test"})
	declReq, err := requestSvc.DeclineRequest(req1.ID, u2.User.ID)
	if err != nil || declReq.Status != "declined" {
		t.Errorf("DeclineRequest failed: %v", err)
	}

	// Cancel test
	req2, _ := requestSvc.CreateRequest(u1.User.ID, request.CreateRequest{ToHandle: "@u2", AmountPaise: 50000, Note: "test2"})
	cancReq, err := requestSvc.CancelRequest(req2.ID, u1.User.ID)
	if err != nil || cancReq.Status != "cancelled" {
		t.Errorf("CancelRequest failed: %v", err)
	}
}

func TestSplitBillCalculation(t *testing.T) {
	_, authSvc, _, _, requestSvc := setupTest(t)

	u1, _ := authSvc.Signup(auth.SignupRequest{Name: "Host", Email: "host@pocketa.money", Password: "p", Handle: "@host"})
	authSvc.Signup(auth.SignupRequest{Name: "Friend 1", Email: "f1@pocketa.money", Password: "p", Handle: "@f1"})
	authSvc.Signup(auth.SignupRequest{Name: "Friend 2", Email: "f2@pocketa.money", Password: "p", Handle: "@f2"})

	// Split total ₹1,000 = 100000 paise among 3 people
	// 100000 / 3 = 33333 with remainder 1 -> First handle gets 33334, second gets 33333
	reqs, err := requestSvc.SplitBill(u1.User.ID, request.SplitRequest{
		Handles:    []string{"@f1", "@f2"},
		TotalPaise: 100000,
		Note:       "Pizza party",
	})
	if err != nil {
		t.Fatalf("SplitBill failed: %v", err)
	}
	if len(reqs) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(reqs))
	}
	if reqs[0].AmountPaise != 50000 || reqs[1].AmountPaise != 50000 {
		t.Errorf("Unexpected split shares: %d, %d", reqs[0].AmountPaise, reqs[1].AmountPaise)
	}
}
