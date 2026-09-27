package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tony19053000/wallet-api/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	res, err := h.svc.Signup(req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "pocketa_session",
		Value:    res.Token,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.BadRequest(w, "invalid_json", "Invalid request body")
		return
	}

	res, err := h.svc.Login(req)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "pocketa_session",
		Value:    res.Token,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	if token != "" {
		_ = h.svc.Logout(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "pocketa_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	user, err := h.svc.GetUserByToken(token)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	wallet, err := h.svc.GetWalletByUserID(user.ID)
	if err != nil {
		httpx.HandleError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, MeResponse{
		User:   user,
		Wallet: wallet,
	})
}
