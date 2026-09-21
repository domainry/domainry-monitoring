package capability

import (
	"testing"

	"github.com/domainry/domainry-foundation/modulecapability/contracttest"
	monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/module"
)

func TestMonitoringCapabilityTracksSourceOwnedAdapter(t *testing.T) {
	binding, err := Open(Inputs{})
	if err != nil {
		t.Fatal(err)
	}
	contracttest.VerifyBinding(t, binding)
	summary, err := binding.CapabilitySummary(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	routes, err := monitoringhttp.CapabilityRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Categories) != 1 || summary.Categories[0].OperationCount != len(routes) || len(summary.Composition.ValidationScopes) != 0 {
		t.Fatalf("Monitoring capability summary=%+v", summary)
	}
	contracttest.VerifyModuleRemoteParity(t, binding)
}
