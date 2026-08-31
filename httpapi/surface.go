// Package httpapi exposes Monitoring's product HTTP contract independently
// from whether its SDK Binding is local or remote.
package httpapi

import (
	foundationhttp "github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/module"
)

// NewSurface adapts a Monitoring Binding to the module-owned product HTTP
// paths. The same Surface is used with Module and SaaS Bindings.
func NewSurface(binding monitoringsdk.Binding) (foundationhttp.Surface, error) {
	return monitoringhttp.NewSurface(binding)
}
