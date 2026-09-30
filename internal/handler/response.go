package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/katarsisxyz/openpray/internal/model"
	"github.com/katarsisxyz/openpray/internal/service"
	"github.com/katarsisxyz/openpray/prayer"
)

type timesResponse struct {
	Imsak   string `json:"imsak"`
	Fajr    string `json:"fajr"`
	Sunrise string `json:"sunrise"`
	Dhuha   string `json:"dhuha"`
	Dhuhr   string `json:"dhuhr"`
	Asr     string `json:"asr"`
	Maghrib string `json:"maghrib"`
	Isha    string `json:"isha"`
}

type locationResponse struct {
	Lat       float64          `json:"lat"`
	Lng       float64          `json:"lng"`
	Elevation float64          `json:"elevation"`
	Timezone  string           `json:"timezone"`
	City      *cityRefResponse `json:"city,omitempty"`
}

type cityRefResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Province string `json:"province"`
}

type cityResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Capital   string  `json:"capital"`
	Province  string  `json:"province"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Elevation float64 `json:"elevation"`
	Timezone  string  `json:"timezone"`
}

type methodRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type dayResponse struct {
	Date  string        `json:"date"`
	Times timesResponse `json:"times"`
}

type dailyResponse struct {
	Location locationResponse  `json:"location"`
	Method   methodRefResponse `json:"method"`
	dayResponse
}

type monthlyResponse struct {
	Location locationResponse  `json:"location"`
	Method   methodRefResponse `json:"method"`
	Year     int               `json:"year"`
	Month    int               `json:"month"`
	Days     []dayResponse     `json:"days"`
}

type methodResponse struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	FajrAngle    float64             `json:"fajr_angle"`
	IshaAngle    float64             `json:"isha_angle"`
	DhuhaAngle   float64             `json:"dhuha_angle"`
	AsrFactor    float64             `json:"asr_factor"`
	RiseAngle    float64             `json:"rise_angle"`
	Ihtiyat      int                 `json:"ihtiyat_minutes"`
	ImsakMinutes int                 `json:"imsak_minutes"`
	Adjustments  adjustmentsResponse `json:"adjustments_minutes"`
}

type adjustmentsResponse struct {
	Imsak   int `json:"imsak"`
	Fajr    int `json:"fajr"`
	Sunrise int `json:"sunrise"`
	Dhuha   int `json:"dhuha"`
	Dhuhr   int `json:"dhuhr"`
	Asr     int `json:"asr"`
	Maghrib int `json:"maghrib"`
	Isha    int `json:"isha"`
}

func toLocation(p model.Place) locationResponse {
	l := locationResponse{Lat: p.Coord.Lat, Lng: p.Coord.Lng, Elevation: p.Coord.Elevation, Timezone: p.Location.String()}
	if c := p.City; c != nil {
		l.City = &cityRefResponse{ID: c.ID, Name: c.Name, Province: c.Province}
	}
	return l
}

func toCity(c model.City) cityResponse {
	return cityResponse{
		ID: c.ID, Name: c.Name, Capital: c.Capital, Province: c.Province,
		Lat: c.Coord.Lat, Lng: c.Coord.Lng, Elevation: c.Coord.Elevation, Timezone: c.Timezone,
	}
}

func toMethodRef(m model.Method) methodRefResponse {
	return methodRefResponse{ID: m.ID, Name: m.Name}
}

func toMethod(m model.Method) methodResponse {
	return methodResponse{
		ID: m.ID, Name: m.Name,
		FajrAngle: m.FajrAngle, IshaAngle: m.IshaAngle, DhuhaAngle: m.DhuhaAngle,
		AsrFactor: m.AsrFactor, RiseAngle: m.RiseAngle,
		Ihtiyat: m.Ihtiyat, ImsakMinutes: m.ImsakMinutes,
		Adjustments: adjustmentsResponse(m.Adjust),
	}
}

func toDay(d model.Day) dayResponse {
	hm := func(t time.Time) string { return t.Format("15:04") }
	t := d.Times
	return dayResponse{
		Date: d.Date.Format(time.DateOnly),
		Times: timesResponse{
			Imsak: hm(t.Imsak), Fajr: hm(t.Fajr), Sunrise: hm(t.Sunrise), Dhuha: hm(t.Dhuha),
			Dhuhr: hm(t.Dhuhr), Asr: hm(t.Asr), Maghrib: hm(t.Maghrib), Isha: hm(t.Isha),
		},
	}
}

// writeServiceError maps service errors to HTTP status codes.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrMethodNotFound):
		writeError(w, http.StatusBadRequest, err.Error()+" (see /v1/methods)")
	case errors.Is(err, service.ErrCityNotFound):
		writeError(w, http.StatusNotFound, err.Error()+" (see /v1/cities)")
	case errors.Is(err, service.ErrInvalidTimezone):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, prayer.ErrUndefined):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
