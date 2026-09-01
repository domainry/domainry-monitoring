package saas

import (
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"
	"strings"

	"github.com/domainry/domainry-foundation/modulecapability"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/contract"
	monitoringcapability "github.com/domainry/domainry-monitoring/capability"
	monitoringapplication "github.com/domainry/domainry-monitoring/internal/application/monitoring"
)

const maxRequestBytes = 2 << 20

type Options struct{ BearerToken string }
type Handler struct {
	token string
	mux   *stdhttp.ServeMux
}

func New(options Options) (*Handler, error) {
	capability, err := monitoringcapability.Open(monitoringcapability.Inputs{})
	if err != nil {
		return nil, err
	}
	capabilityHTTP, err := modulecapability.NewHTTPHandler(capability, func(*stdhttp.Request) error { return nil })
	if err != nil {
		return nil, err
	}
	handler := &Handler{token: strings.TrimSpace(options.BearerToken), mux: stdhttp.NewServeMux()}
	handler.mux.Handle(modulecapability.SummaryPath, handler.authHandler(capabilityHTTP))
	handler.mux.Handle(modulecapability.CategoriesPath, handler.authHandler(capabilityHTTP))
	handler.mux.Handle(modulecapability.ValidationPath, handler.authHandler(capabilityHTTP))
	handler.register()
	return handler, nil
}

func (s *Handler) Routes() stdhttp.Handler {
	return s.mux
}

func (s *Handler) register() {
	s.mux.HandleFunc("GET /live", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("GET /ready", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("GET /v1/descriptor", s.auth(s.descriptor))
	s.mux.HandleFunc("POST /v1/health", s.auth(s.health))
	s.mux.HandleFunc("POST /v1/metrics", s.auth(s.metrics))
}

func (s *Handler) auth(next stdhttp.HandlerFunc) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if s.token != "" && r.Header.Get("Authorization") != "Bearer "+s.token {
			writeJSON(w, stdhttp.StatusUnauthorized, map[string]any{"error": "monitoring.unauthorized"})
			return
		}
		next(w, r)
	}
}

func (s *Handler) authHandler(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		s.auth(next.ServeHTTP)(w, r)
	})
}

func (*Handler) descriptor(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	writeJSON(w, stdhttp.StatusOK, monitoringsdk.Descriptor{ProtocolVersion: monitoringsdk.ProtocolVersionV1, Mode: monitoringsdk.DeploymentModeSaaS, Capabilities: []string{"health", "metrics", "readiness", "migration_telemetry"}})
}

func (*Handler) health(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var input contract.HealthRequest
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	if strings.TrimSpace(input.RuntimeID) == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "monitoring.runtime_id_required"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, monitoringapplication.EvaluateHealth(input))
}

func (*Handler) metrics(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var input contract.MetricsRequest
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	if strings.TrimSpace(input.RuntimeID) == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "monitoring.runtime_id_required"})
		return
	}
	writeJSON(w, stdhttp.StatusOK, monitoringapplication.EvaluateMetrics(input))
}

func decodeJSON(w stdhttp.ResponseWriter, r *stdhttp.Request, target any) error {
	reader := stdhttp.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "monitoring.request_invalid"})
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]any{"error": "monitoring.request_invalid"})
		return errors.New("multiple JSON values")
	}
	return nil
}

func writeJSON(w stdhttp.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
