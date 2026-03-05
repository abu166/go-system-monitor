package api

import (
	"encoding/json"
	"net/http"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"success":false,"error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, model.ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func writeSuccess[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, model.SuccessResponse[T]{
		Success: true,
		Data:    data,
	})
}
