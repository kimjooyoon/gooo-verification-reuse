#!/usr/bin/env bash
set -Eeuo pipefail

if [ "$#" -ne 6 ]; then
  echo "usage: ci-conformance.sh REPOSITORY BINARY SUBJECT_SHA OUTPUT BUILD_OBSERVATION TEST_OBSERVATION" >&2
  exit 64
fi

repository=$1
binary=$2
subject_sha=$3
output=$4
build_observation=$5
test_observation=$6

before=$(git -C "$repository" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
source="examples/verification-reuse/main.gooo"
contract="contracts/verification-reuse-denominator-v1.json"
corpus="fixtures/reuse-cases-v1.json"

run_case() {
  local scenario=$1
  local case_output="$output/$scenario"
  "$binary" run \
    --source "$source" \
    --contract "$contract" \
    --corpus "$corpus" \
    --scenario "$scenario" \
    --tree-root "$repository" \
    --out "$case_output" \
    --build-observation "$build_observation" \
    --test-observation "$test_observation" \
    --subject-sha "$subject_sha"
  jq -e '
    .schema == "gooo/verification-reuse/verification-receipt/v1" and
    .summary.total == 12 and
    .authority.repository_writes == 0 and
    .authority.local_test_executions == 0 and
    .authority.cross_project_required_gates == 0 and
    .inventory.root_readme_excluded == true and
    ([.cells[] | select(.state == "UNKNOWN") | .unknown |
      (.stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by | type) == "array")] | all) and
    ([.. | objects | keys[]? | select(test("percent|percentage|score|average|total_wall|improvement"; "i"))] | length) == 0
  ' "$case_output/verification-receipt.json" >/dev/null
}

mkdir -p "$output"
run_case valid-reuse-plan
run_case cache-hit-only
run_case source-tree-mismatch
run_case negative-executed-duration
run_case failed-prior-result
run_case policy-conflict
run_case mismatch-over-missing

jq -e '
  .decision == "CLOSED" and
  .claim.state == "CLOSED" and
  .summary == {total:12,closed:12,unknown:0,refuted:0} and
  .reuse.plan_status == "CLOSED" and .reuse.authorized == true and
  .reuse.cache_hit == true and .reuse.plan_only == true and
  .reuse.actual_reused == 0 and .reuse.consumer_test_executions == 0 and
  .execution_counts == {executed:2,reused:0,skipped:1,not_observed:0} and
  ([.operations[] | select(.status == "EXECUTED") | (.wall_ms != null and .peak_rss_kib != null and .wall_ms >= 0)] | all)
' "$output/valid-reuse-plan/verification-receipt.json" >/dev/null

jq -e '
  .decision == "UNKNOWN" and .claim.state == "UNKNOWN" and
  .cache_hit == true and .reuse.actual_reused == 0 and
  .cells[0].state == "UNKNOWN" and .cells[0].unknown.blocked_by == ["source_digest"]
' "$output/cache-hit-only/verification-receipt.json" >/dev/null

jq -e '.decision == "REFUTED" and .claim.reason == "SOURCE_DIGEST_MISMATCH" and .cache_hit == true and .reuse.actual_reused == 0' "$output/source-tree-mismatch/verification-receipt.json" >/dev/null
jq -e '.decision == "REFUTED" and .claim.reason == "NEGATIVE_EXECUTED_DURATION" and any(.operations[]; .wall_ms == -1)' "$output/negative-executed-duration/verification-receipt.json" >/dev/null
jq -e '.decision == "REFUTED" and .claim.reason == "PRIOR_TERMINAL_RESULT_FAILED" and .prior_bindings.prior_terminal_result == "FAIL"' "$output/failed-prior-result/verification-receipt.json" >/dev/null
jq -e '.decision == "REFUTED" and .claim.reason == "POLICY_IDENTITY_MISMATCH" and .cache_hit == true' "$output/policy-conflict/verification-receipt.json" >/dev/null
jq -e '.decision == "REFUTED" and .claim.reason == "SOURCE_DIGEST_MISMATCH" and any(.cells[]; .cell_id == "POLICY_BINDING" and .state == "UNKNOWN")' "$output/mismatch-over-missing/verification-receipt.json" >/dev/null

jq -s -S '{schema:"gooo/verification-reuse/ci-summary/v1",cases:map({scenario,decision,summary,execution_counts,reuse}),authority:{repository_writes:0,local_test_executions:0,cross_project_required_gates:0},root_readme_excluded:true}' \
  "$output"/*/verification-receipt.json > "$output/ci-summary.json"

after=$(git -C "$repository" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"
test "$(find "$output" -type f | wc -l | tr -d ' ')" -ge 35
