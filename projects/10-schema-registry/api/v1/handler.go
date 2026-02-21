// Package api exposes the schema registry over a Confluent-compatible REST API.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Polqt/schemaregistry/registry/compat"
	"github.com/Polqt/schemaregistry/registry/schema"
)

// Handler builds the HTTP handler for the schema registry REST API.
// Endpoints mirror the Confluent Schema Registry API v1.
//
//	GET    /subjects                              → list subjects
//	GET    /subjects/{subject}/versions           → list versions
//	GET    /subjects/{subject}/versions/latest    → latest schema
//	GET    /subjects/{subject}/versions/{version} → specific version
//	POST   /subjects/{subject}/versions           → register schema
//	DELETE /subjects/{subject}                    → delete subject
//	GET    /schemas/ids/{id}                      → get by global ID
//	POST   /compatibility/subjects/{subject}/versions/latest → check compat
//	GET    /config/{subject}                      → get compat level
//	PUT    /config/{subject}                      → set compat level
func Handler(reg *schema.Registry) http.Handler {
	mux := http.NewServeMux()

	// List subjects.
	mux.HandleFunc("GET /subjects", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, reg.Subjects())
	})

	// List versions.
	mux.HandleFunc("GET /subjects/", func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, reg)
	})

	// Get schema by global ID.
	mux.HandleFunc("GET /schemas/ids/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/schemas/ids/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			httpError(w, 400, "invalid id")
			return
		}
		s, err := reg.GetByID(id)
		if err != nil {
			httpError(w, 404, err.Error())
			return
		}
		writeJSON(w, map[string]string{"schema": s.Content})
	})

	// Compatibility check endpoint.
	mux.HandleFunc("POST /compatibility/subjects/", func(w http.ResponseWriter, r *http.Request) {
		// Path: /compatibility/subjects/{subject}/versions/latest
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/compatibility/subjects/"), "/")
		if len(parts) < 1 {
			httpError(w, 400, "bad path")
			return
		}
		subject := parts[0]

		var req struct {
			Schema string        `json:"schema"`
			Format schema.Format `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		latest, err := reg.GetLatest(subject)
		if err != nil {
			httpError(w, 404, err.Error())
			return
		}
		level, _ := reg.CompatLevel(subject)
		checker := &compat.Checker{}
		err = checker.Check(latest.Format, latest.Content, req.Schema, level)
		if err != nil {
			writeJSON(w, map[string]bool{"is_compatible": false})
			return
		}
		writeJSON(w, map[string]bool{"is_compatible": true})
	})

	// Config endpoints.
	mux.HandleFunc("GET /config/", func(w http.ResponseWriter, r *http.Request) {
		subject := strings.TrimPrefix(r.URL.Path, "/config/")
		level, err := reg.CompatLevel(subject)
		if err != nil {
			httpError(w, 404, err.Error())
			return
		}
		writeJSON(w, map[string]string{"compatibility": string(level)})
	})
	mux.HandleFunc("PUT /config/", func(w http.ResponseWriter, r *http.Request) {
		subject := strings.TrimPrefix(r.URL.Path, "/config/")
		var req struct{ Compatibility schema.CompatLevel `json:"compatibility"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, 400, err.Error())
			return
		}
		if err := reg.SetCompatLevel(subject, req.Compatibility); err != nil {
			httpError(w, 404, err.Error())
			return
		}
		writeJSON(w, map[string]string{"compatibility": string(req.Compatibility)})
	})

	return mux
}

// handle dispatches /subjects/{subject}/... routes.
func handle(w http.ResponseWriter, r *http.Request, reg *schema.Registry) {
	path := strings.TrimPrefix(r.URL.Path, "/subjects/")
	parts := strings.SplitN(path, "/", 3)

	if len(parts) < 1 {
		httpError(w, 400, "bad path")
		return
	}
	subject := parts[0]

	if len(parts) == 1 {
		// DELETE /subjects/{subject}
		if r.Method == http.MethodDelete {
			if err := reg.Delete(subject); err != nil {
				httpError(w, 404, err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		httpError(w, 405, "method not allowed")
		return
	}

	if parts[1] != "versions" {
		httpError(w, 404, "not found")
		return
	}

	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			// GET /subjects/{subject}/versions
			vers, err := reg.Versions(subject)
			if err != nil {
				httpError(w, 404, err.Error())
				return
			}
			writeJSON(w, vers)
		case http.MethodPost:
			// POST /subjects/{subject}/versions — register schema
			var req struct {
				Schema string        `json:"schema"`
				Format schema.Format `json:"schemaType"`
			}
			if req.Format == "" {
				req.Format = schema.FormatJSONSchema
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httpError(w, 400, err.Error())
				return
			}
			id, err := reg.Register(subject, req.Format, req.Schema)
			if err != nil {
				httpError(w, 409, err.Error())
				return
			}
			w.WriteHeader(http.StatusOK)
			writeJSON(w, map[string]int{"id": id})
		default:
			httpError(w, 405, "method not allowed")
		}
		return
	}

	// GET /subjects/{subject}/versions/{version}
	versionStr := parts[2]
	var s *schema.Schema
	var err error
	if versionStr == "latest" {
		s, err = reg.GetLatest(subject)
	} else {
		v, parseErr := strconv.Atoi(versionStr)
		if parseErr != nil {
			httpError(w, 400, "invalid version")
			return
		}
		s, err = reg.GetVersion(subject, v)
	}
	if err != nil {
		httpError(w, 404, err.Error())
		return
	}
	writeJSON(w, map[string]any{
		"subject": s.Subject,
		"version": s.Version,
		"id":      s.ID,
		"schema":  s.Content,
		"schemaType": string(s.Format),
	})
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("encode: %v", err), 500)
	}
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"error_code": code, "message": msg})
}
