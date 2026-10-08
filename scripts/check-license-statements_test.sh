#!/usr/bin/env bash
# Fixture tests for check-license-statements.sh. Same style as scripts/verify-versions_test.sh:
# build a small known-good tree in a temp directory, confirm the checker
# accepts it, then break one thing at a time and confirm it is rejected with
# the right message. Written before the checker (Constitution III).
#
# Usage: scripts/check-license-statements_test.sh   (exit 0 = all cases passed)

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$HERE/check-license-statements.sh"
FIXTURE_LICENSE="$HERE/testdata/license-statements/agpl-3.0-fixture.txt"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT

passed=0; failed=0

# Writes a complete, valid tree into $1.
make_good_tree() {
  local r="$1"
  mkdir -p "$r/backend/docs" "$r/frontend" "$r/docs/deployments"
  cp "$FIXTURE_LICENSE" "$r/LICENSE"
  cat > "$r/NOTICE" <<'EOF'
mansooba

Copyright (C) 2026 Sharique A Farooqui

Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0-only).
See LICENSE for the full text.

The owner may offer this software under separate commercial terms in the
future; no such offer exists today.

Third-party components: see THIRD-PARTY-NOTICES.md.
EOF
  cat > "$r/README.md" <<'EOF'
# mansooba

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0-only).
EOF
  cat > "$r/CONTRIBUTING.md" <<'EOF'
# Contributing

Bug reports, questions and ideas are always welcome via issues.
EOF
  cat > "$r/frontend/package.json" <<'EOF'
{
  "name": "frontend",
  "private": true,
  "license": "AGPL-3.0-only"
}
EOF
  cat > "$r/backend/docs/swagger.json" <<'EOF'
{
  "info": {
    "title": "mansooba API",
    "license": { "name": "AGPL-3.0-only", "url": "https://www.gnu.org/licenses/agpl-3.0.html" }
  }
}
EOF
  cat > "$r/docs/deployments/running-locally-using-docker.md" <<'EOF'
# Running locally

Nothing licence-related to see here.
EOF
}

# Adds a valid CLA.md, cla.yml and the post-CLA CONTRIBUTING.md to an
# already-good tree.
add_cla_files() {
  local r="$1"
  mkdir -p "$r/.github/workflows"
  cat > "$r/CLA.md" <<'EOF'
# Contributor License Agreement (Version 1)

By signing, you (an **individual** contributor) confirm you have the right
to contribute this work, including any needed permission from your
**employer**. You **retain** your copyright, and grant the project owner a
perpetual, worldwide, royalty-free right to use, modify, distribute,
**sublicense** and **relicense** your contribution under any terms,
including commercial ones, together with a **patent** licence for it.
EOF
  cat > "$r/CONTRIBUTING.md" <<'EOF'
# Contributing

Bug reports, questions and ideas are always welcome via issues.

Code contributions need a one-time signature of [CLA.md](CLA.md); the CLA
check comments with the exact sentence to post.
EOF
  cat > "$r/.github/workflows/cla.yml" <<'EOF'
name: CLA
on:
  pull_request_target:
    types: [opened, synchronize, closed]
  issue_comment:
    types: [created]
permissions:
  contents: write
  pull-requests: write
  statuses: write
  actions: write
jobs:
  cla:
    runs-on: ubuntu-26.04
    steps:
      - uses: step-security/contributor-assistant-github-action@b9bd60bf1b766fa48dae03427059187137236239
        with:
          path-to-signatures: signatures/v1/cla.json
          branch: cla-signatures
EOF
}

# run_case <name> <expected exit> <expected output substring or ""> <mutation shell snippet using $R>
run_case() {
  local name="$1" want_code="$2" want_text="$3" mutate="$4"
  local R="$WORK/case-$passed-$failed"
  rm -rf "$R"; make_good_tree "$R"
  if [[ -n "$mutate" ]]; then ( cd "$R" && eval "$mutate" ); fi
  local out code
  out="$(VERIFY_ROOT="$R" "$CHECKER" 2>&1)"; code=$?
  if [[ "$code" == "$want_code" && ( -z "$want_text" || "$out" == *"$want_text"* ) ]]; then
    printf 'ok   %s\n' "$name"; passed=$((passed+1))
  else
    printf 'FAIL %s\n     want exit %s containing "%s"; got exit %s\n     output: %s\n' \
      "$name" "$want_code" "$want_text" "$code" "$(printf '%s' "$out" | head -5 | tr '\n' '|')"
    failed=$((failed+1))
  fi
}

