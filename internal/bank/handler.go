package bank

import (
	"encoding/json"
	"net/http"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListBankAccounts(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	accounts, err := h.svc.ListBankAccounts(u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"bank_accounts": accounts,
		"count":         len(accounts),
	})
}

func (h *Handler) AddBankAccount(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	var req CreateBankAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	account, err := h.svc.AddBankAccount(u.ID, req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, account)
}

func (h *Handler) DeleteBankAccount(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	id := r.PathValue("id")
	if err := h.svc.DeleteBankAccount(u.ID, id); err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Bank account deleted"})
}
