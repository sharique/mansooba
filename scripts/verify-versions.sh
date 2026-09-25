#!/usr/bin/env bash
# Guards two things between quarterly audits (ADR-034, specs/015-tech-stack-refresh):
#
#   A. No floating references: image tags, download URLs and CI runner labels
#      must name an explicit version.
#   B. Versions declared in more than one place must agree (Go, Node, the S3
#      emulator tag, and the Loki/Grafana/Alloy tags across the two compose files).
#
# It does NOT judge whether a pinned version is current or supported; that needs
# live data and is the quarterly check's job (docs/guides/quarterly-tech-stack-check.md).
#
# Read-only, no network. Run from anywhere:  scripts/verify-versions.sh
# Exit: 0 all checks passed, 1 at least one check failed, 2 the script could not run.
# VERIFY_ROOT overrides the tree to check (used by verify-versions_test.sh).

set -uo pipefail

ROOT="${VERIFY_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
fails=0

fail() { printf 'FAIL %s: %s (%s)\n' "$1" "$2" "$3"; fails=$((fails+1)); }
rel() { printf '%s' "${1#"$ROOT"/}"; }

required=(compose.yml compose.prod.yml backend/Dockerfile frontend/Dockerfile backend/go.mod
          terraform/user-data.sh .github/workflows/ci.yml)
for f in "${required[@]}"; do
  [[ -f "$ROOT/$f" ]] || { echo "error: required file missing: $f (under $ROOT)" >&2; exit 2; }
done

