package handler

import (
	"net/http"

	"github.com/katarsisxyz/openpray/api"
)

// OpenAPISpec serves the OpenAPI spec, for Swagger UI or SDK generators.
func OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Write(api.OpenAPI)
}
