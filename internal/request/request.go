package request

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
	"github.com/tony19053000/wallet-api/internal/transfer"
)

type MoneyRequest struct {
	ID             string    `json:"id"`
	FromUserID     string    `json:"from_user_id"`
	FromUserName   string    `json:"from_user_name,omitempty"`
	FromUserHandle string    `json:"from_user_handle,omitempty"`
	ToUserID       string    `json:"to_user_id"`
	ToUserName     string    `json:"to_user_name,omitempty"`
	ToUserHandle   string    `json:"to_user_handle,omitempty"`
	AmountPaise    int64     `json:"amount_paise"`
	Note           string    `json:"note"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateRequest struct {
	ToHandle    string `json:"to_handle"`
	AmountPaise int64  `json:"amount_paise"`
	Note        string `json:"note"`
}

type SplitRequest struct {
	Handles     []string `json:"handles"`
	TotalPaise  int64    `json:"total_paise"`
	Note        string   `json:"note"`
}

type Service struct {
	db          *sql.DB
	transferSvc *transfer.Service
}

func NewService(database *sql.DB, transferSvc *transfer.Service) *Service {
	return &Service{db: database, transferSvc: transferSvc}
}

func (s *Service) CreateRequest(fromUserID string, req CreateRequest) (*MoneyRequest, error) {
	if req.AmountPaise <= 0 {
		return nil, &httpx.CustomError{Code: "invalid_amount", Message: "Request amount must be greater than zero", Status: http.StatusBadRequest}
	}

	req.ToHandle = auth.FormatHandle(req.ToHandle)
	var toUser auth.User
	err := s.db.QueryRow("SELECT id, name, handle FROM users WHERE handle = ?", req.ToHandle).Scan(&toUser.ID, &toUser.Name, &toUser.Handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "user_not_found", Message: "Target user handle not found", Status: http.StatusNotFound}
		}
		return nil, err
	}

	if toUser.ID == fromUserID {
		return nil, &httpx.CustomError{Code: "invalid_recipient", Message: "Cannot request money from yourself", Status: http.StatusBadRequest}
	}

	id := db.NewID("req_")
	now := time.Now().UTC()
	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = "Payment request"
	}

	_, err = s.db.Exec(`
		INSERT INTO money_requests (id, from_user_id, to_user_id, amount_paise, note, status, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?)
	`, id, fromUserID, toUser.ID, req.AmountPaise, note, now)
	if err != nil {
		return nil, err
	}

	return s.GetRequestByID(id)
}

func (s *Service) SplitBill(fromUserID string, req SplitRequest) ([]MoneyRequest, error) {
	if req.TotalPaise <= 0 {
		return nil, &httpx.CustomError{Code: "invalid_amount", Message: "Total amount must be greater than zero", Status: http.StatusBadRequest}
	}
	if len(req.Handles) == 0 {
		return nil, &httpx.CustomError{Code: "invalid_participants", Message: "At least one participant handle is required", Status: http.StatusBadRequest}
	}

	n := int64(len(req.Handles))
	baseShare := req.TotalPaise / n
	remainder := req.TotalPaise % n

	var created []MoneyRequest
	for i, h := range req.Handles {
		share := baseShare
		if i == 0 {
			share += remainder
		}
		cr, err := s.CreateRequest(fromUserID, CreateRequest{
			ToHandle:    h,
			AmountPaise: share,
			Note:        req.Note,
		})
		if err != nil {
			return nil, err
		}
		created = append(created, *cr)
	}

	return created, nil
}

func (s *Service) ListRequests(userID string, direction string) ([]MoneyRequest, error) {
	query := `
		SELECT r.id, r.from_user_id, fu.name, fu.handle, r.to_user_id, tu.name, tu.handle,
		       r.amount_paise, r.note, r.status, r.created_at
		FROM money_requests r
		JOIN users fu ON r.from_user_id = fu.id
		JOIN users tu ON r.to_user_id = tu.id
	`
	var args []any

	if direction == "incoming" {
		query += " WHERE r.to_user_id = ?"
		args = append(args, userID)
	} else if direction == "outgoing" {
		query += " WHERE r.from_user_id = ?"
		args = append(args, userID)
	} else {
		query += " WHERE r.to_user_id = ? OR r.from_user_id = ?"
		args = append(args, userID, userID)
	}

	query += " ORDER BY r.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []MoneyRequest
	for rows.Next() {
		var mr MoneyRequest
		err := rows.Scan(
			&mr.ID, &mr.FromUserID, &mr.FromUserName, &mr.FromUserHandle,
			&mr.ToUserID, &mr.ToUserName, &mr.ToUserHandle,
			&mr.AmountPaise, &mr.Note, &mr.Status, &mr.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, mr)
	}

	if requests == nil {
		requests = []MoneyRequest{}
	}

	return requests, nil
}

func (s *Service) GetRequestByID(id string) (*MoneyRequest, error) {
	var mr MoneyRequest
	err := s.db.QueryRow(`
		SELECT r.id, r.from_user_id, fu.name, fu.handle, r.to_user_id, tu.name, tu.handle,
		       r.amount_paise, r.note, r.status, r.created_at
		FROM money_requests r
		JOIN users fu ON r.from_user_id = fu.id
		JOIN users tu ON r.to_user_id = tu.id
		WHERE r.id = ?
	`, id).Scan(
		&mr.ID, &mr.FromUserID, &mr.FromUserName, &mr.FromUserHandle,
		&mr.ToUserID, &mr.ToUserName, &mr.ToUserHandle,
		&mr.AmountPaise, &mr.Note, &mr.Status, &mr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "request_not_found", Message: "Money request not found", Status: http.StatusNotFound}
		}
		return nil, err
	}
	return &mr, nil
}

func (s *Service) PayRequest(id string, payerUserID string) (*MoneyRequest, error) {
	req, err := s.GetRequestByID(id)
	if err != nil {
		return nil, err
	}

	if req.ToUserID != payerUserID {
		return nil, &httpx.CustomError{Code: "forbidden", Message: "Only the designated payer can pay this request", Status: http.StatusForbidden}
	}

	if req.Status != "pending" {
		return nil, &httpx.CustomError{Code: "invalid_status", Message: "Only pending requests can be paid", Status: http.StatusConflict}
	}

	// Resolve requester wallet ID
	var requesterWalletID string
	err = s.db.QueryRow("SELECT id FROM wallets WHERE user_id = ?", req.FromUserID).Scan(&requesterWalletID)
	if err != nil {
		return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Requester wallet not found", Status: http.StatusNotFound}
	}

	// Perform transfer
	_, err = s.transferSvc.Transfer(payerUserID, transfer.CreateTransferRequest{
		ToWalletID:  requesterWalletID,
		AmountPaise: req.AmountPaise,
		Note:        req.Note,
	})
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec("UPDATE money_requests SET status = 'paid' WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	req.Status = "paid"
	return req, nil
}

func (s *Service) DeclineRequest(id string, payerUserID string) (*MoneyRequest, error) {
	req, err := s.GetRequestByID(id)
	if err != nil {
		return nil, err
	}

	if req.ToUserID != payerUserID {
		return nil, &httpx.CustomError{Code: "forbidden", Message: "Only the designated payer can decline this request", Status: http.StatusForbidden}
	}

	if req.Status != "pending" {
		return nil, &httpx.CustomError{Code: "invalid_status", Message: "Only pending requests can be declined", Status: http.StatusConflict}
	}

	_, err = s.db.Exec("UPDATE money_requests SET status = 'declined' WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	req.Status = "declined"
	return req, nil
}

func (s *Service) CancelRequest(id string, requesterUserID string) (*MoneyRequest, error) {
	req, err := s.GetRequestByID(id)
	if err != nil {
		return nil, err
	}

	if req.FromUserID != requesterUserID {
		return nil, &httpx.CustomError{Code: "forbidden", Message: "Only the requester can cancel this request", Status: http.StatusForbidden}
	}

	if req.Status != "pending" {
		return nil, &httpx.CustomError{Code: "invalid_status", Message: "Only pending requests can be cancelled", Status: http.StatusConflict}
	}

	_, err = s.db.Exec("UPDATE money_requests SET status = 'cancelled' WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	req.Status = "cancelled"
	return req, nil
}
