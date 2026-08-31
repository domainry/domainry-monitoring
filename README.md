# Domainry Monitoring

Domainry Monitoring aggregates Runtime owner observations into consistent health and metric snapshots.

## Module mode

Generated project composition supplies:

```go
monitoringmodule.NewFactory(monitoringmodule.OptionsFromEnvironment())
```

Aggregation runs inside the Runtime process and borrows host observation ports.

## SaaS mode

Run the service with:

```sh
DOMAINRY_MONITORING_TOKEN=replace-me ADDR=:8090 go run ./cmd/monitoring-server
```

Generated Runtime composition supplies `monitoringremote.NewFactory(monitoringremote.ConfigFromEnvironment())`. The Remote Binding negotiates `domainry-monitoring-protocol-v1`, collects the same host observations used by Module mode, and sends them to the SaaS evaluator.

Service endpoints:

- `GET /live`, `GET /ready`
- `GET /v1/descriptor`
- `POST /v1/health`
- `POST /v1/metrics`

The versioned API requires a bearer token. TLS is expected to terminate at the deployment ingress or service mesh.

## Source layout

- `module` is the public facade for embedded Runtime composition.
- `saas` is the public facade for the standalone service.
- `internal/application/monitoring` owns snapshot evaluation.
- `internal/assembly` composes Module and SaaS topologies.
- `internal/transport/http` owns the HTTP protocol implementation.
