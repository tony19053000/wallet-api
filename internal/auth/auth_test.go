package auth_test

import (
	"testing"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
)

func setupTestDB(t *testing.T) *auth.Service {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return auth.NewService(database)
}

func TestSignupAndLoginSuccess(t *testing.T) {
	svc := setupTestDB(t)

	// Test Signup
	res, err := svc.Signup(auth.SignupRequest{
		Name:     "Kavya Shah",
		Email:    "kavya@pocketa.money",
		Phone:    "+91 98765 43210",
		Password: "password123",
		Handle:   "@kavya.s",
	})
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	if res.User.Name != "Kavya Shah" || res.User.Handle != "@kavya.s" {
		t.Errorf("Unexpected user data: %+v", res.User)
	}
	if res.Wallet.BalancePaise != 0 || res.Wallet.Status != "active" {
		t.Errorf("Unexpected wallet data: %+v", res.Wallet)
	}
	if res.Token == "" {
		t.Error("Expected valid bearer token")
	}

	// Test GetUserByToken
	u, err := svc.GetUserByToken(res.Token)
	if err != nil || u.ID != res.User.ID {
		t.Errorf("GetUserByToken failed: %v", err)
	}

	// Test Login
	lRes, err := svc.Login(auth.LoginRequest{
		Email:    "kavya@pocketa.money",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if lRes.User.ID != res.User.ID || lRes.Token == "" {
		t.Errorf("Unexpected login response: %+v", lRes)
	}

	// Test Logout
	if err := svc.Logout(lRes.Token); err != nil {
		t.Errorf("Logout failed: %v", err)
	}

	_, err = svc.GetUserByToken(lRes.Token)
	if err == nil {
		t.Error("Expected error after logging out")
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	svc := setupTestDB(t)

	_, err := svc.Signup(auth.SignupRequest{
		Name:     "User One",
		Email:    "dup@pocketa.money",
		Password: "pass",
		Handle:   "@user1",
	})
	if err != nil {
		t.Fatalf("Initial signup failed: %v", err)
	}

	_, err = svc.Signup(auth.SignupRequest{
		Name:     "User Two",
		Email:    "dup@pocketa.money",
		Password: "pass",
		Handle:   "@user2",
	})
	if err == nil {
		t.Error("Expected error on duplicate email signup")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	svc := setupTestDB(t)

	_, err := svc.Signup(auth.SignupRequest{
		Name:     "User Three",
		Email:    "user3@pocketa.money",
		Password: "correctpassword",
		Handle:   "@user3",
	})
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	_, err = svc.Login(auth.LoginRequest{
		Email:    "user3@pocketa.money",
		Password: "wrongpassword",
	})
	if err == nil {
		t.Error("Expected error for wrong password")
	}
}