[[ -x "$CHECKER" ]] || { echo "FAIL: $CHECKER does not exist or is not executable (expected before implementation)"; exit 1; }

run_case "good tree passes"                                    0 "OK: all licence statements agree" ""

run_case "missing LICENSE is rejected"                         1 "FAIL missing-file" "rm LICENSE"
run_case "LICENSE with wrong text is rejected"                 1 "FAIL wrong-license-text" "printf 'not the real license\n' > LICENSE"
run_case "LICENSE with one changed byte is rejected"            1 "FAIL wrong-license-text" "sed -i '1s/.*/GNU AFFERO GENERAL PUBLIC LICENSE (MODIFIED)/' LICENSE"

run_case "missing NOTICE is rejected"                          1 "FAIL missing-file" "rm NOTICE"
run_case "NOTICE without the copyright line is rejected"       1 "FAIL notice-incomplete" "sed -i '/Copyright/d' NOTICE"
run_case "NOTICE without AGPL-3.0-only is rejected"            1 "FAIL notice-incomplete" "sed -i 's/AGPL-3.0-only/a different license/' NOTICE"

run_case "README without a licence section is rejected"        1 "FAIL readme-incomplete" "sed -i '/## License/,\$d' README.md"
run_case "README without a link to LICENSE is rejected"        1 "FAIL readme-incomplete" "sed -i 's#\\[GNU Affero General Public License v3.0\\](LICENSE)#GNU Affero General Public License v3.0#' README.md"

run_case "frontend package.json missing license is rejected"   1 "FAIL manifest-mismatch" "sed -i '/\"license\"/d' frontend/package.json"
run_case "frontend package.json wrong license is rejected"     1 "FAIL manifest-mismatch" "sed -i 's/AGPL-3.0-only/MIT/' frontend/package.json"

run_case "swagger missing license.name is rejected"            1 "FAIL manifest-mismatch" "sed -i 's/\"name\": \"AGPL-3.0-only\", //' backend/docs/swagger.json"
run_case "swagger wrong license.name is rejected"              1 "FAIL manifest-mismatch" "sed -i 's/AGPL-3.0-only/Apache-2.0/' backend/docs/swagger.json"

run_case "'undecided' in a scanned doc is rejected"            1 "FAIL stale-statement" "printf '\\nLicensing is still undecided.\\n' >> docs/deployments/running-locally-using-docker.md"
run_case "'no license' in README is rejected"                  1 "FAIL stale-statement" "printf '\\nThis project has no license yet.\\n' >> README.md"
run_case "'unlicensed' in NOTICE is rejected"                  1 "FAIL stale-statement" "printf '\\nThis is unlicensed software.\\n' >> NOTICE"
run_case "the word appearing outside the scanned set is ignored" 0 "OK: all licence statements agree" "mkdir -p unrelated && printf 'this word is undecided here but nobody looks\\n' > unrelated/notes.md"

# run_case_cla <name> <expected exit> <expected output substring> <mutation using $R, applied after add_cla_files>
run_case_cla() {
  local name="$1" want_code="$2" want_text="$3" mutate="$4"
  local R="$WORK/cla-$passed-$failed"
  rm -rf "$R"; make_good_tree "$R"; add_cla_files "$R"
  if [[ -n "$mutate" ]]; then ( cd "$R" && eval "$mutate" ); fi
  local out code
  out="$(VERIFY_ROOT="$R" "$CHECKER" 2>&1)"; code=$?
  if [[ "$code" == "$want_code" && ( -z "$want_text" || "$out" == *"$want_text"* ) ]]; then
    printf 'ok   %s\n' "$name"; passed=$((passed+1))
  else
    printf 'FAIL %s\n     want exit %s containing "%s"; got exit %s\n     output: %s\n' \
      "$name" "$want_code" "$want_text" "$code" "$(printf '%s' "$out" | head -5 | tr '\n' '|')"
    failed=$((failed+1))
  fi
}

