package module

import (
	"context"

	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	"github.com/domainry/domainry-monitoring-sdk/modulehost"
	monitoringapplication "github.com/domainry/domainry-monitoring/application"
)

type binding struct {
	runtimeID string
	host      modulehost.Host
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

var _ monitoringsdk.Binding = (*binding)(nil)
