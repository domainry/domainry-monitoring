# When should an operations view use Monitoring health and metrics?

## Problems solved

- Aggregates owner-produced Runtime health, storage readiness, and operational metrics with failure isolation instead of scraping logs or querying module internals.

## Business scenarios

- An operator investigates elevated errors by comparing Runtime health, owner metrics, and storage readiness on one operations page.
- A deployment gate checks whether required components are ready without treating business-record counts or Audit events as health signals.
- A live process reports not-ready while storage is unavailable, and stale/partial metrics remain visibly stale instead of becoming a green aggregate.

## Use when

Use Monitoring for current operational state, aggregate health, readiness, saturation, error, or latency observations used in incident diagnosis and deployment visibility.

## Do not use when

Do not use Monitoring for immutable actor history, business analytics, user alerts, or automated repair commands.

## How to use

Collect bounded typed observations from each owner, isolate collection failures, aggregate freshness/status/metrics, and present current state with observation timestamps and partial-data warnings.

## Adaptation cookbook

| Operational requirement | Adapt with | Concrete implementation | Wrong adaptation |
| --- | --- | --- | --- |
| Operator needs one Runtime health page | Aggregated health snapshot | Query owner observations, show component status/freshness/errors, and mark partial collection without declaring the whole Runtime healthy | Scraping log strings or returning healthy when one required owner timed out |
| Database or object storage may be unavailable | Storage readiness observation | Report connection/readiness evidence without exposing credentials or mutating storage | Running repair SQL from the health endpoint |
| Product asks for monthly revenue trend | Report | Define governed business measures/dimensions and data scope | Deriving product analytics from request counters |
| Compliance asks who changed a Role | Audit query | Read immutable actor-attributed evidence | Treating a current metric or log line as historical proof |

## Example

During an incident, process liveness succeeds because Runtime can answer probes, while readiness fails because required storage is unavailable; business traffic remains gated. The operations page shows database state, object-storage observation time, latency/error-rate sample time, and Notification metrics unavailable with a collection error. Expired samples are `stale`, missing owners make the aggregate `partial`, and Monitoring triggers no hidden repair. Operators/Automation/Notification may react through their own permissions. Audit retains historical actor evidence; Report computes business analytics.

## Permissions and scope

Operational health/metric access is separate from business-record access and recovery commands. Responses must not expose secrets, raw tenant data, or arbitrary labels with sensitive values.

## Boundaries

Monitoring aggregates observations. Each owner defines its metrics/readiness facts and remains responsible for recovery or business state.
