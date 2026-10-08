#!/usr/bin/env bash
# Fixture tests for check-licenses.sh. No network: the real tools are
# replaced by stubs that print a canned scenario and exit
# with a canned code, so the test exercises this script's own logic — the
# allow-list comparison, the exceptions layer, and the icon check — not the
# real tools' classification. Running the real tools against the real trees
# is a separate manual check.
#
# Usage: scripts/check-licenses_test.sh   (exit 0 = all cases passed)

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$HERE/check-licenses.sh"
FIXTURES="$HERE/testdata/licenses"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT

passed=0; failed=0

# A minimal buildable backend (real permissive local dependency) so `go build`
# succeeds fast with no network; the stub LICHEN_CMD ignores the resulting
# binary's actual content and reports whatever scenario the case selects.
make_good_backend() {
  local r="$1"
  mkdir -p "$r/backend/cmd/server"
  # go.mod holds a placeholder: its replace directive needs an absolute path.
  sed "s#@FIXTURES@#$FIXTURES#g" "$FIXTURES/good/backend/go.mod" > "$r/backend/go.mod"
  cp -r "$FIXTURES/good/backend/cmd" "$r/backend/"
  [[ -f "$FIXTURES/good/backend/go.sum" ]] && cp "$FIXTURES/good/backend/go.sum" "$r/backend/go.sum"
}

# A minimal frontend tree: package.json/package-lock.json (content unused —
# the stub LICENSE_CHECKER_CMD ignores them) plus one icon usage.
make_frontend() {
  local r="$1" icon_line="$2"
  mkdir -p "$r/frontend/app" "$r/frontend/node_modules/@iconify/collections"
  cat > "$r/frontend/package.json" <<'EOF'
{"name": "frontend", "private": true}
EOF
  echo '{}' > "$r/frontend/package-lock.json"
  cat > "$r/frontend/app/Example.vue" <<EOF
<template>
  <Icon name="$icon_line" />
</template>
EOF
  cat > "$r/frontend/node_modules/@iconify/collections/collections.json" <<'EOF'
{
  "mdi": { "name": "Material Design Icons", "license": { "spdx": "Apache-2.0" } },
  "bad-icons": { "name": "Bad Icons", "license": { "spdx": "CC-BY-NC-4.0" } }
}
EOF
}

make_good_tree() {
  local r="$1"
  make_good_backend "$r"
  make_frontend "$r" "mdi:home"
  cp "$FIXTURES/../licenses/allowed.txt" "$r/allowed.txt" 2>/dev/null || true
}

# run_case <name> <expected exit> <expected output substring> <env assignments...> -- (no positional mutation; env drives the scenario)
run_case() {
  local name="$1" want_code="$2" want_text="$3"; shift 3
  local R="$WORK/case-$passed-$failed"
  rm -rf "$R"; make_good_tree "$R"
  local out code

  # Defaults, each overridable by a same-named KEY=VALUE in "$@". `env`
  # itself cannot do this merge (a later duplicate assignment on one `env`
  # command line always wins, so hardcoding the defaults after "$@" would
  # silently discard a caller's override, and a plain shell parameter
  # expansion like ${VAR:-default} can't see a value that only exists as an
  # argument to the same `env` invocation) — so the merge happens here, in
  # this shell, before env is ever invoked.
  local LICENSE_ROOT="$R"
  local LICHEN_CMD="$FIXTURES/stubs/stub-lichen.sh"
  local LICENSE_CHECKER_CMD="$FIXTURES/stubs/stub-license-checker.sh"
  local LICENSE_ALLOWED="$HERE/licenses/allowed.txt"
  local LICENSE_EXCEPTIONS="$WORK/empty-exceptions.txt"
  local -a passthrough=()
  for a in "$@"; do
    case "$a" in
      LICENSE_ROOT=*) LICENSE_ROOT="${a#*=}" ;;
      LICHEN_CMD=*) LICHEN_CMD="${a#*=}" ;;
      LICENSE_CHECKER_CMD=*) LICENSE_CHECKER_CMD="${a#*=}" ;;
      LICENSE_ALLOWED=*) LICENSE_ALLOWED="${a#*=}" ;;
      LICENSE_EXCEPTIONS_OVERRIDE=*) LICENSE_EXCEPTIONS="${a#*=}" ;;
      *) passthrough+=("$a") ;;
    esac
  done
  out="$(env "${passthrough[@]}" \
      LICENSE_ROOT="$LICENSE_ROOT" \
      LICHEN_CMD="$LICHEN_CMD" \
      LICENSE_CHECKER_CMD="$LICENSE_CHECKER_CMD" \
      LICENSE_ALLOWED="$LICENSE_ALLOWED" \
      LICENSE_EXCEPTIONS="$LICENSE_EXCEPTIONS" \
      "$CHECKER" 2>&1)"
  code=$?
  if [[ "$code" == "$want_code" && ( -z "$want_text" || "$out" == *"$want_text"* ) ]]; then
    printf 'ok   %s\n' "$name"; passed=$((passed+1))
  else
    printf 'FAIL %s\n     want exit %s containing "%s"; got exit %s\n     output: %s\n' \
      "$name" "$want_code" "$want_text" "$code" "$(printf '%s' "$out" | head -8 | tr '\n' '|')"
    failed=$((failed+1))
  fi
}

