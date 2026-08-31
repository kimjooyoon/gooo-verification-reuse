#!/usr/bin/env bash
set -Eeuo pipefail

if [ "$#" -lt 5 ]; then
  echo "usage: measure-command.sh METRICS RESULT LOG OPERATION_ID COMMAND [ARGS...]" >&2
  exit 64
fi

metrics=$1
result_path=$2
log_path=$3
operation_id=$4
shift 4

time_output="${metrics}.time"
mkdir -p "$(dirname "$metrics")" "$(dirname "$result_path")" "$(dirname "$log_path")"
status=0
if LC_ALL=C /usr/bin/time -f '%e %M' -o "$time_output" "$@" >"$log_path" 2>&1; then
  status=0
else
  status=$?
fi

seconds="0"
rss="0"
read -r seconds rss < "$time_output"
wall_ms=$(awk -v seconds="$seconds" 'BEGIN { printf "%d", (seconds * 1000) + 0.5 }')
peak_rss_kib=$(awk -v rss="$rss" 'BEGIN { printf "%d", rss + 0 }')
result_digest=""
if [ -f "$result_path" ]; then
  result_digest="sha256:$(sha256sum "$result_path" | awk '{print $1}')"
fi
terminal_result="PASS"
if [ "$status" -ne 0 ]; then
  terminal_result="FAIL"
fi

jq -S -n \
  --arg operation_id "$operation_id" \
  --arg status "$([ "$status" -eq 0 ] && echo EXECUTED || echo NOT_OBSERVED)" \
  --arg clock_domain "linux/amd64/github.actions.monotonic.v1" \
  --arg result_digest "$result_digest" \
  --arg terminal_result "$terminal_result" \
  --argjson wall_ms "$wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  '{status:$status,operation_id:$operation_id,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,clock_domain:$clock_domain,result_digest:$result_digest,terminal_result:$terminal_result}' \
  > "$metrics"

if [ "$status" -ne 0 ]; then
  cat "$log_path" >&2
  exit "$status"
fi
