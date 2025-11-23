package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Chamistery/Test_task/internal/models"
)

func (h *Handlers) HandleUserSetIsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req models.SetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if err := h.storage.SetUserIsActive(req.UserID, req.IsActive); err != nil {
		h.respondError(w, http.StatusNotFound, models.ErrNotFound, "user not found")
		return
	}

	user, err := h.storage.GetUser(req.UserID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

func (h *Handlers) HandleUsersGetReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id query parameter required")
		return
	}

	prs, err := h.storage.GetPRsByReviewer(userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response := models.UserReviewsResponse{
		UserID:       userID,
		PullRequests: prs,
	}

	h.respondJSON(w, http.StatusOK, response)
}
