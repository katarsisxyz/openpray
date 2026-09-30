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

// Daily handles GET /v1/times?lat=&lng=&tz=[&date=YYYY-MM-DD][&method=][&elev=].
func (h *PrayerHandler) Daily(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query, err := parseQuery(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	date := time.Now().In(query.Location)
	if s := q.Get("date"); s != "" {
		if date, err = time.ParseInLocation(time.DateOnly, s, query.Location); err != nil {
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
		Location:    toLocation(query),
		Method:      toMethodRef(s.Method),
		dayResponse: toDay(s.Days[0]),
	})
}

// Monthly handles GET /v1/times/month?lat=&lng=&tz=&year=&month=[&method=][&elev=].
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
		Location: toLocation(query),
		Method:   toMethodRef(s.Method),
		Year:     year,
		Month:    month,
		Days:     days,
	})
}

// parseQuery reads the location and method parameters shared by all schedule endpoints.
func parseQuery(q url.Values) (service.Query, error) {
	var query service.Query
	var err error
	if query.Coord.Lat, err = parseFloat(q, "lat", -90, 90, true); err != nil {
		return query, err
	}
	if query.Coord.Lng, err = parseFloat(q, "lng", -180, 180, true); err != nil {
		return query, err
	}
	if query.Coord.Elevation, err = parseFloat(q, "elev", 0, 9000, false); err != nil {
		return query, err
	}

	// tz is required: guessing it from longitude can be off by an hour near
	// zone borders (e.g. West vs. East Kalimantan).
	tz := q.Get("tz")
	if tz == "" {
		return query, errors.New("tz is required, e.g. Asia/Jakarta, Asia/Makassar, Asia/Jayapura")
	}
	if query.Location, err = time.LoadLocation(tz); err != nil {
		return query, fmt.Errorf("unknown tz: %q", tz)
	}

	query.MethodID = q.Get("method")
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
