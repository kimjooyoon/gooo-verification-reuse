# gooo-verification-reuse

This repository defines a fail-closed, deterministic protocol for deciding
whether verification evidence may be reused. The first release emits a reuse
authorization plan; it does not suppress the CI build or test that produces
the evidence. That separation keeps the safety argument independent of the
optimization it may later authorize.

The implementation follows one metaprogramming chain:

`main.gooo` → semantic IR → generated Go evaluator artifact → machine receipt → human report

Reuse requires explicit equality for the source and tree digests, generated
artifact digest, Go/toolchain identity, command semantics, dependency inputs,
fixture/corpus identity, policy identity, platform/clock domain, and prior
terminal result. A cache hit is only an input to this decision. It is never
proof by itself.

All claims are `CLOSED`, `UNKNOWN`, or `REFUTED`, with `REFUTED` taking
precedence. Missing bindings carry a stage, step, reason, unknown class, next
operation, and minimal blocked-by frontier. Known identity mismatches,
negative executed durations, failed prior results, and policy conflicts are
`REFUTED`.

CI is the verification authority. It records exact integer operation counts and
per-operation wall/RSS observations. Generated outputs are written only to a
caller-owned temporary directory. The root README is excluded from the
inventory and tree digest; repository writes, local test executions, and
required cross-project gates are recorded as zero.
