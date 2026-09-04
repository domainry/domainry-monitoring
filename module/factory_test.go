package module

import (
	"context"
	"testing"

	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
)

type hostStub struct{}

func (hostStub) Identity() modulehost.Identity {
	return modulehost.Identity{TemplateID: "template", TemplateVersion: "v1"}
}
func (hostStub) Storage() modulehost.Storage     { return storageStub{} }
func (hostStub) Migration() modulehost.Migration { return migrationStub{} }
func (hostStub) Scheduler() modulehost.Component {
	return componentStub{value: map[string]any{"runtime_available": true, "unresolved_dead_letters": 1}}
}
func (hostStub) Lifecycle() modulehost.Component {
	return componentStub{value: map[string]any{"warning": true}}
}
func (hostStub) Metrics() modulehost.Metrics { return metricsStub{} }

type storageStub struct{}

func (storageStub) Status(context.Context) (map[string]any, error) {
	return map[string]any{"ping": "ok"}, nil
}
func (storageStub) Readiness(context.Context) error { return nil }

type migrationStub struct{}

func (migrationStub) Status(context.Context) (modulehost.Status, error) {
	return modulehost.Status{Current: true, Payload: map[string]any{"current": true}}, nil
}
func (migrationStub) Readiness(context.Context) error              { return nil }
func (migrationStub) Telemetry(context.Context) (int, bool, error) { return 0, true, nil }

type componentStub struct {
	value map[string]any
	err   error
}

func (s componentStub) Observe(context.Context) (map[string]any, error) { return s.value, s.err }

type metricsStub struct{}

func (metricsStub) Observe(context.Context) (map[string]any, map[string]string) {
	return map[string]any{"objects": 2}, map[string]string{"audit": "unavailable"}
}

func TestModuleAggregatesHealthAndMetrics(t *testing.T) {
	binding, err := NewFactory(Options{}).OpenModule(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime"}, hostStub{})
	if err != nil {
		t.Fatal(err)
	}
	health := binding.Health(t.Context())
	if health["status"] != "degraded" {
		t.Fatalf("health=%#v", health)
	}
	metrics := binding.Metrics(t.Context())
	if metrics["objects"] != 2 || metrics["errors"] == nil {
		t.Fatalf("metrics=%#v", metrics)
	}
	provider, ok := binding.(modulehttp.Provider)
	if !ok || len(provider.HTTPAdapters()) != 1 {
		t.Fatalf("Monitoring HTTP adapters=%v", provider)
	}
	if err := modulehttp.ValidateAdapter(provider.HTTPAdapters()[0]); err != nil {
		t.Fatal(err)
	}
}

func TestModuleRejectsIncompleteHost(t *testing.T) {
	if _, err := NewFactory(Options{}).OpenModule(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime"}, nil); err == nil {
		t.Fatal("nil host accepted")
	}
}
