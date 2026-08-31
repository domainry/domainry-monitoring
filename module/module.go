// Package module exposes the embedded Monitoring topology facade.
package module

import moduleassembly "github.com/domainry/domainry-monitoring/internal/assembly/module"

type Options = moduleassembly.Options
type Factory = moduleassembly.Factory

func OptionsFromEnvironment() Options     { return moduleassembly.OptionsFromEnvironment() }
func NewFactory(options Options) *Factory { return moduleassembly.NewFactory(options) }
