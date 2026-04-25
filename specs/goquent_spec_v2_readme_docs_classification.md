# Goquent spec v2: README/docs classification for AI-safe ORM positioning

このメモは `specs/goquent_ai_safe_orm_roadmap_v2.md` の仕様意図を、README / docs に載せる範囲へ落とすための分類である。

## Classification policy

- **A: Implemented/current feature.** README の主要導線に書いてよい。
- **B: Implemented, but human-controlled only.** 実装済みでも AI / MCP workflow の導線には載せない。
- **C: Planned/future.** 現時点の README 主要導線には書かない。
- **D: Explicit non-goal/prohibited.** 書く場合は禁止事項・trust boundary としてのみ書く。
- **E: Ambiguous.** spec を修正して明確化するまで current feature として書かない。

原則として、README の主要導線に載せるのは **A** のみ。**B** は human-controlled path の補足、**C** は roadmap docs、**D** は禁止事項、**E** は spec 修正後に再判定する。

| Feature | Classification | README treatment | Docs treatment | Notes |
|---|---:|---|---|---|
| MCP 経由の DB write | D | 禁止事項としてのみ書く。current MCP feature として書かない。 | MCP guide の safety / prohibited actions に明記する。 | `read-only default` ではなく、現時点の MCP scope は read-only と表現する方がよい。 |
| MCP 経由の migration apply | D | 禁止事項としてのみ書く。 | MCP guide と migration guide に「MCP は apply しない」と明記する。 | migration apply は human-controlled deployment path であり、MCP tool ではない。 |
| MCP 経由の raw SQL execution | D | 禁止事項としてのみ書く。 | raw SQL guide / MCP guide に禁止事項として書く。 | `review_query` や `explain_query` はよいが、execution は不可。 |
| AI coding agent による本番 DB 操作の自律実行 | D | README の trust boundary に明記する。 | AI agent playbook の禁止事項に入れる。 | Goquent は AI に DB 実行権限を与える ORM ではない。 |
| OperationSpec の join / aggregate / subquery | C | current feature として書かない。必要なら “not supported in current OperationSpec MVP” とだけ書く。 | OperationSpec guide の out-of-scope / future extension に置く。 | 永久禁止ではないが、MVP/current では非対応。 |
| OperationSpec の insert / update / delete | C, with D boundary | current feature として書かない。AI workflow には載せない。 | OperationSpec guide では out-of-scope。AI/MCP 経由の write 実行は D として明記する。 | 構造化 mutation interface 自体は将来検討余地があるが、AI の自律 write 実行とは分離する。 |
| `goquent migrate apply` | B | Quick start / AI workflow には載せない。載せるなら “human-controlled deployment path” の補足に限定する。 | Migration guide に、plan / dry-run / approval / preflight 後の人間操作として説明する。 | 実装済みでも AI-safe ORM の主要導線は `plan`, `dry-run`, `review`。 |
| `goquent migrate apply --approve` | B | README では主要導線にしない。migration section の補足に限定する。 | Approval reason の audit/context として説明する。 | `--approve` は business approval を意味しない。人間が理由を記録するための mechanism。 |
| `goquent review --config` | C | 現時点では書かない。 | 実装が reserved なら future/reserved として扱う。 | flag が存在しても config loading / validation / tests がなければ current feature ではない。 |
| `goquent review --write-baseline` | C | 書かない。 | optional / future adoption aid として roadmap に留める。 | baseline は review の信頼性を下げやすいため、実装・運用ルールが固まるまで README に出さない。 |
| SARIF output | C | 書かない。 | CI docs の future/optional に留める。 | 実装済みであっても主要導線は pretty / JSON / GitHub annotation で十分。SARIF は補助。 |
| `go install` など CLI install / distribution 手順 | E | 実際に検証済みの install command だけ書く。未確認なら書かない。 | Installation docs で package path / versioning / release policy を明確化する。 | spec v2 が distribution を定義していないため、実装確認または spec 追記が必要。 |
| `db.From(User).Where(user.ID.Eq(...))` のような typed DSL | E | 現在の public API と完全一致しない限り書かない。 | API guide では実装済み API のみ使用。roadmap docs では “non-normative API sketch” と明記する。 | spec の Example API Direction が current API に見えるため要修正。 |
| schema diff / DB introspection 由来の MigrationPlan | C | current feature として書かない。 | Migration docs では、実装済みなら SQL migration review を中心にする。schema diff / introspection は future/optional。 | Phase 5/6 では task/option 色が強い。 |
| Manifest の generated-code fingerprint | A if implemented; otherwise C | manifest verify / stale detection の一部として高レベルに書いてよい。 | Manifest guide で詳細説明する。 | README では fingerprint 名の詳細より「stale manifest を検知できる」を中心にする。 |
| Manifest の database fingerprint | C / optional | current required feature として書かない。DB 接続時の optional check としてのみ触れる。 | Manifest stale detection docs で optional database check として説明する。 | `database_fingerprint` は optional として扱う。未設定時に stale 扱いするかは config 次第。 |
| raw SQL の推奨度と README での扱い | B | 推奨経路として書かない。escape hatch として短く説明し、review 必須にする。 | raw SQL guide / AI playbook で使用条件と review 手順を明記する。 | raw SQL は human-controlled escape hatch。AI/MCP から raw execution しない。 |
| RiskLow の README 上の表現 | A, documentation boundary | README の trust boundary に必ず入れる。 | Risk guide / AI playbook に明記する。 | “low structural DB risk, not business approval”。「安全」と書かない。 |
| static review が `partial` / `unsupported` の場合の扱い | A, documentation boundary | README の trust boundary に短く入れる。 | static review guide / AI playbook に詳細を書く。 | partial/unsupported は pass ではない。AI は「安全」と言ってはいけない。 |
| MCP の `propose_repository_method` や `write_safe_migration` prompt の表現 | E | README には current feature として書かない。 | MCP docs では、実装済みなら “proposal/review prompt only; no file write, no DB apply” と明記。 | `write_safe_migration` は誤読されやすい。`review_migration_change` / `propose_migration_plan` などへ改名推奨。 |
| README に `migrate apply` を載せるべきか | B | 主要導線・AI workflow には載せない。載せるなら “Human-controlled migration deployment” の補足に限定。 | Migration guide で plan / dry-run / approval / preflight 後の手順として扱う。 | README の first-run story は `review`, `manifest verify`, `migrate plan/dry-run` を中心にする。 |