shopt -s nullglob
workflows=("$ROOT"/.github/workflows/*.yml "$ROOT"/.github/workflows/*.yaml)
shopt -u nullglob
compose_files=("$ROOT/compose.yml" "$ROOT/compose.prod.yml")
dockerfiles=("$ROOT/backend/Dockerfile" "$ROOT/frontend/Dockerfile")
scanned=("${compose_files[@]}" "${dockerfiles[@]}" "$ROOT/terraform/user-data.sh" "${workflows[@]}")

# find_first <file> <ERE with one capture group>  ->  prints "line|capture", status 0 if found.
find_first() {
  local f="$1" re="$2" n=0 line
  while IFS= read -r line || [[ -n "$line" ]]; do
    n=$((n+1))
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    if [[ "$line" =~ $re ]]; then printf '%s|%s' "$n" "${BASH_REMATCH[1]}"; return 0; fi
  done < "$f"
  return 1
}

# ── A. Floating references ────────────────────────────────────────────────────

# check_image <ref> <file> <line>
check_image() {
  local ref="$1" f="$2" n="$3" last tag
  ref="${ref%\"}"; ref="${ref#\"}"; ref="${ref%\'}"; ref="${ref#\'}"
  [[ "$ref" == *'$'* ]] && return 0                        # variable, cannot judge
  [[ "$ref" == scratch ]] && return 0
  [[ "$ref" == ghcr.io/sharique/mansooba-* ]] && return 0  # the project's own release images
  [[ "$ref" == *@sha256:* ]] && return 0                   # pinned by digest
  last="${ref##*/}"
  if [[ "$last" != *:* ]]; then fail floating-reference "image '$ref' has no tag" "$(rel "$f"):$n"; return 0; fi
  tag="${last#*:}"
  if [[ "$tag" == latest ]]; then fail floating-reference "image '$ref' uses :latest" "$(rel "$f"):$n"
  elif ! [[ "$tag" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    fail floating-reference "image '$ref' tag '$tag' is not an exact version (need major.minor.patch)" "$(rel "$f"):$n"
  fi
}

# compose files and workflow service containers: `image: <ref>`
for f in "${compose_files[@]}" "${workflows[@]}"; do
  n=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    n=$((n+1))
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    if [[ "$line" =~ ^[[:space:]]*-?[[:space:]]*image:[[:space:]]*([^[:space:]#]+) ]]; then
      check_image "${BASH_REMATCH[1]}" "$f" "$n"
    fi
  done < "$f"
done

# Dockerfiles: FROM <ref> [AS <alias>]; earlier stage aliases are not images
for f in "${dockerfiles[@]}"; do
  n=0; aliases=" "
  while IFS= read -r line || [[ -n "$line" ]]; do
    n=$((n+1))
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    if [[ "$line" =~ ^[[:space:]]*[Ff][Rr][Oo][Mm][[:space:]]+(--platform=[^[:space:]]+[[:space:]]+)?([^[:space:]]+)([[:space:]]+[Aa][Ss][[:space:]]+([^[:space:]]+))? ]]; then
      ref="${BASH_REMATCH[2]}"; alias="${BASH_REMATCH[4]:-}"
      lower="$(printf '%s' "$ref" | tr '[:upper:]' '[:lower:]')"
      if [[ "$aliases" != *" $lower "* ]]; then check_image "$ref" "$f" "$n"; fi
      [[ -n "$alias" ]] && aliases="$aliases$(printf '%s' "$alias" | tr '[:upper:]' '[:lower:]') "
    fi
  done < "$f"
done

# download URLs that follow a moving "latest"
for f in "${scanned[@]}"; do
  n=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    n=$((n+1))
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    if [[ "$line" =~ (releases/latest|latest/download) ]]; then
      fail floating-reference "download URL follows a moving latest release" "$(rel "$f"):$n"
    fi
  done < "$f"
done

# CI runner labels ending in -latest
for f in "${workflows[@]}"; do
  n=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    n=$((n+1))
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    if [[ "$line" =~ runs-on:[[:space:]]*[\"\']?([A-Za-z0-9._-]+) ]] && [[ "${BASH_REMATCH[1]}" == *-latest ]]; then
      fail floating-reference "runner label '${BASH_REMATCH[1]}' floats; name a version such as ubuntu-24.04" "$(rel "$f"):$n"
    fi
  done < "$f"
done

# ── B. Declared versions agree ─────────────────────────────────────────────────

# Go: go.mod is the source of truth (major.minor); CI matrix and the builder image must match.
gomod="$ROOT/backend/go.mod"
if r="$(find_first "$gomod" '^go[[:space:]]+([0-9]+\.[0-9]+)')"; then
  go_want="${r#*|}"
  ci_go=""; ci_go_loc=""
  for f in "${workflows[@]}"; do
    if r="$(find_first "$f" 'go:[[:space:]]*\[[[:space:]]*"([0-9]+\.[0-9]+)')"; then ci_go="${r#*|}"; ci_go_loc="$(rel "$f"):${r%%|*}"; break; fi
  done
  if [[ -z "$ci_go" ]]; then fail version-mismatch "cannot find the Go matrix version in the workflows" ".github/workflows"
  elif [[ "$ci_go" != "$go_want" ]]; then fail version-mismatch "Go $ci_go in CI differs from go.mod's $go_want" "$ci_go_loc"; fi
  if r="$(find_first "$ROOT/backend/Dockerfile" 'FROM[[:space:]]+golang:([0-9]+\.[0-9]+)')"; then
    [[ "${r#*|}" == "$go_want" ]] || fail version-mismatch "Go ${r#*|} in the Dockerfile differs from go.mod's $go_want" "backend/Dockerfile:${r%%|*}"
  else fail version-mismatch "cannot find a golang:<version> builder in the Dockerfile" "backend/Dockerfile"; fi
else
  fail version-mismatch "cannot read the go directive" "backend/go.mod"
fi

# Node: frontend/.nvmrc is the source of truth (major); the Dockerfile and CI must match.
nvmrc="$ROOT/frontend/.nvmrc"
if [[ ! -f "$nvmrc" ]]; then
  fail version-mismatch "frontend/.nvmrc is missing; it is the single declared Node major" "frontend/.nvmrc"
else
  node_want="$(tr -d 'vV \r\n' < "$nvmrc")"; node_want="${node_want%%.*}"
  if r="$(find_first "$ROOT/frontend/Dockerfile" 'FROM[[:space:]]+node:([0-9]+)')"; then
    [[ "${r#*|}" == "$node_want" ]] || fail version-mismatch "Node ${r#*|} in the Dockerfile differs from .nvmrc's $node_want" "frontend/Dockerfile:${r%%|*}"
  else fail version-mismatch "cannot find a node:<version> builder in the Dockerfile" "frontend/Dockerfile"; fi
  ci_node_ok=0
  for f in "${workflows[@]}"; do
    if r="$(find_first "$f" 'node-version-file:[[:space:]]*([^[:space:]]+)')"; then
      [[ "${r#*|}" == *".nvmrc" ]] && { ci_node_ok=1; break; }
    fi
    if r="$(find_first "$f" 'node(-version)?:[[:space:]]*\[?[[:space:]]*["'"'"']?([0-9]+)')"; then
      # second capture group is the major
      line_no="${r%%|*}"
      major="$(sed -n "${line_no}p" "$f" | grep -oE '[0-9]+' | head -1)"
      if [[ "$major" == "$node_want" ]]; then ci_node_ok=1; break; fi
      fail version-mismatch "Node $major in CI differs from .nvmrc's $node_want" "$(rel "$f"):$line_no"; ci_node_ok=1; break
    fi
  done
  [[ "$ci_node_ok" == 1 ]] || fail version-mismatch "cannot find the Node version used by CI" ".github/workflows"
fi

# S3 emulator: the tag must be the same in compose.yml and CI.
if r="$(find_first "$ROOT/compose.yml" 'localstack/localstack:([^[:space:]"'"'"']+)')"; then
  ls_want="${r#*|}"; ls_seen=0
  for f in "${workflows[@]}"; do
    if r2="$(find_first "$f" 'localstack/localstack:([^[:space:]"'"'"']+)')"; then
      ls_seen=1
      [[ "${r2#*|}" == "$ls_want" ]] || fail version-mismatch "LocalStack ${r2#*|} in CI differs from compose.yml's $ls_want" "$(rel "$f"):${r2%%|*}"
    fi
  done
  [[ "$ls_seen" == 1 ]] || fail version-mismatch "no LocalStack service found in the workflows" ".github/workflows"
else
  fail version-mismatch "no LocalStack image found" "compose.yml"
fi

# Observability: the two compose files must pin the same tags.
for img in grafana/loki grafana/grafana grafana/alloy; do
  a="$(find_first "$ROOT/compose.yml" "$img:([^[:space:]\"']+)")" || a=""
  b="$(find_first "$ROOT/compose.prod.yml" "$img:([^[:space:]\"']+)")" || b=""
  if [[ -z "$a" || -z "$b" ]]; then
    fail version-mismatch "$img is missing from one of the compose files" "compose.yml / compose.prod.yml"
  elif [[ "${a#*|}" != "${b#*|}" ]]; then
    fail version-mismatch "$img is ${a#*|} in compose.yml but ${b#*|} in compose.prod.yml" "compose.prod.yml:${b%%|*}"
  fi
done

if [[ "$fails" -eq 0 ]]; then echo "OK: all version checks passed"; exit 0; fi
printf '\n%d check(s) failed\n' "$fails"
exit 1
