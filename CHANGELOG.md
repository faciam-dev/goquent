# Changelog

## Unreleased
- `Where` no longer attempts to automatically detect dotted values as column names.
  Use `WhereColumn` for column-to-column comparisons.
- Added generic `SelectOne` and `SelectAll` APIs supporting struct and map destinations.
- Added generic `Insert`, `Update`, and `Upsert` helpers with unified struct and map support.
- Added `PK` option to configure primary key columns for map writes.
- Added boolean dialect compatibility with configurable `BoolScanPolicy` and field tags
  `boolstrict`/`boollenient`.
