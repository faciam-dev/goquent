# Benchmark Results

```
$ go test -bench . ./tests
```

The benchmark shows scanning maps and structs roughly 1.5x faster than GORM in the same environment.
