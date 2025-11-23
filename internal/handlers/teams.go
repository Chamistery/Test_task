package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Chamistery/Test_task/internal/models"
)

func (h *Handlers) HandleTeamAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	exists, err := h.storage.TeamExists(team.TeamName)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if exists {
		h.respondError(w, http.StatusBadRequest, models.ErrTeamExists, "team_name already exists")
		return
	}

	if err := h.storage.CreateTeam(&team); err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	createdTeam, err := h.storage.GetTeam(team.TeamName)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"team": createdTeam,
	})
}

func (h *Handlers) HandleTeamGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name query parameter required")
		return
	}

	team, err := h.storage.GetTeam(teamName)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if team == nil {
		h.respondError(w, http.StatusNotFound, models.ErrNotFound, "team not found")
		return
	}

	h.respondJSON(w, http.StatusOK, team)
}

func (h *Handlers) HandleTeamDeactivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req models.BulkDeactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	start := time.Now()
	deactivated, reassigned, err := h.storage.BulkDeactivateTeamMembers(req.TeamName)
	duration := time.Since(start)

	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response := models.BulkDeactivateResponse{
		DeactivatedUsers: deactivated,
		ReassignedPRs:    reassigned,
		Duration:         duration.String(),
	}

	h.respondJSON(w, http.StatusOK, response)
}
