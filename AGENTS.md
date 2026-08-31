# Domainry Monitoring development guide

- This repository is an independent Go module and must not import Domainry Runtime or Plane implementation packages.
- Public production packages are topology facades only: `module` for embedded composition and `saas` for the remote service.
- Monitoring application evaluation and topology implementations stay under `internal`.
- Module mode borrows observation capabilities from the host through `domainry-monitoring-sdk/modulehost`; it does not own host storage, migrations, scheduling, or lifecycle state.
- SaaS and Module modes must produce equivalent health and metric snapshots from equivalent observations.
- Monitoring currently owns no database. Do not add a migration ledger or persistence package unless Monitoring gains source-owned durable business state.
- Consume released Domainry module tags; do not add local `replace` directives.
