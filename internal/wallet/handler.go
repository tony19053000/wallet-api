package wallet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	walletRes, err := h.svc.GetWalletResponse(u.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, walletRes)
}

func (h *Handler) TopUp(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	var req TopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	res, err := h.svc.TopUp(u.ID, req.AmountPaise, req.Source)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))

	filter := TransactionFilter{
		Type:   query.Get("type"),
		From:   query.Get("from"),
		To:     query.Get("to"),
		Limit:  limit,
		Cursor: query.Get("cursor"),
	}

	txns, err := h.svc.GetTransactions(u.ID, filter)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"transactions": txns,
		"count":        len(txns),
	})
}

func (h *Handler) GetStatementCSV(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		httpx.Unauthorized(w, "unauthorized", "Authentication required")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	csvData, err := h.svc.GenerateStatementCSV(u.ID, from, to)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	filename := fmt.Sprintf("pocketa-statement-%s.csv", u.Handle)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvData)
}
