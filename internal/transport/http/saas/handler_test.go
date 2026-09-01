package saas

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/domainry/domainry-foundation/modulecapability"
	"github.com/domainry/domainry-monitoring-sdk/contract"
)

func TestHandlerEvaluatesAuthenticatedHealth(t *testing.T) {
	server, err := New(Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()
	input := contract.HealthRequest{RuntimeID: "runtime-1", Identity: contract.Identity{TemplateID: "template"}, Storage: contract.ComponentObservation{Payload: map[string]any{"ping": "ok"}}, Migration: contract.MigrationObservation{Current: true, Payload: map[string]any{"current": true}}, Scheduler: contract.ComponentObservation{Payload: map[string]any{"runtime_available": true}}, Lifecycle: contract.ComponentObservation{Payload: map[string]any{}}}
	body, _ := json.Marshal(input)
	request := httptest.NewRequest(stdhttp.MethodPost, "/v1/health", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != stdhttp.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload["status"] != "ok" || payload["runtime_id"] != "runtime-1" {
		t.Fatalf("payload=%#v error=%v", payload, err)
	}
}

func TestHandlerRejectsUnauthorizedAndInvalidRequests(t *testing.T) {
	server, err := New(Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(stdhttp.MethodGet, "/v1/descriptor", nil))
	if unauthorized.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("status=%d", unauthorized.Code)
	}
	capabilityUnauthorized := httptest.NewRecorder()
	handler.ServeHTTP(capabilityUnauthorized, httptest.NewRequest(stdhttp.MethodGet, modulecapability.SummaryPath, nil))
	if capabilityUnauthorized.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("capability status=%d", capabilityUnauthorized.Code)
	}
	invalid := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodPost, "/v1/metrics", bytes.NewBufferString(`{"unknown":true}`))
	request.Header.Set("Authorization", "Bearer secret")
	handler.ServeHTTP(invalid, request)
	if invalid.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status=%d", invalid.Code)
	}
}
