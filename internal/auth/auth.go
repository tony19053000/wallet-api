package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Handle       string    `json:"handle"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	KYCLevel     string    `json:"kyc_level"`
	CreatedAt    time.Time `json:"created_at"`
}

type Wallet struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	BalancePaise        int64     `json:"balance_paise"`
	Status              string    `json:"status"`
	DailySendLimitPaise int64     `json:"daily_send_limit_paise"`
	CreatedAt           time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	db *sql.DB
}

func NewService(database *sql.DB) *Service {
	return &Service{db: database}
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Handle   string `json:"handle"`
}

type SignupResponse struct {
	User   *User   `json:"user"`
	Wallet *Wallet `json:"wallet"`
	Token  string  `json:"token"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

type MeResponse struct {
	User   *User   `json:"user"`
	Wallet *Wallet `json:"wallet"`
}

func FormatHandle(h string) string {
	h = strings.TrimSpace(h)
	if !strings.HasPrefix(h, "@") {
		h = "@" + h
	}
	return strings.ToLower(h)
}

func (s *Service) Signup(req SignupRequest) (*SignupResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	req.Handle = FormatHandle(req.Handle)

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("name, email, and password are required")
	}

	if req.Handle == "@" || req.Handle == "" {
		// generate handle from name/email
		parts := strings.Split(req.Email, "@")
		req.Handle = "@" + parts[0]
	}

	// Check existing email
	var exists int
	err := s.db.QueryRow("SELECT COUNT(1) FROM users WHERE email = ?", req.Email).Scan(&exists)
	if err == nil && exists > 0 {
		return nil, &httpx.CustomError{Code: "email_taken", Message: "Email address is already registered", Status: http.StatusConflict}
	}

	// Check existing handle
	err = s.db.QueryRow("SELECT COUNT(1) FROM users WHERE handle = ?", req.Handle).Scan(&exists)
	if err == nil && exists > 0 {
		return nil, &httpx.CustomError{Code: "handle_taken", Message: "Handle is already taken", Status: http.StatusConflict}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := db.NewID("usr_")
	walletID := db.NewID("wal_")
	now := time.Now().UTC()

	// Default limit: basic ₹10,000 = 1,000,000 paise
	dailyLimit := int64(1000000)
	kycLevel := "basic"

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO users (id, name, email, phone, handle, password_hash, role, kyc_level, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 'user', ?, ?)
	`, userID, req.Name, req.Email, req.Phone, req.Handle, string(hash), kycLevel, now)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		INSERT INTO wallets (id, user_id, balance_paise, status, daily_send_limit_paise, created_at)
		VALUES (?, ?, 0, 'active', ?, ?)
	`, walletID, userID, dailyLimit, now)
	if err != nil {
		return nil, err
	}

	sessionID := db.NewID("ses_")
	token := db.NewToken()
	expiresAt := now.Add(30 * 24 * time.Hour)

	_, err = tx.Exec(`
		INSERT INTO sessions (id, token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, token, userID, expiresAt, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user := &User{
		ID:        userID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Handle:    req.Handle,
		Role:      "user",
		KYCLevel:  kycLevel,
		CreatedAt: now,
	}

	wallet := &Wallet{
		ID:                  walletID,
		UserID:              userID,
		BalancePaise:        0,
		Status:              "active",
		DailySendLimitPaise: dailyLimit,
		CreatedAt:           now,
	}

	return &SignupResponse{
		User:   user,
		Wallet: wallet,
		Token:  token,
	}, nil
}

func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var u User
	err := s.db.QueryRow(`
		SELECT id, name, email, phone, handle, password_hash, role, kyc_level, created_at
		FROM users WHERE email = ?
	`, req.Email).Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Handle, &u.PasswordHash, &u.Role, &u.KYCLevel, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "invalid_credentials", Message: "Invalid email or password", Status: http.StatusUnauthorized}
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, &httpx.CustomError{Code: "invalid_credentials", Message: "Invalid email or password", Status: http.StatusUnauthorized}
	}

	sessionID := db.NewID("ses_")
	token := db.NewToken()
	now := time.Now().UTC()
	expiresAt := now.Add(30 * 24 * time.Hour)

	_, err = s.db.Exec(`
		INSERT INTO sessions (id, token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, token, u.ID, expiresAt, now)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		User:  &u,
		Token: token,
	}, nil
}

func (s *Service) Logout(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *Service) GetUserByToken(token string) (*User, error) {
	if token == "" {
		return nil, &httpx.CustomError{Code: "unauthorized", Message: "Authentication token required", Status: http.StatusUnauthorized}
	}

	var u User
	var expiresAt time.Time
	err := s.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.phone, u.handle, u.role, u.kyc_level, u.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token = ?
	`, token).Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Handle, &u.Role, &u.KYCLevel, &u.CreatedAt, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "unauthorized", Message: "Invalid or expired token", Status: http.StatusUnauthorized}
		}
		return nil, err
	}

	if time.Now().After(expiresAt) {
		s.Logout(token)
		return nil, &httpx.CustomError{Code: "unauthorized", Message: "Session expired", Status: http.StatusUnauthorized}
	}

	return &u, nil
}

func (s *Service) GetWalletByUserID(userID string) (*Wallet, error) {
	var w Wallet
	err := s.db.QueryRow(`
		SELECT id, user_id, balance_paise, status, daily_send_limit_paise, created_at
		FROM wallets WHERE user_id = ?
	`, userID).Scan(&w.ID, &w.UserID, &w.BalancePaise, &w.Status, &w.DailySendLimitPaise, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}
