package module

import (
	"testing"

	"github.com/domainry/domainry-foundation/modulecapability/contracttest"
)

func TestMonitoringCapabilityTracksSourceOwnedSurface(t *testing.T) {
	binding, err := NewCapabilityBinding()
	if err != nil {
		t.Fatal(err)
	}
	contracttest.VerifyBinding(t, binding)
	summary, err := binding.CapabilitySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Categories) != 1 || summary.Categories[0].OperationCount != len(monitoringRoutes()) || len(summary.Scenarios.ValidationScopes) != 0 {
		t.Fatalf("Monitoring capability summary=%+v", summary)
	}
	contracttest.VerifyModuleRemoteParity(t, binding)
}
