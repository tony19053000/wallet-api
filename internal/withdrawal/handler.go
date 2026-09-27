package withdrawal

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

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	var req CreateWithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	wdr, err := h.svc.Withdraw(u.ID, req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, wdr)
}

func (h *Handler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	withdrawals, err := h.svc.ListWithdrawals(u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"withdrawals": withdrawals,
		"count":       len(withdrawals),
	})
}
