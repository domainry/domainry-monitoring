package capability

import (
	"encoding/json"
	"fmt"

	"github.com/domainry/domainry-foundation/modulecapability"
	monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/module"
)

const monitoringOperationsCategory = "monitoring.operations"

// openContract projects the Monitoring-owned product HTTP contract and
// adaptation semantics into the common SDK capability facet. Monitoring has no
// authorable endpoint parameters or configuration candidates, so it truthfully
// discloses no validation scopes; the common method remains available and
// rejects any invented kind as outside the owner contract.
func openContract(_ Inputs) (*modulecapability.StaticBinding, error) {
	routes, err := monitoringhttp.CapabilityRoutes()
	if err != nil {
		return nil, err
	}
	document, err := modulecapability.CategoryFromHTTPRoutes(modulecapability.HTTPRouteCategory{
		Owner: "monitoring",
		Category: modulecapability.CategorySummary{
			Key: monitoringOperationsCategory, Name: "Runtime monitoring", Description: "Read Monitoring-owned aggregated Runtime operational metrics.",
			AssemblyChains: []string{"runtime_observations_to_monitoring_snapshot"}, ValidationScopes: []string{},
		},
		Routes: routes, Operations: monitoringhttp.CapabilityOpenAPIOperations(),
		Components: map[string]map[string]json.RawMessage{
			"securitySchemes": {"BearerAuth": json.RawMessage(`{"type":"http","scheme":"bearer","bearerFormat":"JWT"}`)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("project Monitoring HTTP capability: %w", err)
	}
	summary := modulecapability.ModuleSummary{
		Identity: modulecapability.ModuleIdentity{
			Key: "monitoring", SourceOwner: "monitoring", ModuleVersion: "domainry-monitoring-protocol-v1", ValidationRevision: "monitoring-no-authorable-candidates-v1",
			SupportedDeploymentModes: []modulecapability.DeploymentMode{modulecapability.DeploymentModeModule, modulecapability.DeploymentModeSaaS},
		},
		Name: "Monitoring", Description: "Aggregates Runtime health, readiness, migration telemetry, and owner-shaped operational metrics without taking ownership of source module state.",
		Composition: modulecapability.ModuleComposition{
			ProvidedCapabilities: []string{"monitoring.runtime_health", "monitoring.runtime_metrics", "monitoring.storage_readiness", "monitoring.migration_readiness", "monitoring.migration_telemetry"},
			RequiredModules:      []string{}, OptionalModules: []string{"audit", "lifecycle", "scheduler"}, ConflictingModules: []string{},
			AssemblyChains: []string{"runtime_observations_to_monitoring_snapshot"}, ValidationScopes: []string{},
		},
	}
	return modulecapability.NewStaticBinding(summary, []modulecapability.CategoryDocument{document}, nil)
}
