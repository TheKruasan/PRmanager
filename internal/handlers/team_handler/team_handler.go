package teamhandler

import (
	"encoding/json"
	"log"
	"net/http"
	"prmanager/internal/handlers/interfaces"
	"prmanager/internal/models"
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

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req models.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TeamName == "" {
		h.writeError(w, "INVALID_REQUEST", "team_name is required", http.StatusBadRequest)
		return
	}

	team, err := h.service.CreateTeam(r.Context(), &req)
	if err != nil {
		h.writeError(w, "ERROR_CREATING_TEAM", err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"team": team,
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.writeError(w, "INVALID_REQUEST", "team_name is required", http.StatusBadRequest)
		return
	}

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

func (h *Handler) writeError(w http.ResponseWriter, code, message string, status int) {
	h.logger.Printf("Teams Error: %s - %s (status: %d)", code, message, status)

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
