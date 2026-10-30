#!/usr/bin/env bash
# Run the signup tests for one requirement ID.
# Exit 1 when a test fails or when a non-live test is skipped.
set -euo pipefail

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "usage: bash scripts/prove-requirement.sh SIGNUP-08" >&2
  exit 1
fi

id=$1
uscore=$(printf '%s' "$id" | tr '-' '_')
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
file="$root/apps/api/internal/signup/signup_test.go"

# SIGNUP-08 matches TestSignup_SIGNUP_08_...
# Acceptance IDs match a "// req ACCEPT-..." comment on the func line.
tests=$(awk -v id="$id" -v uscore="$uscore" '
  /^func Test/ {
    name = $2
    sub(/\(.*/, "", name)
    if (name ~ ("_" uscore "_") || name ~ ("_" uscore "$") || index($0, "// req " id)) {
      print name
    }
  }
' "$file")

if [ -z "$tests" ]; then
  echo "no signup test for $id" >&2
  exit 1
fi

pattern=$(printf '%s\n' "$tests" | paste -sd '|' -)
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

set +e
(
  cd "$root/apps/api"
  go test -count=1 -v -run "^(${pattern})$" ./internal/signup/...
) | tee "$tmp"
status=${PIPESTATUS[0]}
set -e

skipped=$(grep '^--- SKIP:' "$tmp" | grep -v 'Live' || true)
if [ -n "$skipped" ]; then
  echo "proof failed: $id is still skipped" >&2
  printf '%s\n' "$skipped" >&2
  exit 1
fi

if [ "$status" -ne 0 ]; then
  echo "proof failed: $id" >&2
  exit "$status"
fi

echo "proof passed: $id"
