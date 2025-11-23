package handlers

import (
	"net/http"
)

func (h *Handlers) HandleStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		h.respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	stats, err := h.storage.GetStatistics()
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, stats)
}
