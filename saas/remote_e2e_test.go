package saas_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	"github.com/domainry/domainry-monitoring-sdk/remote"
	monitoringmodule "github.com/domainry/domainry-monitoring/module"
	monitoringsaas "github.com/domainry/domainry-monitoring/saas"
)

type remoteHost struct{}

func (remoteHost) Identity() modulehost.Identity {
	return modulehost.Identity{TemplateID: "template", TemplateVersion: "v1"}
}
func (remoteHost) Storage() modulehost.Storage     { return remoteStorage{} }
func (remoteHost) Migration() modulehost.Migration { return remoteMigration{} }
func (remoteHost) Scheduler() modulehost.Component {
	return remoteComponent{map[string]any{"runtime_available": true}}
}
func (remoteHost) Lifecycle() modulehost.Component { return remoteComponent{map[string]any{}} }
func (remoteHost) Metrics() modulehost.Metrics     { return remoteMetrics{} }

type remoteStorage struct{}

func (remoteStorage) Status(context.Context) (map[string]any, error) {
	return map[string]any{"ping": "ok"}, nil
}
func (remoteStorage) Readiness(context.Context) error { return nil }

type remoteMigration struct{}

func (remoteMigration) Status(context.Context) (modulehost.Status, error) {
	return modulehost.Status{Current: true, Payload: map[string]any{"current": true}}, nil
}
func (remoteMigration) Readiness(context.Context) error              { return nil }
func (remoteMigration) Telemetry(context.Context) (int, bool, error) { return 0, true, nil }

type remoteComponent struct{ value map[string]any }

func (c remoteComponent) Observe(context.Context) (map[string]any, error) { return c.value, nil }

type remoteMetrics struct{}

func (remoteMetrics) Observe(context.Context) (map[string]any, map[string]string) {
	return map[string]any{"objects": 3}, nil
}

func TestRemoteBindingUsesSaaSEvaluator(t *testing.T) {
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(handler.Routes())
	defer service.Close()
	factory := monitoringmodule.NewSaaSFactory(remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "secret", Client: service.Client()}))
	binding, err := factory.(interface {
		OpenSaaS(context.Context, monitoringsdk.ApplicationRef, modulehost.Host) (monitoringsdk.Binding, error)
	}).OpenSaaS(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	if binding.Descriptor().Mode != monitoringsdk.DeploymentModeSaaS {
		t.Fatalf("descriptor=%#v", binding.Descriptor())
	}
	provider, ok := binding.(modulehttp.Provider)
	if !ok || len(provider.HTTPAdapters()) != 1 || provider.HTTPAdapters()[0].Routes()[0].Pattern() != "GET /monitoring/metrics" {
		t.Fatalf("SaaS Monitoring HTTP adapters=%v", provider)
	}
	if health := binding.Health(t.Context()); health["status"] != "ok" || health["runtime_id"] != "runtime-1" {
		t.Fatalf("health=%#v", health)
	}
	if metrics := binding.Metrics(t.Context()); metrics["objects"] != float64(3) || metrics["runtime_id"] != "runtime-1" {
		t.Fatalf("metrics=%#v", metrics)
	}
}

func TestRemoteBindingRejectsWrongToken(t *testing.T) {
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(handler.Routes())
	defer service.Close()
	_, err = remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "wrong", Client: service.Client()}).OpenSaaS(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}, remoteHost{})
	if err == nil {
		t.Fatal("unauthorized descriptor accepted")
	}
}

func TestModuleAndSaaSTopologiesProduceEquivalentSnapshots(t *testing.T) {
	application := monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}
	moduleBinding, err := monitoringmodule.NewFactory(monitoringmodule.Options{}).OpenModule(t.Context(), application, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(handler.Routes())
	defer service.Close()
	remoteBinding, err := remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "secret", Client: service.Client()}).OpenSaaS(t.Context(), application, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	if !equivalent(moduleBinding.Health(t.Context()), remoteBinding.Health(t.Context())) {
		t.Fatal("health topology mismatch")
	}
	if !equivalent(moduleBinding.Metrics(t.Context()), remoteBinding.Metrics(t.Context())) {
		t.Fatal("metrics topology mismatch")
	}
	if err := remoteBinding.Descriptor().Validate(); err != nil || remoteBinding.Descriptor().Mode != monitoringsdk.DeploymentModeSaaS {
		t.Fatalf("remote descriptor=%+v err=%v", remoteBinding.Descriptor(), err)
	}
}

func equivalent(left, right map[string]any) bool {
	return normalized(left) == normalized(right)
}
