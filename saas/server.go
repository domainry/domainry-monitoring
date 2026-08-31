// Package saas exposes the standalone Monitoring service facade.
package saas

import saasassembly "github.com/domainry/domainry-monitoring/internal/assembly/saas"

type Options = saasassembly.Options
type Server = saasassembly.Server

func New(options Options) *Server { return saasassembly.New(options) }
