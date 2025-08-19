# Changelog

## Unreleased
- `Where` no longer attempts to automatically detect dotted values as column names.
  Use `WhereColumn` for column-to-column comparisons.
- Added generic `SelectOne` and `SelectAll` APIs supporting struct and map destinations.
- Added generic `Insert`, `Update`, and `Upsert` helpers with unified struct and map support.
