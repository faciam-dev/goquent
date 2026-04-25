# Examples

The runnable AI-safe ORM example lives in [`../examples/ai-safe-orm`](../examples/ai-safe-orm).

It demonstrates:

- simple CRUD planning without database execution.
- tenant-scoped SaaS filters.
- soft-delete default filtering.
- PII warning with access reason.
- warning suppression with owner and expiration.
- migration review.
- manifest generation and verification inputs.
- OperationSpec compilation.
- CI review commands.
- MCP integration commands.

Run it:

```bash
go run ./examples/ai-safe-orm
```

Review the example project:

```bash
go run ./cmd/goquent review --format pretty --fail-on blocked ./examples/ai-safe-orm
go run ./cmd/goquent migrate plan ./examples/ai-safe-orm/migrations/001_add_users.sql
go run ./cmd/goquent migrate plan ./examples/ai-safe-orm/migrations/002_drop_legacy_email.sql
go run ./cmd/goquent manifest --format json \
  --schema ./examples/ai-safe-orm/schema.json \
  --policy ./examples/ai-safe-orm/policies.json
```

The example intentionally contains review findings so the output is visible; `--fail-on blocked`
keeps the walkthrough command runnable.

Verify the checked-in manifest:

```bash
go run ./cmd/goquent manifest verify \
  --manifest ./examples/ai-safe-orm/goquent.manifest.json \
  --schema ./examples/ai-safe-orm/schema.json \
  --policy ./examples/ai-safe-orm/policies.json
```

Compile the example OperationSpec:

```bash
go run ./cmd/goquent operation compile \
  --manifest ./examples/ai-safe-orm/goquent.manifest.json \
  --spec ./examples/ai-safe-orm/operation.json \
  --values ./examples/ai-safe-orm/values.json
```

The checked-in `goquent.manifest.json` is intentionally small and should be regenerated when the
example schema or policies change.
