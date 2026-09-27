package admin

import (
	"net/http"

	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListWallets(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	wallets, err := h.svc.ListWallets(status)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"wallets": wallets,
		"count":   len(wallets),
	})
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := h.svc.GetWalletDetail(id)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) FreezeWallet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.SetWalletStatus(id, "frozen"); err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Wallet frozen successfully"})
}

func (h *Handler) UnfreezeWallet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.SetWalletStatus(id, "active"); err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Wallet unfrozen successfully"})
}

func (h *Handler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	transfers, err := h.svc.ListTransfers(status)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"transfers": transfers,
		"count":     len(transfers),
	})
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetMetrics()
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, metrics)
}

func (h *Handler) CheckLedger(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.CheckLedger()
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}
