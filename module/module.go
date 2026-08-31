// Package module exposes the embedded Monitoring topology facade.
package module

import (
	monitoringsdk "github.com/domainry/domainry-monitoring-sdk"
	moduleassembly "github.com/domainry/domainry-monitoring/internal/assembly/module"
	saasassembly "github.com/domainry/domainry-monitoring/internal/assembly/saas"
)

type Options = moduleassembly.Options
type Factory = moduleassembly.Factory

func OptionsFromEnvironment() Options     { return moduleassembly.OptionsFromEnvironment() }
func NewFactory(options Options) *Factory { return moduleassembly.NewFactory(options) }

// NewSaaSFactory keeps SaaS product HTTP ownership in Monitoring while the
// supplied SDK Factory remains responsible for remote service calls.
func NewSaaSFactory(remote monitoringsdk.Factory) monitoringsdk.Factory {
	return saasassembly.NewFactory(remote)
}
