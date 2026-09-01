package saas_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/domainry/domainry-foundation/modulecapability"
	"github.com/domainry/domainry-foundation/modulecapability/contracttest"
	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	"github.com/domainry/domainry-monitoring-sdk/remote"
	monitoringcapability "github.com/domainry/domainry-monitoring/capability"
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

func monitoringCapabilitySHA256(t testing.TB) string {
	t.Helper()
	binding, err := monitoringcapability.Open(monitoringcapability.Inputs{})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := binding.CapabilitySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return summary.Identity.ContractSHA256
}

func TestRemoteBindingUsesSaaSEvaluator(t *testing.T) {
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(handler.Routes())
	defer service.Close()
	factory := monitoringmodule.NewSaaSFactory(remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "secret", Client: service.Client(), CapabilityContractSHA256: monitoringCapabilitySHA256(t)}))
	binding, err := factory.(interface {
		OpenSaaS(context.Context, monitoringsdk.ApplicationRef, modulehost.Host) (monitoringsdk.Binding, error)
	}).OpenSaaS(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	if binding.Descriptor().Mode != monitoringsdk.DeploymentModeSaaS {
		t.Fatalf("descriptor=%#v", binding.Descriptor())
	}
	contracttest.VerifyBinding(t, binding)
	provider, ok := binding.(modulehttp.Provider)
	if !ok || len(provider.HTTPSurfaces()) != 1 || provider.HTTPSurfaces()[0].Routes()[0].Pattern != "GET /operations/monitoring/metrics" {
		t.Fatalf("SaaS Monitoring HTTP surfaces=%v", provider)
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
	_, err = remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "wrong", Client: service.Client(), CapabilityContractSHA256: monitoringCapabilitySHA256(t)}).OpenSaaS(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}, remoteHost{})
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
	directSummary, err := moduleBinding.CapabilitySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	remoteBinding, err := remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "secret", Client: service.Client(), CapabilityContractSHA256: directSummary.Identity.ContractSHA256}).OpenSaaS(t.Context(), application, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}
	contracttest.VerifyBinding(t, remoteBinding)
	if !equivalent(moduleBinding.Health(t.Context()), remoteBinding.Health(t.Context())) {
		t.Fatal("health topology mismatch")
	}
	if !equivalent(moduleBinding.Metrics(t.Context()), remoteBinding.Metrics(t.Context())) {
		t.Fatal("metrics topology mismatch")
	}
	remoteSummary, err := remoteBinding.CapabilitySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	assertCanonicalCapabilityEqual(t, directSummary, remoteSummary)
	for _, category := range directSummary.Categories {
		directDocument, directErr := moduleBinding.CapabilityCategory(t.Context(), category.Key)
		remoteDocument, remoteErr := remoteBinding.CapabilityCategory(t.Context(), category.Key)
		if directErr != nil || remoteErr != nil {
			t.Fatalf("load capability category %q: direct=%v remote=%v", category.Key, directErr, remoteErr)
		}
		assertCanonicalCapabilityEqual(t, directDocument, remoteDocument)
	}
	if _, err := remote.NewFactory(remote.Config{Endpoint: service.URL, Token: "secret", Client: service.Client(), CapabilityContractSHA256: strings.Repeat("0", 64)}).OpenSaaS(t.Context(), application, remoteHost{}); err == nil {
		t.Fatal("Monitoring Remote accepted a stale capability digest")
	}
}

func assertCanonicalCapabilityEqual(t *testing.T, left, right any) {
	t.Helper()
	leftBytes, err := modulecapability.CanonicalJSON(left)
	if err != nil {
		t.Fatal(err)
	}
	rightBytes, err := modulecapability.CanonicalJSON(right)
	if err != nil {
		t.Fatal(err)
	}
	if string(leftBytes) != string(rightBytes) {
		t.Fatalf("capability topology mismatch\ndirect=%s\nremote=%s", leftBytes, rightBytes)
	}
}

func equivalent(left, right map[string]any) bool {
	return normalized(left) == normalized(right)
}
