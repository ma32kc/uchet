package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/k1ra/uchet/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal"
	msg := "internal server error"

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		msg = err.Error()
	case errors.Is(err, domain.ErrValidation):
		status = http.StatusBadRequest
		code = "validation_error"
		msg = err.Error()
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "forbidden"
		msg = err.Error()
	default:
		slog.Error("request failed", "err", err)
	}

	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: msg}})
}
