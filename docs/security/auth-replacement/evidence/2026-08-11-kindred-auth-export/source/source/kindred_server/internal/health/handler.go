package health

import (
	"net/http"

	"kindred_server/internal/platform/router"
	"kindred_server/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func RegisterRoutes(r *router.Router, h *Handler) {
	r.Handle(http.MethodGet, "/health", http.HandlerFunc(h.Check))
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, h.service.Check())
}
