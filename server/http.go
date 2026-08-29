package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/contract"
	monitoringapplication "github.com/domainry/domainry-monitoring/application"
)

const maxRequestBytes = 2 << 20

type Options struct{ BearerToken string }

type Server struct{ token string }

func New(options Options) *Server { return &Server{token: strings.TrimSpace(options.BearerToken)} }

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /v1/descriptor", s.auth(s.descriptor))
	mux.HandleFunc("POST /v1/health", s.auth(s.health))
	mux.HandleFunc("POST /v1/metrics", s.auth(s.metrics))
	return mux
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" && r.Header.Get("Authorization") != "Bearer "+s.token {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "monitoring.unauthorized"})
			return
		}
		next(w, r)
	}
}

func (*Server) descriptor(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, monitoringsdk.Descriptor{ProtocolVersion: monitoringsdk.ProtocolVersionV1, Mode: monitoringsdk.DeploymentModeSaaS, Capabilities: []string{"health", "metrics", "readiness", "migration_telemetry"}})
}

func (*Server) health(w http.ResponseWriter, r *http.Request) {
	var input contract.HealthRequest
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	if strings.TrimSpace(input.RuntimeID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "monitoring.runtime_id_required"})
		return
	}
	writeJSON(w, http.StatusOK, monitoringapplication.EvaluateHealth(input))
}

func (*Server) metrics(w http.ResponseWriter, r *http.Request) {
	var input contract.MetricsRequest
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	if strings.TrimSpace(input.RuntimeID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "monitoring.runtime_id_required"})
		return
	}
	writeJSON(w, http.StatusOK, monitoringapplication.EvaluateMetrics(input))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "monitoring.request_invalid"})
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "monitoring.request_invalid"})
		return errors.New("multiple JSON values")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
