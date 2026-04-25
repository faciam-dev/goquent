# Goquent Vision

Goquent is a small ORM with an AI-safe database review boundary for both humans and AI coding
agents.
The core idea is simple: database intent should be inspectable before it is executed.

Traditional ORM code often hides risk inside builder chains, raw SQL strings, or migrations.
Goquent keeps the normal query-builder workflow, but adds reviewable artifacts:

- `QueryPlan` explains SQL, params, touched tables, predicates, risk, and warnings.
- `RiskEngine` classifies structural database risk deterministically.
- Policy metadata catches tenant, soft-delete, required-filter, and PII mistakes.
- `goquent review` makes those checks available in CI and pull requests.
- `MigrationPlan` reviews schema changes before apply.
- `Manifest` exports schema and policy context that AI tools can use as deterministic review
  input.
- `OperationSpec` gives AI tools a narrow read-only interface instead of free-form SQL.
- The MCP server exposes context and review tools without DB write capability.

Goquent does not try to prove business correctness. `RiskLow` means the database operation shape
looks low-risk to Goquent. It does not mean the user is authorized, the product behavior is correct,
or the change has been approved.

The intended workflow is:

1. Build the query or migration.
2. Generate a plan.
3. Review risk and policy warnings.
4. Require approval for high or destructive changes.
5. Keep suppressions narrow, reasoned, and expiring.
6. Run the same checks in CI.
7. Give AI tools current manifest context and read-only review tools.

This keeps Goquent useful as a regular Go ORM while making AI-generated database work reviewable
and bounded.
