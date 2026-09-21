// Package capability exposes Monitoring's source-owned capability contract
// without opening a monitoring host or transport.
package capability

import (
	"github.com/domainry/domainry-foundation/modulecapability"
)

type Inputs struct{}

func Open(inputs Inputs) (*modulecapability.StaticBinding, error) {
	return openContract(inputs)
}
