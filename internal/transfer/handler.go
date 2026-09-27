package transfer

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

func (h *Handler) LookupUser(w http.ResponseWriter, r *http.Request) {
	handle := r.URL.Query().Get("handle")
	if handle == "" {
		httpx.BadRequest(w, "missing_handle", "Query parameter 'handle' is required")
		return
	}

	res, err := h.svc.LookupUserByHandle(handle)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	var req CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	tr, err := h.svc.Transfer(u.ID, req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, tr)
}

func (h *Handler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	transfers, err := h.svc.ListTransfers(u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"transfers": transfers,
		"count":     len(transfers),
	})
}

func (h *Handler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httpx.BadRequest(w, "missing_id", "Transfer ID required")
		return
	}

	tr, err := h.svc.GetTransferByID(id, u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, tr)
}
