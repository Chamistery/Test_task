package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Chamistery/Test_task/internal/models"
)

func (h *Handlers) HandlePullRequestCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req models.CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	exists, err := h.storage.PRExists(req.PullRequestID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if exists {
		h.respondError(w, http.StatusConflict, models.ErrPRExists, "PR id already exists")
		return
	}

	author, err := h.storage.GetUser(req.AuthorID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if author == nil {
		h.respondError(w, http.StatusNotFound, models.ErrNotFound, "author not found")
		return
	}

	reviewers, err := h.service.AssignReviewers(req.AuthorID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	pr := &models.PullRequest{
		PullRequestID:     req.PullRequestID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          req.AuthorID,
		AssignedReviewers: reviewers,
	}

	if err := h.storage.CreatePullRequest(pr); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	createdPR, err := h.storage.GetPullRequest(pr.PullRequestID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"pr": createdPR,
	})
}

func (h *Handlers) HandlePullRequestMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req models.MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	pr, err := h.storage.MergePullRequest(req.PullRequestID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if pr == nil {
		h.respondError(w, http.StatusNotFound, models.ErrNotFound, "pull request not found")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handlers) HandlePullRequestReassign(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req models.ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	pr, err := h.storage.GetPullRequest(req.PullRequestID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if pr == nil {
		h.respondError(w, http.StatusNotFound, models.ErrNotFound, "pull request not found")
		return
	}

	if pr.Status == "MERGED" {
		h.respondError(w, http.StatusConflict, models.ErrPRMerged, "cannot reassign on merged PR")
		return
	}

	isAssigned, err := h.storage.IsReviewerAssigned(req.PullRequestID, req.OldUserID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !isAssigned {
		h.respondError(w, http.StatusConflict, models.ErrNotAssigned, "reviewer is not assigned to this PR")
		return
	}

	newReviewerID, err := h.service.FindReplacementReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if newReviewerID == "" {
		h.respondError(w, http.StatusConflict, models.ErrNoCandidate, "no active replacement candidate in team")
		return
	}

	if err := h.storage.ReassignReviewer(req.PullRequestID, req.OldUserID, newReviewerID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	updatedPR, err := h.storage.GetPullRequest(req.PullRequestID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response := models.ReassignResponse{
		PR:         *updatedPR,
		ReplacedBy: newReviewerID,
	}

	h.respondJSON(w, http.StatusOK, response)
}
