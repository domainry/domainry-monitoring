package module

import (
	"testing"

	"github.com/domainry/domainry-foundation/modulehttp"
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
)

func TestSurfaceUsesMonitoringSDKHTTPContract(t *testing.T) {
	contract := monitoringsdk.MonitoringHTTPSurfaceContract()
	surface := &surface{}
	if surface.Owner() != contract.Owner || surface.Name() != contract.Name {
		t.Fatalf("surface identity=%s/%s contract=%s/%s", surface.Owner(), surface.Name(), contract.Owner, contract.Name)
	}
	routes := surface.Routes()
	if len(routes) != 1 || routes[0].Pattern != contract.Routes[0].Pattern || routes[0].Governance == nil {
		t.Fatalf("surface routes=%#v", routes)
	}
	if surface.OpenAPIOperations()[routes[0].Pattern]["operationId"] != "getMonitoringMetrics" {
		t.Fatalf("surface OpenAPI=%#v", surface.OpenAPIOperations())
	}
	if err := modulehttp.ValidateSurface(surface); err != nil {
		t.Fatal(err)
	}
}
