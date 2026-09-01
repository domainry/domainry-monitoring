package module

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
)

type surface struct {
	binding monitoringsdk.Binding
}

func (*surface) ContractVersion() string { return modulehttp.ContractVersion }
func (*surface) Owner() string           { return monitoringsdk.MonitoringHTTPSurfaceContract().Owner }
func (*surface) Name() string            { return monitoringsdk.MonitoringHTTPSurfaceContract().Name }
func (*surface) Routes() []modulehttp.Route {
	return monitoringRoutes()
}
func monitoringRoutes() []modulehttp.Route {
	contract := monitoringsdk.MonitoringHTTPSurfaceContract()
	routes := make([]modulehttp.Route, 0, len(contract.Routes))
	for _, route := range contract.Routes {
		exposures := make([]modulehttp.Exposure, len(route.Exposures))
		for index, exposure := range route.Exposures {
			exposures[index] = modulehttp.Exposure(exposure)
		}
		routes = append(routes, modulehttp.Route{
			Pattern: route.Pattern, Exposures: exposures, Authentication: modulehttp.Authentication(route.Authentication),
			Permission: route.Permission, AnyPermissions: append([]string(nil), route.AnyPermissions...), PrincipalOnly: route.PrincipalOnly,
			Governance: &modulehttp.Governance{
				EffectClass: modulehttp.EffectClass(route.EffectClass), HighRiskPolicy: modulehttp.HighRiskPolicy(route.HighRiskPolicy),
				IdempotencyDecision: route.IdempotencyDecision, AuditClass: route.AuditClass,
			},
		})
	}
	return routes
}
func (*surface) OpenAPIOperations() map[string]map[string]any {
	return monitoringOpenAPIOperations()
}
func monitoringOpenAPIOperations() map[string]map[string]any {
	return monitoringsdk.MonitoringHTTPSurfaceContract().OpenAPI
}
func (s *surface) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.binding.Metrics(r.Context()))
	})
}

func NewSurface(binding monitoringsdk.Binding) (modulehttp.Surface, error) {
	if binding == nil {
		return nil, errors.New("Monitoring binding is unavailable")
	}
	return &surface{binding: binding}, nil
}

var _ modulehttp.Surface = (*surface)(nil)
var _ modulehttp.OpenAPIProvider = (*surface)(nil)
