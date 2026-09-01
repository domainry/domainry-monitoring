package saas

import monitoringhttp "github.com/domainry/domainry-monitoring/internal/transport/http/saas"

type Options = monitoringhttp.Options
type Server = monitoringhttp.Handler

func New(options Options) (*Server, error) { return monitoringhttp.New(options) }
