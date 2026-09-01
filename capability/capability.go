// Package capability exposes Monitoring's source-owned capability contract
// without opening a monitoring host or transport.
package capability

import (
	"github.com/domainry/domainry-foundation/modulecapability"
	monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/module"
)

type Inputs struct{}

func Open(Inputs) (*modulecapability.StaticBinding, error) {
	return monitoringhttp.NewCapabilityBinding()
}
