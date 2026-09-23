# How should migration readiness and telemetry be observed?

## Problems solved

- Makes schema/migration readiness, progress, failures, and lag visible without allowing Monitoring to own migration execution or declare readiness from process liveness alone.

## Business scenarios

- A rollout waits because one required schema migration has not reached the target revision even though the process is live.
- An operator diagnoses a slow migration from current progress, last successful checkpoint, lag, and owner-reported failure evidence.
- An unavailable or stale telemetry source produces `unknown`, never a cached green readiness decision.

## Use when

Use migration observations during deployment, upgrade, or incident diagnosis when schema readiness and migration progress materially affect safe traffic.

## Do not use when

Do not let Monitoring apply migrations, rewrite ledgers, bypass the host migration lock, or infer business compatibility from a heartbeat.

## How to use

Read owner-produced target/current revision, readiness, checkpoint, progress, lag, and error observations; preserve freshness and distinguish unavailable telemetry from completed migration.

## Adaptation cookbook

| Migration requirement | Adapt with | Concrete implementation | Wrong adaptation |
| --- | --- | --- | --- |
| Traffic must wait for required schema revision | Migration readiness gate | Compare owner-reported current/target revision under the host ledger and lock; keep readiness false until all required owners are ready | Treating process liveness or an empty error string as migration completion |
| Long migration appears stuck | Migration telemetry | Show last checkpoint, rows/units completed, lag, heartbeat freshness, and owner error without changing execution | Restarting or editing migration state from Monitoring |
| Telemetry source times out | Explicit unavailable/partial state | Retain last observation timestamp, mark it stale/unavailable, and avoid a false green aggregate | Converting missing data to zero lag or complete |
| Product needs a customer data import | Data Exchange | Use a durable business import job and row validation | Calling a business import a schema migration |

## Example

A deployment is live but not ready: the owner reports target revision 18, current revision 17, checkpoint `orders-backfill:720000`, lag, and a fresh 72% observation. Traffic cutover remains blocked while the host migrator—under the host database, lock, and sole `_schema_migrations` ledger—continues. If telemetry expires or the owner is unreachable, Monitoring reports `unknown` with last observation time instead of reusing the prior `ready`; it never applies, repairs, or rewrites migration state.

## Permissions and scope

Migration telemetry is operational and may reveal schema identifiers or failure detail, so it requires operator permission. It must not expose credentials or cross-Workspace business data.

## Boundaries

The host and source owners own migrations, the sole ledger, lock, and repair. Monitoring owns observation aggregation only.
