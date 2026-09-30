// Package router is the single place where API endpoints are registered.
package router

import (
	"net/http"

	"github.com/katarsisxyz/openpray/internal/handler"
	"github.com/katarsisxyz/openpray/internal/middleware"
)

// New registers all endpoints and wraps them in middleware.
func New(prayer *handler.PrayerHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /openapi.yaml", handler.OpenAPISpec)

	mux.HandleFunc("GET /v1/methods", prayer.ListMethods)
	mux.HandleFunc("GET /v1/times", prayer.Daily)
	mux.HandleFunc("GET /v1/times/month", prayer.Monthly)

	return middleware.CORS(mux)
}
