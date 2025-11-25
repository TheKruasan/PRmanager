package userhandler

import (
	"encoding/json"
	"log"
	"net/http"
	"prmanager/internal/handlers/interfaces"
)

type Handler struct {
	service interfaces.Service
	logger  *log.Logger
}

func NewHandler(service interfaces.Service, logger *log.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.service.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.writeError(w, "INVALID_REQUEST", "user_id is required", http.StatusBadRequest)
		return
	}

	prs, err := h.service.GetUserReviews(r.Context(), userID)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

func (h *Handler) writeError(w http.ResponseWriter, code, message string, status int) {
	h.logger.Printf("Users Error: %s - %s (status: %d)", code, message, status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	errorResp.Error.Code = code
	errorResp.Error.Message = message

	json.NewEncoder(w).Encode(errorResp)
}
