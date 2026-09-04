package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/domainry/domainry-foundation/modulehttp"
	identitysdk "github.com/domainry/domainry-identity-sdk"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
)

type adapter struct {
	binding    monitoringsdk.Binding
	mux        *http.ServeMux
	routes     []modulehttp.Route
	operations map[string]map[string]any
}

func (*adapter) ContractVersion() string { return modulehttp.ContractVersion }
func (*adapter) Owner() string           { return monitoringsdk.MonitoringHTTPAdapterContract().Owner }
func (*adapter) Name() string            { return monitoringsdk.MonitoringHTTPAdapterContract().Name }
func (s *adapter) Routes() []modulehttp.Route {
	return append([]modulehttp.Route(nil), s.routes...)
}
func monitoringRoutes() ([]modulehttp.Route, error) {
	contract := monitoringsdk.MonitoringHTTPAdapterContract()
	routes := make([]modulehttp.Route, 0, len(contract.Routes))
	for _, declared := range contract.Routes {
		route, err := modulehttp.RouteFromAction(declared.Action)
		if err != nil {
			return nil, fmt.Errorf("project Monitoring Action %q: %w", declared.Action.Key, err)
		}
		routes = append(routes, route)
	}
	return routes, nil
}
func (s *adapter) OpenAPIOperations() map[string]map[string]any {
	return s.operations
}
func monitoringOpenAPIOperations() map[string]map[string]any {
	return monitoringsdk.MonitoringHTTPAdapterContract().OpenAPIOperations()
}
func (s *adapter) Handler() http.Handler { return s.mux }

func NewAdapter(binding monitoringsdk.Binding) (modulehttp.Adapter, error) {
	if binding == nil {
		return nil, errors.New("Monitoring binding is unavailable")
	}
	routes, err := monitoringRoutes()
	if err != nil {
		return nil, err
	}
	s := &adapter{binding: binding, mux: http.NewServeMux(), routes: routes, operations: monitoringOpenAPIOperations()}
	handlers := map[string]http.HandlerFunc{monitoringsdk.ActionMonitoringMetricsRead: s.metrics}
	for _, route := range routes {
		handler, found := handlers[route.Action.Key]
		if !found {
			return nil, fmt.Errorf("Monitoring Action %q has no HTTP handler", route.Action.Key)
		}
		if _, found := s.operations[route.Pattern()]; !found {
			return nil, fmt.Errorf("Monitoring Action %q has no OpenAPI operation", route.Action.Key)
		}
		s.mux.HandleFunc(route.Pattern(), handler)
		delete(handlers, route.Action.Key)
	}
	if len(handlers) != 0 || len(s.operations) != len(routes) {
		return nil, errors.New("Monitoring handler, Action, and OpenAPI inventories differ")
	}
	return s, nil
}

func (s *adapter) metrics(w http.ResponseWriter, r *http.Request) {
	principal, ok := identitysdk.PrincipalFromContext(r.Context())
	if !ok {
		writeModuleError(w, http.StatusUnauthorized, "backend.monitoring.authentication_required")
		return
	}
	if !hasUnrestrictedPermission(principal, monitoringsdk.ActionMonitoringMetricsRead, time.Now()) {
		writeModuleError(w, http.StatusForbidden, "backend.monitoring.metrics_scope_denied")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.binding.Metrics(r.Context()))
}

func writeModuleError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code})
}

var _ modulehttp.Adapter = (*adapter)(nil)
var _ modulehttp.OpenAPIProvider = (*adapter)(nil)
