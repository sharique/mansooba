#!/usr/bin/env bash
# Fixture tests for generate-notices.sh. No network: fixed lichen-template-style
# and license-checker-JSON fixture files stand in for the real tools.
#
# Usage: scripts/generate-notices_test.sh   (exit 0 = all cases passed)

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GENERATOR="$HERE/generate-notices.sh"
F="$HERE/testdata/licenses/notices-fixtures"

WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT
# The fixtures name licence files by absolute path, written as a placeholder.
for f in lichen.txt license-checker.json; do sed "s#@FIXTURES@#$HERE/testdata/licenses#g" "$F/$f" > "$WORK/$f"; done

passed=0; failed=0
ok()  { printf 'ok   %s\n' "$1"; passed=$((passed+1)); }
bad() { printf 'FAIL %s\n     %s\n' "$1" "${2:-}"; failed=$((failed+1)); }

[[ -x "$GENERATOR" ]] || { echo "FAIL: $GENERATOR does not exist or is not executable (expected before implementation)"; exit 1; }

run() {
  LICHEN_NOTICES_OUTPUT="$WORK/lichen.txt" \
  LICENSE_CHECKER_NOTICES_OUTPUT="$WORK/license-checker.json" \
  GO_STDLIB_LICENSE="$F/go-stdlib-LICENSE" \
  GO_STDLIB_VERSION="go1.27.1-test" \
  "$GENERATOR"
}

out1="$(run)"; code1=$?
golden="$(cat "$F/golden-THIRD-PARTY-NOTICES.md")"

if [[ "$code1" -eq 0 && "$out1" == "$golden" ]]; then
  ok "matches the golden file byte for byte"
else
  bad "matches the golden file byte for byte" "exit $code1; diff:
$(diff <(printf '%s' "$out1") <(printf '%s' "$golden") | head -20)"
fi

out2="$(run)"
if [[ "$out2" == "$out1" ]]; then
  ok "two runs give identical output"
else
  bad "two runs give identical output" "outputs differ"
fi

if [[ "$out1" == *"### MIT"* ]]; then
  n="$(printf '%s\n' "$out1" | grep -c '^### MIT$')"
  [[ "$n" -eq 1 ]] && ok "each distinct licence text appears once (MIT, shared by two components)" \
    || bad "each distinct licence text appears once" "MIT heading appeared $n times"
else
  bad "each distinct licence text appears once" "no MIT heading found"
fi

if [[ "$out1" == *"no-license-file-pkg"*"0.0.740"*"https://example.com/no-license-file-pkg"* ]]; then
  ok "the component without a licence file is listed with its licence id and repository URL"
else
  bad "the component without a licence file is listed with its licence id and repository URL"
fi

# It must NOT get its own "## Licence texts" subsection (nothing to show).
after_table="$(printf '%s\n' "$out1" | awk '/## Licence texts/{f=1} f')"
if [[ "$after_table" != *"no-license-file-pkg"* ]]; then
  ok "the component without a licence file has no text section"
else
  bad "the component without a licence file has no text section"
fi

if [[ "$out1" == *"go (standard library)"*"go1.27.1-test"*"BSD-3-Clause"* ]]; then
  ok "the Go standard library itself appears as a component"
else
  bad "the Go standard library itself appears as a component"
fi

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[[ "$failed" -eq 0 ]]
