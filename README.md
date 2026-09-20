# Domainry Monitoring

Product Agent disclosure index: [`capability/agent/index.json`](capability/agent/index.json). It intentionally contains no leaf guides because Runtime composes health, readiness, metrics, and migration telemetry automatically; these operations-owned capabilities must not enter product modeling context.

Domainry Monitoring aggregates Runtime owner observations into consistent health and metric snapshots.

## Module mode

Generated project composition supplies:

```go
monitoringmodule.NewFactory(monitoringmodule.OptionsFromEnvironment())
```

Aggregation runs inside the Runtime process and borrows host observation ports.

The product route `GET /monitoring/metrics` requires the exact
`monitoring.metrics.read` Permission and its same-resource/action data policy.
Because the response is one whole-Runtime aggregate without owner or
organization record facts, only the canonical `all` data scope is accepted;
`owner`, `org`, `org_child`, and `target_org` fail closed.

## SaaS mode

Run the service with:

```sh
DOMAINRY_MONITORING_TOKEN=replace-me ADDR=:8090 go run ./cmd/monitoring-server
```

Generated Runtime composition supplies `monitoringremote.NewFactory(monitoringremote.ConfigFromEnvironment())`. The Remote Binding negotiates `domainry-monitoring-protocol-v1`, collects the same host observations used by Module mode, and sends them to the SaaS evaluator.

Service endpoints:

- `GET /live`, `GET /ready`
- `GET /monitoring/v1/descriptor`
- `POST /monitoring/v1/health`
- `POST /monitoring/v1/metrics`

The versioned API requires a bearer token. TLS is expected to terminate at the deployment ingress or service mesh.

## Source layout

- `module` is the public facade for embedded Runtime composition.
- `saas` is the public facade for the standalone service.
- `internal/application/monitoring` owns snapshot evaluation.
- `internal/assembly` composes Module and SaaS topologies.
- `internal/transport/http` owns the HTTP protocol implementation.
