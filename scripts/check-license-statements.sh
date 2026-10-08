#!/usr/bin/env bash
# Every place that states this project's licence must agree, and no scanned
# document may still say the project is unlicensed or undecided.
#
# It does NOT judge whether the licence choice itself is right. It only
# checks consistency between the artifacts that state it: LICENSE, NOTICE, README.md, CONTRIBUTING.md, frontend/package.json's
# "license" field, backend/docs/swagger.json's info.license, and every
# Markdown file under docs/ (this is the code repository; the separate
# documentation repository is swept by hand).
#
# Read-only, no network. Run from anywhere: scripts/check-license-statements.sh
# Exit: 0 all checks passed, 1 at least one check failed, 2 the script could
# not run (a required file or directory is missing).
# VERIFY_ROOT overrides the tree to check (used by check-license-statements_test.sh).

set -uo pipefail

ROOT="${VERIFY_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURE_LICENSE="$HERE/testdata/license-statements/agpl-3.0-fixture.txt"
fails=0

fail() { printf 'FAIL %s: %s (%s)\n' "$1" "$2" "$3"; fails=$((fails+1)); }
rel() { printf '%s' "${1#"$ROOT"/}"; }

# Structural prerequisites: without these there is no mansooba tree to check
# at all, so the script cannot run rather than reporting a check failure.
[[ -d "$ROOT/backend" && -d "$ROOT/frontend" ]] || { echo "error: required directory missing (backend/ or frontend/) under $ROOT" >&2; exit 2; }
[[ -f "$FIXTURE_LICENSE" ]] || { echo "error: reference license text missing: $FIXTURE_LICENSE" >&2; exit 2; }

# ── LICENSE: present, and the verbatim FSF AGPL-3.0 text ───────────────────────
if [[ ! -f "$ROOT/LICENSE" ]]; then
  fail missing-file "LICENSE does not exist" "LICENSE"
else
  want_sha="$(sha256sum "$FIXTURE_LICENSE" | cut -d' ' -f1)"
  got_sha="$(sha256sum "$ROOT/LICENSE" | cut -d' ' -f1)"
  [[ "$got_sha" == "$want_sha" ]] || fail wrong-license-text "LICENSE does not match the verbatim AGPL-3.0 text (sha256 $got_sha, want $want_sha)" "LICENSE"
fi

# ── NOTICE: present, with the copyright line and licence identifier ────────────
notice="$ROOT/NOTICE"
if [[ ! -f "$notice" ]]; then
  fail missing-file "NOTICE does not exist" "NOTICE"
else
  grep -q 'Copyright (C) 2026 Sharique A Farooqui' "$notice" || fail notice-incomplete "missing the copyright line" "NOTICE"
  grep -q 'AGPL-3.0-only' "$notice" || fail notice-incomplete "missing the AGPL-3.0-only identifier" "NOTICE"
fi

# ── README: present, with a licence section and a link to LICENSE ──────────────
readme="$ROOT/README.md"
if [[ ! -f "$readme" ]]; then
  fail missing-file "README.md does not exist" "README.md"
else
  grep -q '^## License' "$readme" || fail readme-incomplete "no '## License' section" "README.md"
  grep -q '(LICENSE)' "$readme" || fail readme-incomplete "no link to LICENSE" "README.md"
fi

# ── Manifests: same identifier everywhere one is declared ──────────────────────
# Parsed with jq rather than grep: swagger.json is a large real document with
# many unrelated "name" keys (parameters, schemas, ...), so a text match risks
# picking up the wrong one. jq scopes the query to the exact field.
command -v jq >/dev/null 2>&1 || { echo "error: jq is required and was not found on PATH" >&2; exit 2; }

pkg="$ROOT/frontend/package.json"
[[ -f "$pkg" ]] || { echo "error: required file missing: frontend/package.json (under $ROOT)" >&2; exit 2; }
pkg_license="$(jq -r '.license // empty' "$pkg" 2>/dev/null)"
if [[ -z "$pkg_license" ]]; then fail manifest-mismatch "frontend/package.json has no \"license\" field" "frontend/package.json"
elif [[ "$pkg_license" != "AGPL-3.0-only" ]]; then fail manifest-mismatch "frontend/package.json license is '$pkg_license', want AGPL-3.0-only" "frontend/package.json"
fi

swagger="$ROOT/backend/docs/swagger.json"
[[ -f "$swagger" ]] || { echo "error: required file missing: backend/docs/swagger.json (under $ROOT)" >&2; exit 2; }
swagger_license="$(jq -r '.info.license.name // empty' "$swagger" 2>/dev/null)"
if [[ -z "$swagger_license" ]]; then fail manifest-mismatch "backend/docs/swagger.json has no info.license.name" "backend/docs/swagger.json"
elif [[ "$swagger_license" != "AGPL-3.0-only" ]]; then fail manifest-mismatch "backend/docs/swagger.json info.license.name is '$swagger_license', want AGPL-3.0-only" "backend/docs/swagger.json"
fi

