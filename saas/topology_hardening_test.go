package saas_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/domainry/domainry-foundation/modulehttp"
	identitysdk "github.com/domainry/domainry-identity-sdk"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	"github.com/domainry/domainry-monitoring-sdk/remote"
	monitoringmodule "github.com/domainry/domainry-monitoring/module"
	monitoringsaas "github.com/domainry/domainry-monitoring/saas"
)

type observationHost struct {
	identity  modulehost.Identity
	storage   observationStorage
	migration observationMigration
	scheduler observationComponent
	lifecycle observationComponent
	metrics   observationMetrics
}

func (h observationHost) Identity() modulehost.Identity   { return h.identity }
func (h observationHost) Storage() modulehost.Storage     { return h.storage }
func (h observationHost) Migration() modulehost.Migration { return h.migration }
func (h observationHost) Scheduler() modulehost.Component { return h.scheduler }
func (h observationHost) Lifecycle() modulehost.Component { return h.lifecycle }
func (h observationHost) Metrics() modulehost.Metrics     { return h.metrics }

type observationStorage struct {
	payload map[string]any
	err     error
}

func (s observationStorage) Status(context.Context) (map[string]any, error) {
	return s.payload, s.err
}
func (observationStorage) Readiness(context.Context) error { return nil }

type observationMigration struct {
	status modulehost.Status
	err    error
}

func (m observationMigration) Status(context.Context) (modulehost.Status, error) {
	return m.status, m.err
}
func (observationMigration) Readiness(context.Context) error { return nil }
func (observationMigration) Telemetry(context.Context) (int, bool, error) {
	return 0, true, nil
}

type observationComponent struct {
	payload map[string]any
	err     error
}

func (c observationComponent) Observe(context.Context) (map[string]any, error) {
	return c.payload, c.err
}

type observationMetrics struct {
	sections map[string]any
	errors   map[string]string
}

func (m observationMetrics) Observe(context.Context) (map[string]any, map[string]string) {
	return m.sections, m.errors
}

func TestModuleAndSaaSPreserveFailureAndEmptyWindowObservations(t *testing.T) {
	boundary := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	host := observationHost{
		identity:  modulehost.Identity{TemplateID: "template", TemplateVersion: "v2"},
		storage:   observationStorage{err: errors.New("storage unavailable")},
		migration: observationMigration{status: modulehost.Status{Current: false}, err: context.DeadlineExceeded},
		scheduler: observationComponent{err: context.DeadlineExceeded},
		lifecycle: observationComponent{err: errors.New("lifecycle unavailable")},
		metrics: observationMetrics{
			sections: map[string]any{
				"audit_window": map[string]any{
					"window_start": boundary.Format(time.RFC3339Nano),
					"window_end":   boundary.Format(time.RFC3339Nano),
					"samples":      []any{},
				},
			},
			errors: map[string]string{"audit": "upstream unavailable", "scheduler": context.DeadlineExceeded.Error()},
		},
	}
	moduleBinding, remoteBinding := openTopologies(t, host, false)

	directHealth := moduleBinding.Health(t.Context())
	remoteHealth := remoteBinding.Health(t.Context())
	if !equivalent(directHealth, remoteHealth) {
		t.Fatalf("health topology mismatch\nmodule=%s\nsaas=%s", normalized(directHealth), normalized(remoteHealth))
	}
	checks, ok := directHealth["checks"].(map[string]string)
	if !ok || checks["storage"] != "error" || checks["migration"] != "error" || checks["scheduler"] != "error" || checks["lifecycle"] != "error" {
		t.Fatalf("health error mapping=%#v", directHealth)
	}

	directMetrics := moduleBinding.Metrics(t.Context())
	remoteMetrics := remoteBinding.Metrics(t.Context())
	if !equivalent(directMetrics, remoteMetrics) {
		t.Fatalf("metrics topology mismatch\nmodule=%s\nsaas=%s", normalized(directMetrics), normalized(remoteMetrics))
	}
	window, ok := directMetrics["audit_window"].(map[string]any)
	samples, samplesOK := window["samples"].([]any)
	if !ok || !samplesOK || window["window_start"] != window["window_end"] || len(samples) != 0 {
		t.Fatalf("empty owner window was not preserved: %#v", directMetrics)
	}
	metricErrors, ok := directMetrics["errors"].(map[string]string)
	if !ok || metricErrors["scheduler"] != context.DeadlineExceeded.Error() {
		t.Fatalf("metric error mapping=%#v", directMetrics)
	}
}

func TestModuleAndSaaSProductSurfacesEnforceWorkspaceAuthorization(t *testing.T) {
	moduleBinding, remoteBinding := openTopologies(t, remoteHost{}, true)
	boundary := time.Now().UTC()
	valid := authorizedMetricsPrincipal(boundary)
	wrongWorkspace := clonePrincipal(valid)
	wrongWorkspace.AccessBundle.Subject.WorkspaceID = "other-workspace"
	expired := clonePrincipal(valid)
	expired.AccessBundle.ExpiresAt = boundary

	tests := []struct {
		name      string
		principal *identitysdk.Principal
		want      int
	}{
		{name: "authentication required", want: http.StatusUnauthorized},
		{name: "workspace mismatch", principal: &wrongWorkspace, want: http.StatusForbidden},
		{name: "expired at boundary", principal: &expired, want: http.StatusForbidden},
		{name: "matching workspace", principal: &valid, want: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			moduleStatus, modulePayload := callMetricsAdapter(t, moduleBinding, test.principal)
			remoteStatus, remotePayload := callMetricsAdapter(t, remoteBinding, test.principal)
			if moduleStatus != test.want || remoteStatus != test.want {
				t.Fatalf("status module=%d saas=%d want=%d", moduleStatus, remoteStatus, test.want)
			}
			if normalized(modulePayload) != normalized(remotePayload) {
				t.Fatalf("adapter payload mismatch\nmodule=%s\nsaas=%s", normalized(modulePayload), normalized(remotePayload))
			}
		})
	}
}

func TestRemoteBindingMapsServiceFailureAndTimeout(t *testing.T) {
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(handler.Routes())
	t.Cleanup(service.Close)
	client := service.Client()
	client.Transport = failingObservationTransport{base: client.Transport}
	binding, err := remote.NewFactory(remote.Config{
		Endpoint: service.URL, Token: "secret", Client: client, Timeout: 20 * time.Millisecond,
	}).OpenSaaS(t.Context(), monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}, remoteHost{})
	if err != nil {
		t.Fatal(err)
	}

	healthErrors, ok := binding.Health(t.Context())["errors"].(map[string]string)
	if !ok || !strings.Contains(healthErrors["monitoring_saas"], "status 502") {
		t.Fatalf("health service failure mapping=%#v", healthErrors)
	}
	metricErrors, ok := binding.Metrics(t.Context())["errors"].(map[string]string)
	if !ok || !strings.Contains(metricErrors["monitoring_saas"], context.DeadlineExceeded.Error()) {
		t.Fatalf("metrics timeout mapping=%#v", metricErrors)
	}
}

func TestModuleAndSaaSConcurrentReadsAreDeterministic(t *testing.T) {
	host := observationHost{
		identity:  modulehost.Identity{TemplateID: "template", TemplateVersion: "concurrent"},
		storage:   observationStorage{payload: map[string]any{"ping": "ok"}},
		migration: observationMigration{status: modulehost.Status{Current: true, Payload: map[string]any{"current": true}}},
		scheduler: observationComponent{payload: map[string]any{"runtime_available": true, "unresolved_dead_letters": 2}},
		lifecycle: observationComponent{payload: map[string]any{}},
		metrics: observationMetrics{sections: map[string]any{
			"objects": 3,
			"owners":  map[string]any{"audit": 1, "scheduler": 2},
		}},
	}
	moduleBinding, remoteBinding := openTopologies(t, host, false)
	expectedHealth := normalized(moduleBinding.Health(t.Context()))
	expectedMetrics := normalized(moduleBinding.Metrics(t.Context()))
	if got := normalized(remoteBinding.Health(t.Context())); got != expectedHealth {
		t.Fatalf("initial health mismatch: %s != %s", got, expectedHealth)
	}
	if got := normalized(remoteBinding.Metrics(t.Context())); got != expectedMetrics {
		t.Fatalf("initial metrics mismatch: %s != %s", got, expectedMetrics)
	}

	const readers = 24
	start := make(chan struct{})
	errorsByReader := make(chan error, readers*2)
	var readersDone sync.WaitGroup
	for topology, binding := range map[string]monitoringsdk.Binding{"module": moduleBinding, "saas": remoteBinding} {
		for reader := 0; reader < readers; reader++ {
			readersDone.Add(1)
			go func(topology string, binding monitoringsdk.Binding) {
				defer readersDone.Done()
				<-start
				for iteration := 0; iteration < 8; iteration++ {
					if got := normalized(binding.Health(t.Context())); got != expectedHealth {
						errorsByReader <- fmt.Errorf("%s health changed: %s", topology, got)
						return
					}
					if got := normalized(binding.Metrics(t.Context())); got != expectedMetrics {
						errorsByReader <- fmt.Errorf("%s metrics changed: %s", topology, got)
						return
					}
				}
			}(topology, binding)
		}
	}
	close(start)
	readersDone.Wait()
	close(errorsByReader)
	for err := range errorsByReader {
		t.Error(err)
	}
}

func openTopologies(t *testing.T, host modulehost.Host, decorateRemote bool) (monitoringsdk.Binding, monitoringsdk.Binding) {
	t.Helper()
	application := monitoringsdk.ApplicationRef{RuntimeID: "runtime-1"}
	moduleBinding, err := monitoringmodule.NewFactory(monitoringmodule.Options{}).OpenModule(t.Context(), application, host)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := monitoringsaas.New(monitoringsaas.Options{BearerToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(handler.Routes())
	t.Cleanup(service.Close)
	var factory monitoringsdk.Factory = remote.NewFactory(remote.Config{
		Endpoint: service.URL, Token: "secret", Client: service.Client(),
	})
	if decorateRemote {
		factory = monitoringmodule.NewSaaSFactory(factory)
	}
	opener, ok := factory.(interface {
		OpenSaaS(context.Context, monitoringsdk.ApplicationRef, modulehost.Host) (monitoringsdk.Binding, error)
	})
	if !ok {
		t.Fatalf("SaaS factory %T cannot accept an observation host", factory)
	}
	remoteBinding, err := opener.OpenSaaS(t.Context(), application, host)
	if err != nil {
		t.Fatal(err)
	}
	return moduleBinding, remoteBinding
}

func callMetricsAdapter(t *testing.T, binding monitoringsdk.Binding, principal *identitysdk.Principal) (int, map[string]any) {
	t.Helper()
	provider, ok := binding.(modulehttp.Provider)
	if !ok || len(provider.HTTPAdapters()) != 1 {
		t.Fatalf("Monitoring binding %T has no product HTTP adapter", binding)
	}
	request := httptest.NewRequest(http.MethodGet, "/monitoring/metrics", nil)
	if principal != nil {
		request = request.WithContext(identitysdk.WithRequestIdentity(request.Context(), identitysdk.RequestIdentity{Principal: *principal}))
	}
	response := httptest.NewRecorder()
	provider.HTTPAdapters()[0].Handler().ServeHTTP(response, request)
	payload := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return response.Code, payload
}

func authorizedMetricsPrincipal(now time.Time) identitysdk.Principal {
	bundle := identitysdk.AccessBundle{
		ContractVersion:       identitysdk.CurrentPolicyBundleVersion,
		AuthorizationRevision: "monitoring-integration-authorization",
		ExpiresAt:             now.Add(time.Hour),
		Subject: identitysdk.Subject{
			WorkspaceID: "workspace", SubjectID: "user", OrgID: "org",
			OrgScopeIDs: []string{"org"}, SupportOrgScopeIDs: []string{},
		},
		FunctionGrants: []identitysdk.FunctionGrant{{
			Resource: "monitoring.metrics", Action: "read", Effect: identitysdk.EffectAllow,
		}},
		DataPolicies: []identitysdk.DataPolicy{{
			Key: "monitoring-metrics-read-all", Resource: "monitoring.metrics", Action: "read", Effect: identitysdk.EffectAllow,
			DataScopes: []identitysdk.DataScope{identitysdk.DataScopeAll},
		}},
	}
	return identitysdk.Principal{
		ContractVersion: identitysdk.PrincipalContextContractVersion,
		Known:           true,
		WorkspaceID:     "workspace",
		UserID:          "user",
		AccessBundle:    &bundle,
	}
}

func clonePrincipal(source identitysdk.Principal) identitysdk.Principal {
	clone := source
	bundle := *source.AccessBundle
	bundle.FunctionGrants = append([]identitysdk.FunctionGrant(nil), source.AccessBundle.FunctionGrants...)
	bundle.DataPolicies = append([]identitysdk.DataPolicy(nil), source.AccessBundle.DataPolicies...)
	clone.AccessBundle = &bundle
	return clone
}

type failingObservationTransport struct{ base http.RoundTripper }

func (transport failingObservationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	switch request.URL.Path {
	case "/monitoring/v1/health":
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Status:     "502 Bad Gateway",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"upstream"}`)),
			Request:    request,
		}, nil
	case "/monitoring/v1/metrics":
		<-request.Context().Done()
		return nil, request.Context().Err()
	default:
		return transport.base.RoundTrip(request)
	}
}
