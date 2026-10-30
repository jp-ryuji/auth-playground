#!/usr/bin/env bash
# Print the first pending signup requirement. Exit 0 in every case.
# "NEXT none" means the signup suite has no pending() calls left.
set -euo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
file="$root/apps/api/internal/signup/signup_test.go"

parsed=$(awk '
  /^func Test/ {
    name = $2
    sub(/\(.*/, "", name)
  }
  /pending\(t, "/ && !found {
    found = 1
    line = $0
    if (match(line, /pending\(t, "[^"]+"/)) {
      id = substr(line, RSTART, RLENGTH)
      sub(/^pending\(t, "/, "", id)
      sub(/"$/, "", id)
    }
    test = name
  }
  END {
    if (!found) {
      print "NONE"
      exit 0
    }
    print id
    print test
  }
' "$file")

id=$(printf '%s\n' "$parsed" | awk 'NR==1 { print; exit }')
if [ "$id" = "NONE" ]; then
  echo "NEXT none"
  echo "suite: apps/api/internal/signup/signup_test.go"
  echo "note: no pending() calls remain. If signup-first-login.md is still Draft, promote it to Stable."
  exit 0
fi

test=$(printf '%s\n' "$parsed" | awk 'NR==2 { print; exit }')
id_lower=$(printf '%s' "$id" | tr '[:upper:]' '[:lower:]')

echo "NEXT $id"
echo "test: $test"
case "$id" in
  SIGNUP-*)
    echo "spec: docs/specs/10-flows/signup-first-login.md#$id_lower"
    ;;
  ACCEPT-*)
    echo "spec: docs/specs/10-flows/signup-first-login.md"
    ;;
  *)
    echo "spec: docs/specs/"
    ;;
esac

plan=$(find "$root/docs/plans" -name "*${id_lower}*" -print 2>/dev/null | head -n 1 || true)
if [ -n "$plan" ]; then
  echo "plan: ${plan#"$root"/}"
fi
echo "state: docs/loops/state.md"
echo "proof: make prove PROVE_ID=$id"