run_case_cla "good tree with CLA files passes"                     0 "OK: all licence statements agree" ""
run_case_cla "missing CLA.md is rejected"                          1 "FAIL missing-file" "rm CLA.md"
run_case_cla "CLA.md missing 'retain' is rejected"                 1 "FAIL cla-incomplete" "sed -i 's/\\*\\*retain\\*\\*/keep/' CLA.md"
run_case_cla "CLA.md missing 'sublicense' is rejected"             1 "FAIL cla-incomplete" "sed -i 's/\\*\\*sublicense\\*\\*/share/' CLA.md"
run_case_cla "CLA.md missing 'relicense' is rejected"              1 "FAIL cla-incomplete" "sed -i 's/\\*\\*relicense\\*\\*/reoffer/' CLA.md"
run_case_cla "CLA.md missing 'patent' is rejected"                 1 "FAIL cla-incomplete" "sed -i 's/\\*\\*patent\\*\\*/idea/' CLA.md"
run_case_cla "CLA.md missing 'employer' is rejected"               1 "FAIL cla-incomplete" "sed -i 's/\\*\\*employer\\*\\*/company/' CLA.md"
run_case_cla "CLA.md missing 'individual' is rejected"             1 "FAIL cla-incomplete" "sed -i 's/\\*\\*individual\\*\\*/person/' CLA.md"
run_case_cla "CLA.md missing the version marker is rejected"      1 "FAIL cla-incomplete" "sed -i 's/(Version 1)//' CLA.md"
run_case_cla "CONTRIBUTING.md not mentioning CLA.md is rejected"   1 "FAIL contributing-incomplete" "sed -i 's/\\[CLA.md\\](CLA.md)/the agreement/' CONTRIBUTING.md"
run_case_cla "CONTRIBUTING.md still saying 'not merged yet' is rejected" 1 "FAIL contributing-incomplete" "printf '\\ncode contributions are not merged yet.\\n' >> CONTRIBUTING.md"
run_case_cla "cla.yml missing pull_request_target is rejected"     1 "FAIL cla-workflow-unsafe" "sed -i '/pull_request_target:/,+2d' .github/workflows/cla.yml"
run_case_cla "cla.yml missing issue_comment is rejected"           1 "FAIL cla-workflow-unsafe" "sed -i '/issue_comment:/,+1d' .github/workflows/cla.yml"
run_case_cla "cla.yml with an extra permission is rejected"        1 "FAIL cla-workflow-unsafe" "sed -i 's/actions: write/actions: write\\n  id-token: write/' .github/workflows/cla.yml"
run_case_cla "cla.yml with an actions/checkout step is rejected"   1 "FAIL cla-workflow-unsafe" "sed -i '/steps:/a\\      - uses: actions/checkout@v7' .github/workflows/cla.yml"
run_case_cla "cla.yml referencing the action by a tag is rejected" 1 "FAIL cla-workflow-unsafe" "sed -i 's/@b9bd60bf1b766fa48dae03427059187137236239/@v2.6.1/' .github/workflows/cla.yml"
run_case_cla "cla.yml naming a remote repository is rejected"      1 "FAIL cla-workflow-unsafe" "printf '          remote-repository-name: mansooba\\n' >> .github/workflows/cla.yml"
run_case_cla "cla.yml naming a remote organization is rejected"    1 "FAIL cla-workflow-unsafe" "printf '          remote-organization-name: sharique\\n' >> .github/workflows/cla.yml"

# Every problem is reported, not just the first.
R2="$WORK/multi"; make_good_tree "$R2"
( cd "$R2" && rm LICENSE && sed -i 's/AGPL-3.0-only/MIT/' frontend/package.json )
out="$(VERIFY_ROOT="$R2" "$CHECKER" 2>&1)"; code=$?
n="$(printf '%s\n' "$out" | grep -c '^FAIL ')"
if [[ "$code" == 1 && "$n" -ge 2 && "$out" == *"failed"* ]]; then printf 'ok   %s\n' "reports every failure, with a total"; passed=$((passed+1))
else printf 'FAIL %s (exit %s, %s FAIL lines)\n' "reports every failure, with a total" "$code" "$n"; failed=$((failed+1)); fi

# A required file is missing entirely: the checker itself cannot run.
R3="$WORK/missing-root"; make_good_tree "$R3"; rm -rf "$R3/backend"
VERIFY_ROOT="$R3" "$CHECKER" >/dev/null 2>&1; code=$?
if [[ "$code" == 2 ]]; then printf 'ok   %s\n' "missing required directory exits 2"; passed=$((passed+1))
else printf 'FAIL %s (exit %s)\n' "missing required directory exits 2" "$code"; failed=$((failed+1)); fi

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[[ "$failed" -eq 0 ]]
