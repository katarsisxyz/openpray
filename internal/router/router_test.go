package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/katarsisxyz/openpray/internal/handler"
	"github.com/katarsisxyz/openpray/internal/repository"
	"github.com/katarsisxyz/openpray/internal/service"
)

const jakarta = "lat=-6.1754&lng=106.8272&tz=Asia/Jakarta"

// newTestRouter wires every layer the same way cmd/server does.
func newTestRouter() http.Handler {
	svc := service.NewPrayerService(repository.NewMethodRepository())
	return New(handler.NewPrayerHandler(svc))
}

func get(t *testing.T, path string, out any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v (%s)", path, err, rec.Body)
		}
	}
	return rec.Code
}

type times struct{ Imsak, Fajr, Sunrise, Dhuha, Dhuhr, Asr, Maghrib, Isha string }

func TestDaily(t *testing.T) {
	var resp struct {
		Date     string
		Method   struct{ ID string }
		Location struct{ Timezone string }
		Times    times
	}
	if code := get(t, "/v1/times?"+jakarta+"&date=2026-09-30", &resp); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	// Reference: Kemenag schedule for KOTA JAKARTA on 2026-09-30.
	want := times{"04:14", "04:24", "05:35", "06:02", "11:46", "14:52", "17:50", "18:59"}
	if resp.Times != want {
		t.Errorf("times = %+v, want %+v", resp.Times, want)
	}
	if resp.Date != "2026-09-30" || resp.Method.ID != "kemenag" || resp.Location.Timezone != "Asia/Jakarta" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestMonthly(t *testing.T) {
	var resp struct {
		Year, Month int
		Days        []struct{ Date string }
	}
	if code := get(t, "/v1/times/month?"+jakarta+"&year=2026&month=2&method=muhammadiyah", &resp); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if len(resp.Days) != 28 || resp.Days[0].Date != "2026-02-01" || resp.Days[27].Date != "2026-02-28" {
		t.Errorf("days = %+v", resp.Days)
	}
}

func TestMethods(t *testing.T) {
	var resp struct{ Methods []struct{ ID string } }
	if code := get(t, "/v1/methods", &resp); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if len(resp.Methods) != 3 || resp.Methods[0].ID != "kemenag" {
		t.Errorf("methods = %+v", resp.Methods)
	}
}

func TestHealth(t *testing.T) {
	if code := get(t, "/healthz", nil); code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
}

func TestOpenAPI(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	// Every route in router.go must be documented in the spec.
	body := rec.Body.String()
	for _, path := range []string{"/v1/times:", "/v1/times/month:", "/v1/methods:", "/healthz:", "/openapi.yaml:"} {
		if !strings.Contains(body, "\n  "+path) {
			t.Errorf("openapi.yaml does not document %s", strings.TrimSuffix(path, ":"))
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name, method, path string
		code               int
	}{
		{"missing tz", http.MethodGet, "/v1/times?lat=-6.1&lng=106.8", http.StatusBadRequest},
		{"missing lng", http.MethodGet, "/v1/times?lat=-6.1&tz=Asia/Jakarta", http.StatusBadRequest},
		{"lat out of range", http.MethodGet, "/v1/times?lat=100&lng=106.8&tz=Asia/Jakarta", http.StatusBadRequest},
		{"unknown tz", http.MethodGet, "/v1/times?lat=-6.1&lng=106.8&tz=Mars/Base", http.StatusBadRequest},
		{"unknown method", http.MethodGet, "/v1/times?" + jakarta + "&method=foo", http.StatusBadRequest},
		{"bad date format", http.MethodGet, "/v1/times?" + jakarta + "&date=30-09-2026", http.StatusBadRequest},
		{"invalid month", http.MethodGet, "/v1/times/month?" + jakarta + "&year=2026&month=13", http.StatusBadRequest},
		{"high latitude", http.MethodGet, "/v1/times?lat=69.65&lng=18.96&tz=Europe/Oslo&date=2026-06-21", http.StatusUnprocessableEntity},
		{"unknown route", http.MethodGet, "/v1/unknown", http.StatusNotFound},
		{"wrong HTTP method", http.MethodPost, "/v1/times", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newTestRouter().ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.code {
				t.Errorf("status = %d, want %d (%s)", rec.Code, tt.code, rec.Body)
			}
		})
	}
}
