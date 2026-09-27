package auth

import (
	"context"
	"net/http"

	"github.com/tony19053000/wallet-api/internal/httpx"
)

type contextKey string

const UserContextKey contextKey = "user"

func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := httpx.ExtractToken(r)
		user, err := h.svc.GetUserByToken(token)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (h *Handler) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil || user.Role != "admin" {
			httpx.Forbidden(w, "forbidden", "Admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserFromContext(ctx context.Context) *User {
	u, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return u
}