# ── The CLA (CLA.md, CONTRIBUTING.md's process, cla.yml) is optional overall —
# a tree from before US2 landed has none of it, and that is a valid state —
# but once any one piece of it exists, all of it must, and consistently.
cla="$ROOT/CLA.md"
claw="$ROOT/.github/workflows/cla.yml"
contributing="$ROOT/CONTRIBUTING.md"
cla_expected=0
[[ -f "$cla" ]] && cla_expected=1
[[ -f "$claw" ]] && cla_expected=1

if [[ "$cla_expected" == 1 ]]; then

# ── CLA.md: present, and covers the required terms and version marker ──────────
if [[ ! -f "$cla" ]]; then
  fail missing-file "CLA.md does not exist" "CLA.md"
else
  for term in retain sublicense relicense patent employer individual; do
    grep -qi "$term" "$cla" || fail cla-incomplete "missing the term '$term'" "CLA.md"
  done
  grep -q 'Version 1' "$cla" || fail cla-incomplete "missing the 'Version 1' marker" "CLA.md"
fi

# ── CONTRIBUTING.md: mentions CLA.md and no longer defers to it ────────────────
if [[ -f "$contributing" ]]; then
  grep -qi 'CLA\.md' "$contributing" || fail contributing-incomplete "does not mention CLA.md" "CONTRIBUTING.md"
  grep -qi 'not merged yet' "$contributing" && fail contributing-incomplete "still says code contributions are not merged yet" "CONTRIBUTING.md"
fi

# ── cla.yml: present, and safe for a pull_request_target workflow ──────────────
if [[ ! -f "$claw" ]]; then
  fail missing-file ".github/workflows/cla.yml does not exist" ".github/workflows/cla.yml"
else
  grep -q 'pull_request_target:' "$claw" || fail cla-workflow-unsafe "does not trigger on pull_request_target" ".github/workflows/cla.yml"
  grep -q 'issue_comment:' "$claw" || fail cla-workflow-unsafe "does not trigger on issue_comment" ".github/workflows/cla.yml"
  grep -q 'actions/checkout' "$claw" && fail cla-workflow-unsafe "checks out the pull request's code (unsafe under pull_request_target)" ".github/workflows/cla.yml"
  # permissions: block must declare only contents/pull-requests/statuses/actions
  if command -v yq >/dev/null 2>&1; then
    extra="$(yq -r '.permissions // {} | keys | .[]' "$claw" 2>/dev/null | grep -vE '^(contents|pull-requests|statuses|actions)$' || true)"
  else
    # No yq: fall back to a line-based scan of the top-level permissions block.
    extra="$(awk '/^permissions:/{p=1;next} p && /^[a-zA-Z]/{p=0} p && /^[[:space:]]+[a-zA-Z_-]+:/{gsub(/:.*/,"");gsub(/^[[:space:]]+/,"");print}' "$claw" \
      | grep -vE '^(contents|pull-requests|statuses|actions)$' || true)"
  fi
  [[ -z "$extra" ]] || fail cla-workflow-unsafe "permissions include more than contents/pull-requests/statuses/actions: $(printf '%s' "$extra" | tr '\n' ',')" ".github/workflows/cla.yml"
  # Signatures live on a branch of this same repository. Naming a remote
  # organization or repository makes the action demand a personal access
  # token we do not have, and the check then fails on every pull request.
  grep -qE '^[[:space:]]*remote-(organization|repository)-name:' "$claw" && fail cla-workflow-unsafe "names a remote organization or repository (the action then needs a personal access token); signatures belong on a branch of this repository" ".github/workflows/cla.yml"
  # The action must be referenced by a full 40-hex commit SHA, not a tag.
  action_ref="$(grep -oE 'step-security/contributor-assistant-github-action@[^[:space:]]+' "$claw" | head -1 | cut -d@ -f2)"
  if [[ -z "$action_ref" ]]; then
    fail cla-workflow-unsafe "does not reference step-security/contributor-assistant-github-action" ".github/workflows/cla.yml"
  elif ! [[ "$action_ref" =~ ^[0-9a-f]{40}$ ]]; then
    fail cla-workflow-unsafe "references the CLA action by '$action_ref', not a full 40-hex commit SHA" ".github/workflows/cla.yml"
  fi
fi

fi # cla_expected

# ── No stale "unlicensed / undecided" statement in a plausible place ───────────
scanned=("$readme" "$notice" "$ROOT/LICENSE" "$ROOT/CONTRIBUTING.md")
shopt -s nullglob
while IFS= read -r -d '' f; do scanned+=("$f"); done < <(find "$ROOT/docs" -type f -name '*.md' -print0 2>/dev/null)
shopt -u nullglob

for f in "${scanned[@]}"; do
  [[ -f "$f" ]] || continue
  n=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    n=$((n+1))
    if printf '%s' "$line" | grep -qiE 'no licen[cs]e|unlicensed|undecided'; then
      fail stale-statement "still says the project is unlicensed or undecided: \"$(printf '%s' "$line" | head -c 80)\"" "$(rel "$f"):$n"
    fi
  done < "$f"
done

if [[ "$fails" -eq 0 ]]; then echo "OK: all licence statements agree"; exit 0; fi
printf '\n%d check(s) failed\n' "$fails"
exit 1
