package module

import (
	"context"

	"github.com/domainry/domainry-foundation/modulecapability"
	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	monitoringapplication "github.com/domainry/domainry-monitoring/internal/application/monitoring"
)

type binding struct {
	runtimeID  string
	host       modulehost.Host
	surfaces   []modulehttp.Surface
	capability modulecapability.Binding
}

func (b *binding) CapabilitySummary(ctx context.Context) (modulecapability.ModuleSummary, error) {
	return b.capability.CapabilitySummary(ctx)
}
func (b *binding) CapabilityCategory(ctx context.Context, key string) (modulecapability.CategoryDocument, error) {
	return b.capability.CapabilityCategory(ctx, key)
}
func (b *binding) ValidateCapabilityCandidate(ctx context.Context, request modulecapability.ValidationRequest) (modulecapability.ValidationResult, error) {
	return b.capability.ValidateCapabilityCandidate(ctx, request)
}

func (*binding) Descriptor() monitoringsdk.Descriptor {
	return monitoringsdk.Descriptor{ProtocolVersion: monitoringsdk.ProtocolVersionV1, Mode: monitoringsdk.DeploymentModeModule, Capabilities: []string{"health", "metrics", "readiness", "migration_telemetry"}}
}

func (b *binding) Health(ctx context.Context) map[string]any {
	return monitoringapplication.EvaluateHealth(modulehost.CollectHealth(ctx, b.runtimeID, b.host))
}

func (b *binding) Metrics(ctx context.Context) map[string]any {
	return monitoringapplication.EvaluateMetrics(modulehost.CollectMetrics(ctx, b.runtimeID, b.host))
}

func (b *binding) StorageReadiness(ctx context.Context) error { return b.host.Storage().Readiness(ctx) }
func (b *binding) MigrationReadiness(ctx context.Context) error {
	return b.host.Migration().Readiness(ctx)
}
func (b *binding) MigrationTelemetry(ctx context.Context) (int, bool, error) {
	return b.host.Migration().Telemetry(ctx)
}
func (*binding) Close(context.Context) error { return nil }
func (b *binding) HTTPSurfaces() []modulehttp.Surface {
	return append([]modulehttp.Surface(nil), b.surfaces...)
}

var _ monitoringsdk.Binding = (*binding)(nil)
var _ modulehttp.Provider = (*binding)(nil)
