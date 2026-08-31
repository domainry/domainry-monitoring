package saas

import monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http"

type Options = monitoringhttp.Options
type Server = monitoringhttp.Handler

func New(options Options) *Server { return monitoringhttp.New(options) }