## Decisions for the disputed points

### `goquent migrate apply`

`goquent migrate apply` が CLI として実装済みでも、README の主要導線には入れない。AI-safe ORM の入口では、migration を **適用すること** ではなく、migration を **plan / dry-run / review / approval reason / preflight** で検証可能にすることを強調する。

README に載せる場合は、以下のような文脈に限定する。

```text
Migration application is a human-controlled deployment step. AI agents and MCP tools must not run `goquent migrate apply`. Before a human applies a migration, include `goquent migrate plan`, dry-run output, approval reason, and preflight notes in the PR.
```

### MCP write tools

現時点の README では、将来 write tools を持つ可能性を示唆しない。`read-only default` は “設定で write にできる” と読まれる可能性があるため、README では以下のように書く。

```text
The current MCP server is read-only. It exposes schema, manifest, policy, and review context. It does not perform DB writes, migration apply, or raw SQL execution.
```

### OperationSpec scope

README では OperationSpec を current feature として書く場合、次の範囲に限定する。

```text
The current OperationSpec scope is read-only, single-model select operations with explicit fields, filters, ordering, and limit.
```

join / aggregate / subquery は future/optional。insert / update / delete は current AI/MCP workflow には入れない。mutation OperationSpec を将来検討する場合も、AI の自律 DB write とは別仕様として扱う。

### `goquent review --config`

CLI に flag が存在しても、実装が reserved なら current feature ではない。README には書かない。config loading / validation / tests / examples がそろった時点で A に昇格する。

### typed DSL examples

`Example API Direction` にある typed DSL は、現在の public API と一致しない限り README/docs から除外する。spec 側では “non-normative sketch” と明記する。

## Suggested spec v2 edits

### 1. Add a documentation status taxonomy near Section 0

