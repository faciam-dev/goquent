# AI Instructions for Goquent

Use this template when asking an AI coding agent to write or review Goquent database code.

```text
When writing database code with Goquent:
1. Treat Goquent as a deterministic database review boundary, not business approval.
2. Verify the manifest against current schema and policy inputs before relying on it.
3. Prefer OperationSpec for supported read-only select operations; otherwise use Goquent DSL.
4. Review QueryPlan output before execution or PR approval.
5. Never bypass tenant scope, soft delete, PII, or required-filter policy silently.
6. Do not use raw SQL unless it is necessary, parameterized, policy-reviewed, and documented.
7. For migrations, run migrate plan and dry-run before any apply path.
8. Treat RiskLow as low structural DB risk, not business approval.
9. If static review is partial or unsupported, do not claim the query is safe.
10. Use MCP only for read-only context and review tooling.
11. Do not let AI agents operate production databases autonomously.
```

Expanded guidance:

- Use `OperationSpec` for AI-proposed reads when possible.
- Compile OperationSpec with `--require-fresh-manifest` in CI or review workflows.
- If selecting PII, include a narrow `access_reason`.
- If a policy warning appears, fix the query rather than suppressing it.
- Suppressions must include a reason and should include an owner and expiration.
- MCP tools are read-only; do not ask MCP to apply migrations, execute SQL, or perform DB writes.
- MCP prompts are guidance only; they do not write files, approve destructive operations, or apply migrations.
- For migration changes, attach `go run ./cmd/goquent migrate plan <migration.sql>` and dry-run output to the PR.
- For source changes, attach `go run ./cmd/goquent review --format pretty` output or GitHub annotations.

Concrete command shape from the repository root:

```bash
go run ./cmd/goquent manifest verify \
  --manifest goquent.manifest.json \
  --schema schema.json \
  --policy policies.json
go run ./cmd/goquent review --fail-on high ./...
go run ./cmd/goquent migrate plan ./migrations/001_change.sql
go run ./cmd/goquent migrate dry-run ./migrations/001_change.sql
```
