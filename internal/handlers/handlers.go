package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Chamistery/Test_task/internal/models"
	"github.com/Chamistery/Test_task/internal/service"
	"github.com/Chamistery/Test_task/internal/storage"
)

type Handlers struct {
	storage storage.Storage
	service *service.ReviewerService
}

func NewHandlers(storage storage.Storage) *Handlers {
	return &Handlers{
		storage: storage,
		service: service.NewReviewerService(storage),
	}
}

func (h *Handlers) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handlers) respondError(w http.ResponseWriter, status int, code, message string) {
	h.respondJSON(w, status, models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
