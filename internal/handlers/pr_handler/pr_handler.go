package prhandler

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

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.service.CreatePullRequest(r.Context(), req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		switch err.Error() {
		case "PR_EXISTS":
			h.writeError(w, "PR_EXISTS", "PR id already exists", http.StatusConflict)
		case "AUTHOR_NOT_FOUND", "TEAM_NOT_FOUND":
			h.writeError(w, "NOT_FOUND", "Author or team not found", http.StatusNotFound)
		default:
			h.writeError(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	pr, err := h.service.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "PR not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		switch err.Error() {
		case "PR_MERGED":
			h.writeError(w, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
		case "NOT_ASSIGNED":
			h.writeError(w, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
		case "NO_CANDIDATE":
			h.writeError(w, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
		case "NOT_FOUND":
			h.writeError(w, "NOT_FOUND", "PR or user not found", http.StatusNotFound)
		default:
			h.writeError(w, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pr":          result.PR,
		"replaced_by": result.NewReviewerID,
	})
}

func (h *Handler) writeError(w http.ResponseWriter, code, message string, status int) {
	h.logger.Printf("PullRequests Error: %s - %s (status: %d)", code, message, status)

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
