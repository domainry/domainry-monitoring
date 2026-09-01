package module

import (
	"encoding/json"
	"fmt"

	"github.com/domainry/domainry-foundation/modulecapability"
)

const monitoringOperationsCategory = "monitoring.operations"

// NewCapabilityBinding projects the Monitoring-owned product HTTP contract and
// adaptation semantics into the common SDK capability facet. Monitoring has no
// authorable endpoint parameters or configuration candidates, so it truthfully
// discloses no validation scopes; the common method remains available and
// rejects any invented kind as outside the owner contract.
func NewCapabilityBinding() (*modulecapability.StaticBinding, error) {
	document, err := modulecapability.CategoryFromHTTPRoutes(modulecapability.HTTPRouteCategory{
		Owner: "monitoring",
		Category: modulecapability.CategorySummary{
			Key: monitoringOperationsCategory, Name: "Runtime monitoring", Description: "Read Monitoring-owned aggregated Runtime operational metrics.",
			AssemblyChains: []string{"runtime_observations_to_monitoring_snapshot"}, ValidationScopes: []string{},
		},
		Routes: monitoringRoutes(), Operations: monitoringOpenAPIOperations(),
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
		Scenarios: modulecapability.AdaptationScenarios{
			UseWhen:              []string{"A PRD needs Runtime health or operational metrics, readiness visibility, migration telemetry, or incident-diagnosis observations"},
			DoNotUseWhen:         []string{"The requirement is immutable business audit history, product analytics and reports, alert delivery, or automated recovery commands"},
			RequirementSignals:   []string{"runtime health", "operational metrics", "readiness", "migration telemetry", "incident diagnosis"},
			ProvidedCapabilities: []string{"monitoring.runtime_health", "monitoring.runtime_metrics", "monitoring.storage_readiness", "monitoring.migration_readiness", "monitoring.migration_telemetry"},
			RequiredModules:      []string{}, OptionalModules: []string{"audit", "lifecycle", "scheduler"}, ConflictingModules: []string{},
			AssemblyChains: []string{"runtime_observations_to_monitoring_snapshot"}, ValidationScopes: []string{},
			SelectionExamples: []modulecapability.ScenarioExample{{Requirement: "Expose a Runtime operations page with aggregate health and metrics for incident diagnosis", Reason: "Monitoring owns collection, error isolation, and aggregation of owner-produced operational observations"}},
			RejectionExamples: []modulecapability.ScenarioExample{{Requirement: "Keep an immutable history of who changed business records", Reason: "Audit owns immutable activity history; Monitoring owns current operational observations"}},
		},
	}
	return modulecapability.NewStaticBinding(summary, []modulecapability.CategoryDocument{document}, nil)
}
