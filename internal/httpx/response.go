package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, `{"error":{"code":"internal_error","message":"failed to encode response"}}`, http.StatusInternalServerError)
		}
	}
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func BadRequest(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusBadRequest, code, message)
}

func Unauthorized(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusUnauthorized, code, message)
}

func Forbidden(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusForbidden, code, message)
}

func NotFound(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusNotFound, code, message)
}

func Conflict(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusConflict, code, message)
}

func UnprocessableEntity(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusUnprocessableEntity, code, message)
}

func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, "internal_error", message)
}

// FormatPaise formats paise int64 to Indian currency format, e.g. 50000 -> ₹500.00, 12500000 -> ₹1,25,000.00
func FormatPaise(paise int64) string {
	negative := false
	if paise < 0 {
		negative = true
		paise = -paise
	}

	rupees := paise / 100
	remPaise := paise % 100

	rStr := strconv.FormatInt(rupees, 10)
	var formattedRupees string
	if len(rStr) <= 3 {
		formattedRupees = rStr
	} else {
		last3 := rStr[len(rStr)-3:]
		rest := rStr[:len(rStr)-3]
		var parts []string
		for len(rest) > 2 {
			parts = append([]string{rest[len(rest)-2:]}, parts...)
			rest = rest[:len(rest)-2]
		}
		if len(rest) > 0 {
			parts = append([]string{rest}, parts...)
		}
		formattedRupees = strings.Join(parts, ",") + "," + last3
	}

	res := fmt.Sprintf("₹%s.%02d", formattedRupees, remPaise)
	if negative {
		return "-" + res
	}
	return res
}
