// Package api contains the OpenAPI spec for the OpenPray HTTP API.
package api

import _ "embed"

// OpenAPI is the OpenAPI 3.1 spec (YAML) served at GET /openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte
