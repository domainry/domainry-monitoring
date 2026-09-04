package module

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/domainry/domainry-foundation/modulehttp"
	identitysdk "github.com/domainry/domainry-identity-sdk"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
)

func TestSurfaceUsesMonitoringSDKHTTPContract(t *testing.T) {
	contract := monitoringsdk.MonitoringHTTPAdapterContract()
	routes, err := monitoringRoutes()
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(routes[0].Pattern(), func(http.ResponseWriter, *http.Request) {})
	adapter := &adapter{mux: mux, routes: routes, operations: monitoringOpenAPIOperations()}
	if adapter.Owner() != contract.Owner || adapter.Name() != contract.Name {
		t.Fatalf("adapter identity=%s/%s contract=%s/%s", adapter.Owner(), adapter.Name(), contract.Owner, contract.Name)
	}
	routes = adapter.Routes()
	if len(routes) != 1 || routes[0].Pattern() != contract.Routes[0].Pattern() || routes[0].Action.Permission == nil || routes[0].Action.Permission.Key != "monitoring.metrics.read" {
		t.Fatalf("adapter routes=%#v", routes)
	}
	if adapter.OpenAPIOperations()[routes[0].Pattern()]["operationId"] != "getMonitoringMetrics" {
		t.Fatalf("adapter OpenAPI=%#v", adapter.OpenAPIOperations())
	}
	if err := modulehttp.ValidateAdapter(adapter); err != nil {
		t.Fatal(err)
	}
}

type metricsBindingStub struct {
	monitoringsdk.Binding
	calls int
}

func (binding *metricsBindingStub) Metrics(context.Context) map[string]any {
	binding.calls++
	return map[string]any{"objects": 3}
}

func TestSurfaceEnforcesUnrestrictedMetricsScopeAtOwningHandler(t *testing.T) {
	binding := &metricsBindingStub{}
	adapter, err := NewAdapter(binding)
	if err != nil {
		t.Fatal(err)
	}

	unauthenticated := httptest.NewRecorder()
	adapter.Handler().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/monitoring/metrics", nil))
	if unauthenticated.Code != http.StatusUnauthorized || binding.calls != 0 {
		t.Fatalf("unauthenticated status=%d calls=%d", unauthenticated.Code, binding.calls)
	}

	now := time.Now().UTC()
	for _, test := range []struct {
		name  string
		scope identitysdk.DataScope
		want  int
	}{
		{name: "all", scope: identitysdk.DataScopeAll, want: http.StatusOK},
		{name: "owner", scope: identitysdk.DataScopeOwner, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			principal := metricsPrincipal(now, monitoringsdk.ActionMonitoringMetricsRead, monitoringsdk.ActionMonitoringMetricsRead, test.scope)
			ctx := identitysdk.WithRequestIdentity(t.Context(), identitysdk.RequestIdentity{Principal: principal})
			request := httptest.NewRequest(http.MethodGet, "/monitoring/metrics", nil).WithContext(ctx)
			response := httptest.NewRecorder()
			adapter.Handler().ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
	if binding.calls != 1 {
		t.Fatalf("metrics calls=%d want=1", binding.calls)
	}
}
