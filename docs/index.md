# Goquent Documentation

Goquent is an AI-safe ORM for Go: a small ORM and query-builder layer with deterministic review artifacts for database code. Start with the README for the project overview, then use this index by role.

If you are an AI coding agent, start with the [AI agent playbook](./ai-agent-playbook.md). It is the operational checklist for repository methods, raw SQL, migrations, manifests, MCP usage, and PR review output.

## For Humans

- [QueryPlan](./query-plan.md): inspect generated SQL, params, tables, predicates, warnings, approval state, and analysis precision.
- [Risk engine](./risk-engine.md): understand structural risk levels and warning codes.
- [Policy DSL](./policy-dsl.md): define tenant scope, soft delete, PII, and required-filter policies.
- [Suppression and approval](./suppression-and-approval.md): document narrow suppressions and intentional risky operations.
- [Static review limits](./static-review-limits.md): interpret `precise`, `partial`, and `unsupported` review output.
- [ORM package API](./orm/README.md): package-level API reference.
- [Generic CRUD guide](./orm/generic-crud.md): typed helper usage and limitations.

## For AI Coding Agents

- [AI agent playbook](./ai-agent-playbook.md): required workflow and checklist before changing database code.
- [AI instructions](./ai-instructions.md): short prompt block for agent setup.
- [OperationSpec](./operation-spec.md): narrow read-only interface for AI-generated select operations.
- [Manifest](./manifest.md): schema and policy context for review and code generation.
- [Manifest stale detection](./manifest-stale-detection.md): freshness checks and stale-manifest behavior.
- [MCP server](./mcp.md): read-only context and review tools for AI editors.

## For CI And Review

- [Review CLI](./review-cli.md): `goquent review` flags, formats, and exit codes.
- [PR review template](./pr-review-template.md): checklist for DB code, policies, manifests, and migrations.
- [Static review limits](./static-review-limits.md): required handling for partial or unsupported analysis.
- [Examples](./examples.md): runnable commands for the AI-safe example project.

## For Migrations

- [MigrationPlan](./migration-plan.md): migration planning, dry-run validation, destructive-operation warnings, approval, and preflight guidance.
- [Manifest stale detection](./manifest-stale-detection.md): avoid planning from stale schema or policy context.
- [AI agent playbook: migrations](./ai-agent-playbook.md#migrations): agent workflow for migration PRs.

## Examples

- [AI-safe ORM example](../examples/ai-safe-orm): runnable safety workflow without a live database.
- [Quickstart example](../examples/quickstart/main.go): minimal ORM usage.

## Lower-Level ORM Guides

- [Driver](./orm/driver/README.md)
- [Model](./orm/model/README.md)
- [Query](./orm/query/README.md)
- [Scanner](./orm/scanner/README.md)
- [Conversion](./orm/conv/README.md)

## Trust Boundary

Goquent reviews the structure of database operations. It does not approve business intent. `RiskLow` is low structural database risk, not business approval. A stale manifest is untrusted. Static review marked `partial` or `unsupported` must not be described as safe.