: > "$WORK/empty-exceptions.txt"

[[ -x "$CHECKER" ]] || { echo "FAIL: $CHECKER does not exist or is not executable (expected before implementation)"; exit 1; }

# ── Go side ──────────────────────────────────────────────────────────────────
run_case "good tree passes" \
  0 "OK: all licence checks passed" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-good.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "a GPL-3.0-only Go dependency is rejected and named" \
  1 "FAIL blocked-licence: fixture.example/gpldep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-gpl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "an AGPL-3.0-only Go dependency is rejected" \
  1 "FAIL blocked-licence: fixture.example/agpldep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-agpl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "an SSPL-1.0 Go dependency is rejected" \
  1 "FAIL blocked-licence: fixture.example/ssrvdep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-sspl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "a Go dependency with no recognizable licence is rejected" \
  1 "FAIL blocked-licence: fixture.example/nolicensedep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-nolicense.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "a component offered as MIT OR GPL-2.0 passes on the permissive option" \
  0 "OK: all licence checks passed" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-or-choice-permissive.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "a component offered as GPL-2.0 only is rejected" \
  1 "FAIL blocked-licence: fixture.example/gpl2dep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-gpl-only.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "an LGPL-3.0 Go dependency fails without an exception" \
  1 "FAIL blocked-licence: fixture.example/lgpldep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-lgpl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

echo "fixture.example/lgpldep * LGPL-3.0 # test exception" > "$WORK/lgpl-exception.txt"
run_case "an LGPL-3.0 Go dependency passes with a matching exception" \
  0 "OK: all licence checks passed" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-lgpl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0 \
  LICENSE_EXCEPTIONS_OVERRIDE="$WORK/lgpl-exception.txt"

echo "fixture.example/some-other-dep * LGPL-3.0 # test exception, wrong component" > "$WORK/lgpl-wrong-exception.txt"
run_case "an exception for a different component does not help" \
  1 "FAIL blocked-licence: fixture.example/lgpldep" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-lgpl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0 \
  LICENSE_EXCEPTIONS_OVERRIDE="$WORK/lgpl-wrong-exception.txt"

# ── npm side ─────────────────────────────────────────────────────────────────
run_case "an UNLICENSED npm package is rejected and named" \
  1 "FAIL blocked-licence: bad-pkg@1.0.0" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-good.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-unlicensed.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "an npm package offered as MIT OR GPL-2.0 passes on the permissive option" \
  0 "OK: all licence checks passed" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-good.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-or-choice.json" LICENSE_CHECKER_STUB_EXIT=0

run_case "an npm package offered as GPL-2.0 only is rejected" \
  1 "FAIL blocked-licence: gpl-pkg@1.0.0" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-good.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-gpl-only.json" LICENSE_CHECKER_STUB_EXIT=0

