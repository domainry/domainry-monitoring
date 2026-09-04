package module

import (
	"testing"

	"github.com/domainry/domainry-foundation/modulecapability/contracttest"
)

func TestMonitoringCapabilityTracksSourceOwnedAdapter(t *testing.T) {
	binding, err := NewCapabilityBinding()
	if err != nil {
		t.Fatal(err)
	}
	contracttest.VerifyBinding(t, binding)
	summary, err := binding.CapabilitySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	routes, err := monitoringRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Categories) != 1 || summary.Categories[0].OperationCount != len(routes) || len(summary.Scenarios.ValidationScopes) != 0 {
		t.Fatalf("Monitoring capability summary=%+v", summary)
	}
	contracttest.VerifyModuleRemoteParity(t, binding)
}
