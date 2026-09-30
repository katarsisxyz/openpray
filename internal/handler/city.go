package handler

import (
	"context"
	"net/http"

	"github.com/katarsisxyz/openpray/internal/model"
)

// CityService is what the city handler needs from the service layer.
type CityService interface {
	Search(ctx context.Context, query string, limit int) ([]model.City, error)
	Get(ctx context.Context, id string) (model.City, error)
}

// CityHandler serves the city lookup endpoints.
type CityHandler struct {
	svc CityService
}

// NewCityHandler returns a CityHandler.
func NewCityHandler(svc CityService) *CityHandler {
	return &CityHandler{svc: svc}
}

// Search handles GET /v1/cities?[q=][&limit=].
func (h *CityHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 20
	if q.Has("limit") {
		var err error
		if limit, err = parseInt(q, "limit", 1, 1000); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	cities, err := h.svc.Search(r.Context(), q.Get("q"), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]cityResponse, len(cities))
	for i, c := range cities {
		out[i] = toCity(c)
	}
	writeJSON(w, http.StatusOK, map[string]any{"cities": out})
}

// Get handles GET /v1/cities/{id}.
func (h *CityHandler) Get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCity(c))
}
