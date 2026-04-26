# Benchmark Toolchain Policy

## Scope

This policy defines a reproducible command path for benchmark comparison in Skoll.

## Benchstat Command Path

In this repository environment, use a pinned Go toolchain when running benchstat:

```bash
GOTOOLCHAIN=go1.25.9 go run golang.org/x/perf/cmd/benchstat@latest artifacts/bench_before.txt artifacts/bench_after.txt
```

Rationale:

- `benchstat@latest` may require a newer Go toolchain than the local default.
- Pinning `GOTOOLCHAIN` avoids command drift between contributors.

## Fallback Evidence Strategy

If benchstat is unavailable in a constrained environment, compare benchmark medians and sample ranges, and record:

1. sampling command
2. baseline values
3. current values
4. delta percentage
