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
	contract := monitoringsdk.MonitoringHTTPSurfaceContract()
	routes, err := monitoringRoutes()
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(routes[0].Pattern(), func(http.ResponseWriter, *http.Request) {})
	surface := &surface{mux: mux, routes: routes, operations: monitoringOpenAPIOperations()}
	if surface.Owner() != contract.Owner || surface.Name() != contract.Name {
		t.Fatalf("surface identity=%s/%s contract=%s/%s", surface.Owner(), surface.Name(), contract.Owner, contract.Name)
	}
	routes = surface.Routes()
	if len(routes) != 1 || routes[0].Pattern() != contract.Routes[0].Pattern() || routes[0].Action.Permission == nil || routes[0].Action.Permission.Key != "monitoring.metrics.read" {
		t.Fatalf("surface routes=%#v", routes)
	}
	if surface.OpenAPIOperations()[routes[0].Pattern()]["operationId"] != "getMonitoringMetrics" {
		t.Fatalf("surface OpenAPI=%#v", surface.OpenAPIOperations())
	}
	if err := modulehttp.ValidateSurface(surface); err != nil {
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
	surface, err := NewSurface(binding)
	if err != nil {
		t.Fatal(err)
	}

	unauthenticated := httptest.NewRecorder()
	surface.Handler().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/operations/monitoring/metrics", nil))
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
			request := httptest.NewRequest(http.MethodGet, "/operations/monitoring/metrics", nil).WithContext(ctx)
			response := httptest.NewRecorder()
			surface.Handler().ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
	if binding.calls != 1 {
		t.Fatalf("metrics calls=%d want=1", binding.calls)
	}
}