# ── Icons ────────────────────────────────────────────────────────────────────
R="$WORK/icon-disallowed"; rm -rf "$R"; make_good_tree "$R"
make_frontend "$R" "bad-icons:skull"
out="$(env LICENSE_ROOT="$R" LICHEN_CMD="$FIXTURES/stubs/stub-lichen.sh" LICENSE_CHECKER_CMD="$FIXTURES/stubs/stub-license-checker.sh" \
  LICENSE_ALLOWED="$HERE/licenses/allowed.txt" LICENSE_EXCEPTIONS="$WORK/empty-exceptions.txt" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-good.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0 \
  "$CHECKER" 2>&1)"; code=$?
if [[ "$code" == 1 && "$out" == *"FAIL blocked-licence"*"bad-icons"* ]]; then printf 'ok   %s\n' "an icon set with a disallowed licence is rejected"; passed=$((passed+1))
else printf 'FAIL an icon set with a disallowed licence is rejected (exit %s)\n     output: %s\n' "$code" "$(printf '%s' "$out" | head -5 | tr '\n' '|')"; failed=$((failed+1)); fi

R="$WORK/icon-missing"; rm -rf "$R"; make_good_tree "$R"
make_frontend "$R" "nonexistent:ghost"
out="$(env LICENSE_ROOT="$R" LICHEN_CMD="$FIXTURES/stubs/stub-lichen.sh" LICENSE_CHECKER_CMD="$FIXTURES/stubs/stub-license-checker.sh" \
  LICENSE_ALLOWED="$HERE/licenses/allowed.txt" LICENSE_EXCEPTIONS="$WORK/empty-exceptions.txt" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-good.txt" LICHEN_STUB_EXIT=0 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0 \
  "$CHECKER" 2>&1)"; code=$?
if [[ "$code" == 1 && "$out" == *"FAIL blocked-licence"*"nonexistent"* ]]; then printf 'ok   %s\n' "a used icon prefix missing from the collection is rejected"; passed=$((passed+1))
else printf 'FAIL a used icon prefix missing from the collection is rejected (exit %s)\n     output: %s\n' "$code" "$(printf '%s' "$out" | head -5 | tr '\n' '|')"; failed=$((failed+1)); fi

# ── Cross-cutting ────────────────────────────────────────────────────────────
R="$WORK/multi"; rm -rf "$R"; make_good_tree "$R"
out="$(env LICENSE_ROOT="$R" LICHEN_CMD="$FIXTURES/stubs/stub-lichen.sh" LICENSE_CHECKER_CMD="$FIXTURES/stubs/stub-license-checker.sh" \
  LICENSE_ALLOWED="$HERE/licenses/allowed.txt" LICENSE_EXCEPTIONS="$WORK/empty-exceptions.txt" \
  LICHEN_STUB_OUTPUT="$FIXTURES/scenarios/go-gpl.txt" LICHEN_STUB_EXIT=1 \
  LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-multi-fail.json" LICENSE_CHECKER_STUB_EXIT=0 \
  "$CHECKER" 2>&1)"; code=$?
n="$(printf '%s\n' "$out" | grep -c '^FAIL ')"
if [[ "$code" == 1 && "$n" -ge 3 ]]; then printf 'ok   %s\n' "reports every failure across Go and npm, not just the first"; passed=$((passed+1))
else printf 'FAIL reports every failure across Go and npm (exit %s, %s FAIL lines)\n' "$code" "$n"; failed=$((failed+1)); fi

run_case "a missing tool exits 2" \
  2 "" \
  LICHEN_CMD="/nonexistent/lichen-does-not-exist" \
  LICENSE_CHECKER_CMD="$FIXTURES/stubs/stub-license-checker.sh" LICENSE_CHECKER_STUB_OUTPUT="$FIXTURES/scenarios/npm-good.json" LICENSE_CHECKER_STUB_EXIT=0

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[[ "$failed" -eq 0 ]]
