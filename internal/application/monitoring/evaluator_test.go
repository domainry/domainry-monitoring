package monitoring

import (
	"testing"

	"github.com/domainry/domainry-monitoring-sdk/contract"
)

func TestEvaluateHealthClassifiesFailuresAndWarnings(t *testing.T) {
	health := EvaluateHealth(contract.HealthRequest{
		RuntimeID: "runtime-1",
		Storage:   contract.ComponentObservation{Error: "unavailable"},
		Migration: contract.MigrationObservation{Current: false},
		Scheduler: contract.ComponentObservation{Payload: map[string]any{"runtime_available": true, "unresolved_dead_letters": float64(2), "lease_expirations": int64(1)}},
		Lifecycle: contract.ComponentObservation{Payload: map[string]any{"warning": true}},
	})
	if health["status"] != "degraded" {
		t.Fatalf("health=%#v", health)
	}
	checks := health["checks"].(map[string]string)
	if checks["storage"] != "error" || checks["migration"] != "outdated" || checks["scheduler"] != "warning" || checks["lifecycle"] != "warning" {
		t.Fatalf("checks=%#v", checks)
	}
}

func TestEvaluateMetricsIncludesSectionsAndErrors(t *testing.T) {
	metrics := EvaluateMetrics(contract.MetricsRequest{RuntimeID: "runtime-1", Sections: map[string]any{"objects": 3}, Errors: map[string]string{"audit": "unavailable"}})
	if metrics["runtime_id"] != "runtime-1" || metrics["objects"] != 3 || metrics["errors"] == nil {
		t.Fatalf("metrics=%#v", metrics)
	}
}
