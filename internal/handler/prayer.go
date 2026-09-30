// Package handler translates HTTP requests into service calls.
package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/katarsisxyz/openpray/internal/model"
	"github.com/katarsisxyz/openpray/internal/service"
)

// PrayerService is what the handler needs from the service layer.
type PrayerService interface {
	Methods(ctx context.Context) ([]model.Method, error)
	Daily(ctx context.Context, q service.Query, date time.Time) (model.Schedule, error)
	Monthly(ctx context.Context, q service.Query, year int, month time.Month) (model.Schedule, error)
}

// PrayerHandler serves the prayer times endpoints.
type PrayerHandler struct {
	svc PrayerService
}

// NewPrayerHandler returns a PrayerHandler.
func NewPrayerHandler(svc PrayerService) *PrayerHandler {
	return &PrayerHandler{svc: svc}
}

// ListMethods handles GET /v1/methods.
func (h *PrayerHandler) ListMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := h.svc.Methods(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]methodResponse, len(methods))
	for i, m := range methods {
		out[i] = toMethod(m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"methods": out})
}

// Daily handles GET /v1/times?(city=|lat=&lng=&tz=[&elev=])[&date=YYYY-MM-DD][&method=].
func (h *PrayerHandler) Daily(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query, err := parseQuery(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var date time.Time // zero = today at the requested place
	if s := q.Get("date"); s != "" {
		if date, err = time.Parse(time.DateOnly, s); err != nil {
			writeError(w, http.StatusBadRequest, "date must be in YYYY-MM-DD format")
			return
		}
	}

	s, err := h.svc.Daily(r.Context(), query, date)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dailyResponse{
		Location:    toLocation(s.Place),
		Method:      toMethodRef(s.Method),
		dayResponse: toDay(s.Days[0]),
	})
}

// Monthly handles GET /v1/times/month?(city=|lat=&lng=&tz=[&elev=])&year=&month=[&method=].
func (h *PrayerHandler) Monthly(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query, err := parseQuery(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	year, err := parseInt(q, "year", 1900, 2200)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	month, err := parseInt(q, "month", 1, 12)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s, err := h.svc.Monthly(r.Context(), query, year, time.Month(month))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	days := make([]dayResponse, len(s.Days))
	for i, d := range s.Days {
		days[i] = toDay(d)
	}
	writeJSON(w, http.StatusOK, monthlyResponse{
		Location: toLocation(s.Place),
		Method:   toMethodRef(s.Method),
		Year:     year,
		Month:    month,
		Days:     days,
	})
}

// parseQuery reads the place and method parameters shared by all schedule
// endpoints. The place is either city, or lat + lng + tz (+ optional elev).
func parseQuery(q url.Values) (service.Query, error) {
	query := service.Query{CityID: q.Get("city"), MethodID: q.Get("method")}
	if query.CityID != "" {
		for _, k := range []string{"lat", "lng", "tz", "elev"} {
			if q.Has(k) {
				return query, fmt.Errorf("use either city or lat/lng/tz, not both (got city and %s)", k)
			}
		}
		return query, nil
	}

	var err error
	if query.Coord.Lat, err = parseFloat(q, "lat", -90, 90, true); err != nil {
		return query, fmt.Errorf("%w (or pass city, see /v1/cities)", err)
	}
	if query.Coord.Lng, err = parseFloat(q, "lng", -180, 180, true); err != nil {
		return query, err
	}
	if query.Coord.Elevation, err = parseFloat(q, "elev", 0, 9000, false); err != nil {
		return query, err
	}
	// tz is required: guessing it from longitude can be off by an hour near
	// zone borders (e.g. West vs. East Kalimantan).
	if query.Timezone = q.Get("tz"); query.Timezone == "" {
		return query, errors.New("tz is required, e.g. Asia/Jakarta, Asia/Makassar, Asia/Jayapura")
	}
	return query, nil
}

func parseFloat(q url.Values, key string, lo, hi float64, required bool) (float64, error) {
	s := q.Get(key)
	if s == "" {
		if required {
			return 0, fmt.Errorf("%s is required", key)
		}
		return 0, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < lo || v > hi {
		return 0, fmt.Errorf("%s must be a number between %g and %g", key, lo, hi)
	}
	return v, nil
}

func parseInt(q url.Values, key string, lo, hi int) (int, error) {
	v, err := strconv.Atoi(q.Get(key))
	if err != nil || v < lo || v > hi {
		return 0, fmt.Errorf("%s is required and must be an integer between %d and %d", key, lo, hi)
	}
	return v, nil
}
