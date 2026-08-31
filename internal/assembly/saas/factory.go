package saas

import (
	"context"
	"fmt"

	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	"github.com/domainry/domainry-monitoring-sdk/saashost"
	monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/module"
)

// Factory decorates the SDK Remote Factory with Monitoring-owned product HTTP.
// Runtime still sees only SDK and modulehttp contracts.
type Factory struct {
	remote monitoringsdk.Factory
}

func NewFactory(remote monitoringsdk.Factory) *Factory {
	return &Factory{remote: remote}
}

func (f *Factory) Open(ctx context.Context, application monitoringsdk.ApplicationRef) (monitoringsdk.Binding, error) {
	if f == nil || f.remote == nil {
		return nil, fmt.Errorf("Monitoring SaaS Remote Factory is required")
	}
	binding, err := f.remote.Open(ctx, application)
	return withHTTPSurface(binding, err)
}

func (f *Factory) OpenSaaS(ctx context.Context, application monitoringsdk.ApplicationRef, host modulehost.Host) (monitoringsdk.Binding, error) {
	remote, ok := f.remote.(saashost.Factory)
	if !ok {
		return nil, fmt.Errorf("Monitoring SaaS Remote Factory does not implement saashost.Factory")
	}
	binding, err := remote.OpenSaaS(ctx, application, host)
	return withHTTPSurface(binding, err)
}

func withHTTPSurface(binding monitoringsdk.Binding, err error) (monitoringsdk.Binding, error) {
	if err != nil {
		return nil, err
	}
	surface, err := monitoringhttp.NewSurface(binding)
	if err != nil {
		_ = binding.Close(context.Background())
		return nil, err
	}
	return &bindingWithHTTPSurface{Binding: binding, surfaces: []modulehttp.Surface{surface}}, nil
}

type bindingWithHTTPSurface struct {
	monitoringsdk.Binding
	surfaces []modulehttp.Surface
}

func (b *bindingWithHTTPSurface) HTTPSurfaces() []modulehttp.Surface {
	return append([]modulehttp.Surface(nil), b.surfaces...)
}

var _ monitoringsdk.Factory = (*Factory)(nil)
var _ saashost.Factory = (*Factory)(nil)
var _ modulehttp.Provider = (*bindingWithHTTPSurface)(nil)
