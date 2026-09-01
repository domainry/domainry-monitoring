package module

import (
	"context"
	"fmt"

	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	monitoringcapability "github.com/domainry/domainry-monitoring/capability"
	monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/module"
)

type Options struct{}

func OptionsFromEnvironment() Options { return Options{} }

type Factory struct{ options Options }

func NewFactory(options Options) *Factory { return &Factory{options: options} }

func (*Factory) Open(context.Context, monitoringsdk.ApplicationRef) (monitoringsdk.Binding, error) {
	return nil, fmt.Errorf("Monitoring Module host is required")
}

func (*Factory) OpenModule(_ context.Context, application monitoringsdk.ApplicationRef, host modulehost.Host) (monitoringsdk.Binding, error) {
	if err := application.Validate(); err != nil {
		return nil, err
	}
	if host == nil || host.Storage() == nil || host.Migration() == nil || host.Scheduler() == nil || host.Lifecycle() == nil || host.Metrics() == nil {
		return nil, fmt.Errorf("Monitoring host is incomplete")
	}
	capability, err := monitoringcapability.Open(monitoringcapability.Inputs{})
	if err != nil {
		return nil, fmt.Errorf("build Monitoring capability disclosure: %w", err)
	}
	result := &binding{runtimeID: application.RuntimeID, host: host, capability: capability}
	surface, err := monitoringhttp.NewSurface(result)
	if err != nil {
		return nil, err
	}
	result.surfaces = []modulehttp.Surface{surface}
	return result, nil
}

var _ monitoringsdk.Factory = (*Factory)(nil)
var _ modulehost.Factory = (*Factory)(nil)