```md
### 0.3 Documentation status taxonomy

This roadmap contains current features, human-controlled features, future ideas, prohibited boundaries, and non-normative API sketches.

README and user-facing docs must follow this rule:

- Current features may be documented as product capabilities only when implemented, tested, and represented by examples.
- Human-controlled features, such as migration apply and raw SQL escape hatches, must not be included in AI or MCP workflows.
- Future or optional features must not appear in the README main path.
- Prohibited actions may be documented only as safety boundaries.
- API sketches are not public API until the implementation matches them.
```

### 2. Strengthen the MCP boundary

Replace “MCP is read-only by default” with:

```md
### 2.10 MCP current scope is read-only

The current MCP scope is read-only. It exists to provide schema, manifest, policy, and review context to AI coding agents.

Current MCP tools must not:

- perform DB writes
- apply migrations
- execute raw SQL
- grant approval for destructive operations
- bypass stale manifest checks

README must not imply that MCP write tools are available or planned as part of the current workflow. Any future write-capable MCP tool would require a separate safety specification and explicit opt-in design.
```

### 3. Clarify migration apply

Add to Phase 5:

```md
#### Migration apply is human-controlled

`goquent migrate apply` is a human-controlled deployment command, not an AI-agent workflow and not an MCP tool.

The README primary migration path should emphasize:

1. `goquent migrate plan`
2. dry-run
3. risk review
4. approval reason for high/destructive changes
5. preflight checks
6. human-controlled apply, if the project uses Goquent for application

AI agents may prepare or review migration artifacts, but must not run migration apply.
```

### 4. Clarify OperationSpec current scope

Replace the OperationSpec MVP scope wording with:

```md
#### Current OperationSpec scope

The current/MVP OperationSpec is intentionally narrow:

- read-only `select` only
- single model only
- explicit select fields
- filters
- order by
- limit

The following are not current OperationSpec features and must not be documented as current README capabilities:

- insert / update / delete
- join
- aggregate
- group by / having
- subquery
- raw SQL
- CTE

These are future design topics, not current product capabilities. Mutation OperationSpec requires a separate safety design and must not be conflated with AI autonomous DB execution.
```

### 5. Mark Example API Direction as non-normative

Rename Section 10 to:

```md
# 10. Non-normative API Direction

The examples in this section are design sketches. They are not public API unless the implementation and examples in the repository match them exactly.

README, docs, and examples must use only implemented public API. Do not copy this section into user-facing docs as current usage.
```

### 6. Clarify reserved CLI flags and optional outputs

Add to Phase 4:

```md
#### Reserved and optional CLI capabilities

A CLI flag is a current documented feature only when it is implemented, validated by tests, and shown in examples.

- `goquent review --config` is future/reserved until config loading and validation are fully implemented.
- `goquent review --write-baseline` is optional/future and must not appear in the README main path until its adoption rules are specified.
- SARIF output is optional/future unless implemented and tested. Even when implemented, it belongs in CI docs rather than the README primary path.
```

### 7. Clarify manifest fingerprints

Add to Phase 6:

```md
#### Fingerprint status

Generated-code and policy fingerprints are part of the core stale-detection model when manifest verification is implemented.

Database fingerprint is optional because it requires live database access. If no database connection is configured, manifest verification must report that the database check was not performed. Whether that is acceptable is controlled by configuration.

README should describe this as stale manifest detection, not as a guarantee that the live database always matches.
```

### 8. Clarify raw SQL as an escape hatch

Add to the Risk / Policy section:

```md
#### Raw SQL is an escape hatch

Raw SQL is allowed only as a human-controlled escape hatch when Goquent's structured API cannot express the operation adequately.

Raw SQL must:

- produce a QueryPlan or review artifact when possible
- emit `RAW_SQL_USED`
- include a reason when required by configuration
- be included in PR review output

Raw SQL execution is not part of the AI/MCP workflow.
```

### 9. Rename or constrain MCP prompts

In Phase 8, change the prompt/tool wording to avoid implying autonomous code or migration execution:

```md
MCP prompts may help draft or review code changes, but they do not write files, apply migrations, execute SQL, or approve destructive operations.

Prefer names such as:

- `draft_repository_method_guidance`
- `review_database_change`
- `propose_migration_plan`
- `review_migration_risk`

Avoid names such as `write_safe_migration` unless the docs explicitly state that the prompt only produces reviewable text and never applies migrations.
```

