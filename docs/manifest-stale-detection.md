# Manifest Stale Detection

AI tools should not rely on stale schema or policy context. Goquent tracks freshness with stable
fingerprints for schema, policy, generated code, and optional database schema.

Verify a stored manifest against current inputs:

```bash
go run ./cmd/goquent manifest verify \
  --manifest goquent.manifest.json \
  --schema schema.json \
  --policy policies.json \
  --code ./orm
```

JSON output is available for automation:

```bash
go run ./cmd/goquent manifest verify --format json \
  --manifest goquent.manifest.json \
  --schema schema.json \
  --policy policies.json
```

`goquent review` can surface stale manifests:

```bash
go run ./cmd/goquent review \
  --manifest goquent.manifest.json \
  --require-fresh-manifest \
  ./...
```

Exit behavior:

- `manifest verify` returns `0` when fresh and `1` when stale.
- `review --require-fresh-manifest` returns `3` when the manifest is stale.

MCP exposes freshness through `goquent://manifest` and `goquent://manifest-status`.

Recommended CI gate:

```bash
go run ./cmd/goquent manifest verify \
  --manifest goquent.manifest.json \
  --schema schema.json \
  --policy policies.json \
  --code ./orm
go run ./cmd/goquent review --manifest goquent.manifest.json --require-fresh-manifest ./...
```

If verification fails, regenerate the manifest or update the schema/policy inputs before asking AI
tools to generate database code.
