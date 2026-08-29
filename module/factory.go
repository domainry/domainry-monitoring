// Package module assembles Monitoring over observations borrowed from its host.
package module

import (
	"context"
	"fmt"

	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
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
	return &binding{runtimeID: application.RuntimeID, host: host}, nil
}

var _ monitoringsdk.Factory = (*Factory)(nil)
var _ modulehost.Factory = (*Factory)(nil)
