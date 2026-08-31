# Verification-evidence reuse protocol v1

## Decision

The protocol makes reuse an evidence decision, not a cache decision. A prior
terminal result may be reused only when every identity below is explicitly
present in both the current observation and the prior receipt:

1. source digest;
2. source-tree digest;
3. generated evaluator artifact digest;
4. Go/toolchain identity;
5. command semantics;
6. dependency inputs;
7. fixture/corpus identity;
8. policy identity;
9. platform and clock domain; and
10. prior terminal result.

The cache-hit bit is recorded, but is never proof. In v1 the authorization is
plan-only. CI still executes its build and test operations, and the reuse
operation is recorded as `SKIPPED` with no invented duration. This makes the
safety evidence independent of test suppression.

## Resolution

Each denominator cell is exactly `CLOSED`, `UNKNOWN`, or `REFUTED`. Resolution
uses `REFUTED > UNKNOWN > CLOSED`. Missing bindings are `UNKNOWN` and carry
`stage`, `step`, `reason`, `unknown_class`, `next_operation`, and
`blocked_by`. The blocked-by list is the smallest known causal frontier for
that decision. Known identity mismatch, negative executed duration, failed
prior result, and execution-status policy conflict are `REFUTED`.

## Fixed denominator

The source contains exactly twelve one-to-one activities bound to the fixed
contract:

| # | Cell | Stage | Proof | Indicator |
|---:|---|---|---|---|
| 1 | SOURCE_TREE_BINDING | BINDING | FOUNDATION | DRIVER |
| 2 | GENERATED_ARTIFACT_BINDING | BINDING | FOUNDATION | DRIVER |
| 3 | TOOLCHAIN_BINDING | BINDING | FOUNDATION | DRIVER |
| 4 | COMMAND_BINDING | BINDING | FOUNDATION | OUTCOME |
| 5 | DEPENDENCY_BINDING | BINDING | COHERENCE | OUTCOME |
| 6 | FIXTURE_BINDING | BINDING | COHERENCE | DRIVER |
| 7 | POLICY_BINDING | BINDING | COHERENCE | GUARDRAIL |
| 8 | PLATFORM_CLOCK_BINDING | BINDING | COHERENCE | GUARDRAIL |
| 9 | PRIOR_RESULT_BINDING | BINDING | REGRESSION | GUARDRAIL |
| 10 | REUSE_PLAN | REUSE | REGRESSION | OUTCOME |
| 11 | OPERATION_ACCOUNTING | OBSERVATION | REGRESSION | OUTCOME |
| 12 | HUMAN_REPORT | REPORT | REGRESSION | GUARDRAIL |

There are no scores, percentages, averages, or inferred improvement claims.
Build and test wall time and peak RSS are retained per operation and clock
domain. Unlike clock domains are never subtracted or aggregated. Exact counts
for `executed`, `reused`, `skipped`, and `not_observed` are emitted for every
scenario.

## CI cases

Actions evaluates one valid plan and six unsafe cases: cache-hit-only,
source-tree-mismatch, negative-executed-duration, failed-prior-result,
policy-conflict, and mismatch-over-missing. The last case proves that a known
refutation remains visible when a separate binding is missing.

All outputs are written beneath the caller-owned runner temporary directory.
The source tree is checked for mutation before and after evaluation. The root
README is excluded from inventory and tree digest. Repository writes, local
test executions, and required cross-project gates are all exact zero values;
other projects are not required inputs.
