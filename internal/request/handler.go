package request

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

func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	res, err := h.svc.CreateRequest(u.ID, req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) SplitBill(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	var req SplitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	res, err := h.svc.SplitBill(u.ID, req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"requests": res,
		"count":    len(res),
	})
}

func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	direction := r.URL.Query().Get("direction")
	requests, err := h.svc.ListRequests(u.ID, direction)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"requests": requests,
		"count":    len(requests),
	})
}

func (h *Handler) PayRequest(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	id := r.PathValue("id")
	res, err := h.svc.PayRequest(id, u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) DeclineRequest(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	id := r.PathValue("id")
	res, err := h.svc.DeclineRequest(id, u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) CancelRequest(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	id := r.PathValue("id")
	res, err := h.svc.CancelRequest(id, u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}
