# Goquent PR Review Template

Use this checklist for pull requests that touch database code, schema, policies, manifests, or AI
tooling.

```markdown
## Database safety

- [ ] I ran `go test ./...`.
- [ ] I ran `go run ./cmd/goquent review --fail-on high ./...`.
- [ ] I included relevant `QueryPlan` or review output for changed database code.
- [ ] Any `RiskLow` result is treated only as structural DB risk, not business approval.
- [ ] Any `partial` or `unsupported` static review output is called out with manual review evidence.

## Policy and tenant safety

- [ ] Tenant-scoped queries include the tenant filter.
- [ ] Soft-delete behavior is intentional.
- [ ] Required filters are present.
- [ ] PII selection is avoided or includes a narrow access reason.

## Migrations

- [ ] I ran `go run ./cmd/goquent migrate plan <migration.sql>`.
- [ ] I ran `go run ./cmd/goquent migrate dry-run <migration.sql>`.
- [ ] Destructive or high-risk migration steps have an explicit approval reason.
- [ ] Preflight checks are documented for data-loss or locking risks.

## Manifest and AI context

- [ ] I ran `go run ./cmd/goquent manifest verify --manifest goquent.manifest.json ...`.
- [ ] The manifest is fresh, or this PR regenerates it.
- [ ] AI-generated reads use OperationSpec where possible.
- [ ] Static review findings marked `partial` or `unsupported` are not described as safe.

## Raw SQL

- [ ] Raw SQL is necessary and the reason is documented.
- [ ] User input is parameterized, not concatenated into SQL text.
- [ ] Tenant scope, soft delete, PII, and required-filter handling are visible in the SQL or review artifact.
- [ ] Raw SQL review output or manual review evidence is attached.

## Suppressions

- [ ] Every suppression has a reason.
- [ ] Temporary suppressions have an expiration date.
- [ ] Non-suppressible findings were fixed instead of hidden.
```
