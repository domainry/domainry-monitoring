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
func (*surface) Owner() string           { return "monitoring" }
func (*surface) Name() string            { return "operations_metrics" }
func (*surface) Routes() []modulehttp.Route {
	return []modulehttp.Route{{
		Pattern:        "GET /operations/monitoring/metrics",
		Exposures:      []modulehttp.Exposure{modulehttp.ExposureOps},
		Authentication: modulehttp.AuthenticationAuthenticated,
		AnyPermissions: []string{"workspace.admin", "runtime_ops.capability_status.read"},
	}}
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
