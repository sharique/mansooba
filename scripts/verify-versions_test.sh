#!/usr/bin/env bash
# Fixture tests for verify-versions.sh (contract: specs/015-tech-stack-refresh/
# contracts/verify-versions.md). Builds a small known-good tree in a temp
# directory, checks the checker accepts it, then breaks one thing at a time and
# checks the checker rejects it with the right message. Written before the
# checker (Constitution III).
#
# Usage: scripts/verify-versions_test.sh        (exit 0 = all cases passed)

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$HERE/verify-versions.sh"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT

passed=0; failed=0

# Writes a complete, valid tree into $1.
make_good_tree() {
  local r="$1"
  mkdir -p "$r/backend" "$r/frontend" "$r/terraform" "$r/.github/workflows"
  cat > "$r/compose.yml" <<'EOF'
services:
  backend:
    build: ./backend
  localstack:
    image: localstack/localstack:4.14.0
  localstack-init:
    image: amazon/aws-cli:2.37.3
  mailpit:
    image: axllent/mailpit:v1.31
  loki:
    image: grafana/loki:3.7
  grafana:
    image: grafana/grafana:13.2
  alloy:
    image: grafana/alloy:v1.20.0
EOF
  cat > "$r/compose.prod.yml" <<'EOF'
services:
  backend:
    image: ghcr.io/sharique/mansooba-backend:latest
  frontend:
    image: ghcr.io/sharique/mansooba-frontend:latest
  loki:
    image: grafana/loki:3.7
  grafana:
    image: grafana/grafana:13.2
  alloy:
    image: grafana/alloy:v1.20.0
EOF
  cat > "$r/backend/Dockerfile" <<'EOF'
FROM golang:1.27-alpine3.24 AS builder
RUN echo build
FROM alpine:3.24
COPY --from=builder /app /app
EOF
  cat > "$r/frontend/Dockerfile" <<'EOF'
FROM node:24-alpine3.24 AS builder
RUN echo build
FROM nginx:1.30-alpine3.24
EOF
  cat > "$r/terraform/user-data.sh" <<'EOF'
#!/bin/bash
curl -fsSL "https://github.com/docker/compose/releases/download/v5.5.1/docker-compose-linux-x86_64" -o /usr/local/bin/docker-compose
EOF
  cat > "$r/.github/workflows/ci.yml" <<'EOF'
jobs:
  test:
    runs-on: ubuntu-26.04
    strategy:
      matrix:
        go: ["1.27.x"]
    services:
      localstack:
        image: localstack/localstack:4.14.0
    steps:
      - uses: actions/checkout@v7
  frontend:
    runs-on: ubuntu-26.04
    steps:
      - uses: actions/setup-node@v7
        with:
          node-version-file: frontend/.nvmrc
EOF
  printf 'module example.com/x\n\ngo 1.27\n' > "$r/backend/go.mod"
  printf '24\n' > "$r/frontend/.nvmrc"
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

run_case "good tree passes"                                   0 "OK: all version checks passed" ""
run_case "own ghcr image with :latest is allowed"             0 "" "sed -i 's#mansooba-backend:latest#mansooba-backend:latest#' compose.prod.yml"
run_case "Dockerfile stage alias is not an image"             0 "" "printf 'FROM builder AS again\n' >> backend/Dockerfile"

run_case "floating :latest image is rejected"                 1 "FAIL floating-reference" "sed -i 's#amazon/aws-cli:2.37.3#amazon/aws-cli:latest#' compose.yml"
run_case "image with no tag is rejected"                      1 "FAIL floating-reference" "sed -i 's#axllent/mailpit:v1.31#axllent/mailpit#' compose.yml"
run_case "non-version tag (golang:alpine) is rejected"        1 "FAIL floating-reference" "sed -i 's#golang:1.27-alpine3.24#golang:alpine#' backend/Dockerfile"
run_case "release-line tags (24, 1.30, 3.24) are accepted"     0 "OK: all version checks passed" ""
run_case "exact patch tags are still accepted"               0 "OK: all version checks passed" "sed -i 's#nginx:1.30-alpine3.24#nginx:1.30.5-alpine3.24#' frontend/Dockerfile"
run_case "Go image tag without a release line is rejected"    1 "FAIL version-mismatch" "sed -i 's#golang:1.27-alpine3.24#golang:1-alpine3.24#' backend/Dockerfile"
run_case "tag that is not a version (nginx:stable) is rejected" 1 "FAIL floating-reference" "sed -i 's#nginx:1.30-alpine3.24#nginx:stable-alpine#' frontend/Dockerfile"
run_case "releases/latest download is rejected"               1 "FAIL floating-reference" "sed -i 's#releases/download/v5.5.1#releases/latest/download#' terraform/user-data.sh"
run_case "runs-on ubuntu-latest is rejected"                  1 "FAIL floating-reference" "sed -i '0,/ubuntu-26.04/s#ubuntu-26.04#ubuntu-latest#' .github/workflows/ci.yml"

run_case "Go version mismatch (go.mod vs CI) is rejected"     1 "FAIL version-mismatch" "printf 'module x\n\ngo 1.25.9\n' > backend/go.mod"
run_case "Go version mismatch (Dockerfile) is rejected"       1 "FAIL version-mismatch" "sed -i 's#golang:1.27-alpine3.24#golang:1.26-alpine3.24#' backend/Dockerfile"
run_case "Node mismatch (.nvmrc vs Dockerfile) is rejected"   1 "FAIL version-mismatch" "printf '22\n' > frontend/.nvmrc"
run_case "CI literal Node version that disagrees is rejected"  1 "FAIL version-mismatch" "sed -i 's#node-version-file: frontend/.nvmrc#node-version: 22#' .github/workflows/ci.yml"
run_case "CI literal Node version that agrees passes"         0 "OK: all version checks passed" "sed -i 's#node-version-file: frontend/.nvmrc#node-version: 24#' .github/workflows/ci.yml"
run_case "missing .nvmrc is rejected"                         1 "FAIL version-mismatch" "rm frontend/.nvmrc"
run_case "LocalStack tag mismatch is rejected"                1 "FAIL version-mismatch" "sed -i 's#localstack/localstack:4.14.0#localstack/localstack:4.13.0#' compose.yml"
run_case "Loki tag differs between compose files"             1 "FAIL version-mismatch" "sed -i 's#grafana/loki:3.7#grafana/loki:3.2.0#' compose.prod.yml"

# Every problem is reported, not just the first.
R2="$WORK/multi"; make_good_tree "$R2"
( cd "$R2" && sed -i 's#amazon/aws-cli:2.37.3#amazon/aws-cli:latest#' compose.yml && printf '22\n' > frontend/.nvmrc )
out="$(VERIFY_ROOT="$R2" "$CHECKER" 2>&1)"; code=$?
n="$(printf '%s\n' "$out" | grep -c '^FAIL ')"
if [[ "$code" == 1 && "$n" -ge 2 && "$out" == *"failed"* ]]; then printf 'ok   %s\n' "reports every failure, with a total"; passed=$((passed+1))
else printf 'FAIL %s (exit %s, %s FAIL lines)\n' "reports every failure, with a total" "$code" "$n"; failed=$((failed+1)); fi

# The failure lines name the file.
R3="$WORK/named"; make_good_tree "$R3"; ( cd "$R3" && sed -i 's#amazon/aws-cli:2.37.3#amazon/aws-cli:latest#' compose.yml )
out="$(VERIFY_ROOT="$R3" "$CHECKER" 2>&1)"
if [[ "$out" == *"compose.yml:"* ]]; then printf 'ok   %s\n' "failure line names file and line"; passed=$((passed+1))
else printf 'FAIL %s\n     output: %s\n' "failure line names file and line" "$out"; failed=$((failed+1)); fi

# A required file is missing: the checker itself cannot run (exit 2).
R4="$WORK/missing"; make_good_tree "$R4"; rm "$R4/compose.yml"
VERIFY_ROOT="$R4" "$CHECKER" >/dev/null 2>&1; code=$?
if [[ "$code" == 2 ]]; then printf 'ok   %s\n' "missing required file exits 2"; passed=$((passed+1))
else printf 'FAIL %s (exit %s)\n' "missing required file exits 2" "$code"; failed=$((failed+1)); fi

printf '\n%d passed, %d failed\n' "$passed" "$failed"
[[ "$failed" -eq 0 ]]
