## Database safety

- [ ] I ran `go test ./...`.
- [ ] I ran `go run ./cmd/goquent review --fail-on high ./...`.
- [ ] I included relevant `QueryPlan` or review output for changed database code.
- [ ] Any `RiskLow` result is treated only as structural DB risk, not business approval.

## Policy and tenant safety

- [ ] Tenant-scoped queries include the tenant filter.
- [ ] Soft-delete behavior is intentional.
- [ ] Required filters are present.
- [ ] PII selection is avoided or includes a narrow access reason.

## Migrations

- [ ] I ran `go run ./cmd/goquent migrate plan <migration.sql>`.
- [ ] Destructive or high-risk migration steps have an explicit approval reason.
- [ ] Preflight checks are documented for data-loss or locking risks.

## Manifest and AI context

- [ ] I ran `go run ./cmd/goquent manifest verify --manifest goquent.manifest.json ...`.
- [ ] The manifest is fresh, or this PR regenerates it.
- [ ] AI-generated reads use OperationSpec where possible.
- [ ] Static review findings marked `partial` or `unsupported` are not described as safe.

## Suppressions

- [ ] Every suppression has a reason.
- [ ] Temporary suppressions have an expiration date.
- [ ] Non-suppressible findings were fixed instead of hidden.
