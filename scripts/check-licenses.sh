#!/usr/bin/env bash
# Guards the licence policy: every
# third-party Go module compiled into the backend, every npm package shipped
# in the frontend's production build, and every icon set actually used, must
# be under a licence on scripts/licenses/allowed.txt (or covered by a
# component-specific line in scripts/licenses/exceptions.txt).
#
# Needs the network in real use (to fetch the two pinned tools), unless
# LICHEN_CMD/LICENSE_CHECKER_CMD are overridden — check-licenses_test.sh
# overrides both to local, no-network stubs.
#
# Usage: scripts/check-licenses.sh [--go] [--npm] [--icons]  (default: all three)
# Exit: 0 all checks passed, 1 at least one blocked/unreviewed licence,
# 2 the script could not run.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${LICENSE_ROOT:-$(cd "$HERE/.." && pwd)}"
LICHEN_CMD="${LICHEN_CMD:-go run github.com/uw-labs/lichen@v0.1.7}"
LICENSE_CHECKER_CMD="${LICENSE_CHECKER_CMD:-npx --yes license-checker-rseidelsohn@5.0.1}"
ALLOWED_FILE="${LICENSE_ALLOWED:-$HERE/licenses/allowed.txt}"
EXCEPTIONS_FILE="${LICENSE_EXCEPTIONS:-$HERE/licenses/exceptions.txt}"

do_go=0; do_npm=0; do_icons=0
if [[ $# -eq 0 ]]; then do_go=1; do_npm=1; do_icons=1; fi
for a in "$@"; do
  case "$a" in
    --go) do_go=1 ;;
    --npm) do_npm=1 ;;
    --icons) do_icons=1 ;;
    *) echo "error: unknown flag $a" >&2; exit 2 ;;
  esac
done

[[ -f "$ALLOWED_FILE" ]] || { echo "error: allow-list not found: $ALLOWED_FILE" >&2; exit 2; }
[[ -f "$EXCEPTIONS_FILE" ]] || { echo "error: exceptions file not found: $EXCEPTIONS_FILE" >&2; exit 2; }

fails=0
fail() { printf 'FAIL blocked-licence: %s\n' "$1"; fails=$((fails+1)); }
info() { printf 'INFO exception applied: %s\n' "$1"; }

is_allowed() {
  local lic="$1"
  grep -qxF "$lic" "$ALLOWED_FILE"
}

# has_exception <component> <version> <licence>
has_exception() {
  local comp="$1" ver="$2" lic="$3" line ecomp ever elic
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%%#*}"
    [[ -z "${line// /}" ]] && continue
    read -r ecomp ever elic <<<"$line"
    [[ -z "$ecomp" ]] && continue
    [[ "$ecomp" == "$comp" ]] || continue
    [[ "$ever" == "*" || "$ever" == "$ver" ]] || continue
    [[ "$elic" == "$lic" ]] && return 0
  done < "$EXCEPTIONS_FILE"
  return 1
}

# A licence expression may be a single id, or "(A OR B)" / "(A AND B)".
# Returns 0 (compatible) if: OR and at least one option is allowed/excepted;
# AND and every part is allowed/excepted; or a single id that is
# allowed/excepted. On failure, prints the parts that blocked it via stdout
# (one per line) for the caller to report.
evaluate_expression() {
  local comp="$1" ver="$2" expr="$3" inner op parts part blocked=() any_ok=0
  inner="${expr#\(}"; inner="${inner%\)}"
  if [[ "$inner" == *" OR "* ]]; then op="OR"; IFS=' OR ' read -ra parts <<<"$inner"
  elif [[ "$inner" == *" AND "* ]]; then op="AND"; IFS=' AND ' read -ra parts <<<"$inner"
  else op="SINGLE"; parts=("$inner"); fi

  for part in "${parts[@]}"; do
    part="$(printf '%s' "$part" | sed -E 's/^ +| +$//g')"
    [[ -z "$part" ]] && continue
    if is_allowed "$part"; then any_ok=1
    elif has_exception "$comp" "$ver" "$part"; then any_ok=1; info "$comp $ver ($part)"
    else blocked+=("$part"); fi
  done

  if [[ "$op" == "OR" ]]; then
    [[ "$any_ok" == 1 ]] && return 0
  else
    [[ "${#blocked[@]}" -eq 0 ]] && return 0
  fi
  printf '%s\n' "${blocked[@]}"
  return 1
}

# ── Go: every module compiled into the backend binary ──────────────────────
check_go() {
  local build_root="$ROOT/backend"
  [[ -d "$build_root" ]] || { echo "error: $build_root does not exist" >&2; exit 2; }
  local tmp_bin tmp_out
  tmp_bin="$(mktemp)"; tmp_out="$(mktemp)"
  ( cd "$build_root" && GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go build -o "$tmp_bin" ./cmd/server ) \
    || { echo "error: could not build $build_root/cmd/server" >&2; rm -f "$tmp_bin" "$tmp_out"; exit 2; }

  local lichen_config
  lichen_config="$(mktemp)"
  { echo 'allow:'; grep -vE '^\s*#|^\s*$' "$ALLOWED_FILE" | while IFS= read -r l; do printf '  - %q\n' "$l"; done; } > "$lichen_config"

  local out lichen_code
  out="$($LICHEN_CMD --config="$lichen_config" "$tmp_bin" 2>&1)"; lichen_code=$?
  rm -f "$tmp_bin" "$lichen_config"
  # lichen itself only ever exits 0 (all allowed) or 1 (something blocked,
  # already reflected in $out below); anything else means the tool could not
  # even run (not found, network failure fetching it, ...).
  if [[ "$lichen_code" != 0 && "$lichen_code" != 1 ]]; then
    echo "error: \$LICHEN_CMD ('$LICHEN_CMD') failed to run (exit $lichen_code): $out" >&2
    exit 2
  fi

  local line mod verpart licenses
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" == *"(not allowed - non-permitted licenses: ["*"])"* ]] || continue
    mod="${line%%@*}"
    verpart="${line#*@}"; verpart="${verpart%%:*}"
    licenses="${line#*non-permitted licenses: [}"; licenses="${licenses%%]*}"
    local lic blocked_any=0 first_blocked=""
    IFS=',' read -ra parts <<<"$licenses"
    for lic in "${parts[@]}"; do
      lic="$(printf '%s' "$lic" | sed -E 's/^ +| +$//g')"
      [[ -z "$lic" ]] && continue
      if is_allowed "$lic"; then continue
      elif has_exception "$mod" "$verpart" "$lic"; then info "$mod $verpart ($lic)"; continue
      else blocked_any=1; [[ -z "$first_blocked" ]] && first_blocked="$lic"; fi
    done
    if [[ "$blocked_any" == 1 ]]; then
      fail "$mod@$verpart licence $licenses (backend binary)"
    fi
  done <<<"$out"
  rm -f "$tmp_out"
}

# ── npm: every production package the frontend ships ────────────────────────
check_npm() {
  local frontend="$ROOT/frontend"
  [[ -d "$frontend" ]] || { echo "error: $frontend does not exist" >&2; exit 2; }
  local out; out="$(cd "$frontend" && $LICENSE_CHECKER_CMD --production --excludePrivatePackages --json 2>/dev/null)"
  [[ -n "$out" ]] || { echo "error: license checker produced no output" >&2; exit 2; }

  local key
  for key in $(printf '%s' "$out" | jq -r 'keys[]'); do
    local namever="$key" name ver lic
    ver="${namever##*@}"; name="${namever%@*}"
    lic="$(printf '%s' "$out" | jq -r --arg k "$key" '.[$k].licenses // "UNLICENSED"')"
    local blocked
    if ! blocked="$(evaluate_expression "$name" "$ver" "$lic")"; then
      fail "$key licence $lic (frontend production tree)"
    fi
  done
}

# ── Icons: every icon set actually used by name/prefix ──────────────────────
check_icons() {
  local app="$ROOT/frontend/app"
  local collections="$ROOT/frontend/node_modules/@iconify/collections/collections.json"
  [[ -d "$app" ]] || { echo "error: $app does not exist" >&2; exit 2; }
  [[ -f "$collections" ]] || { echo "error: $collections does not exist (run npm install first)" >&2; exit 2; }

  local prefixes
  prefixes="$(grep -rhoE '(name|:name)="[a-z][a-z0-9-]*:[a-z0-9-]+"|:name="`[a-z][a-z0-9-]*:[a-z0-9-]+`"' "$app" 2>/dev/null \
    | grep -oE '"[a-z][a-z0-9-]*:|`[a-z][a-z0-9-]*:' | tr -d '"`:' | sort -u)"

  local p lic
  for p in $prefixes; do
    lic="$(jq -r --arg p "$p" '.[$p].license.spdx // empty' "$collections")"
    if [[ -z "$lic" ]]; then
      fail "icon set '$p' is not in the installed collection (frontend/app)"
    elif ! is_allowed "$lic" && ! has_exception "icon:$p" "*" "$lic"; then
      fail "icon set '$p' licence $lic (frontend/app)"
    fi
  done
}

[[ "$do_go" == 1 ]] && check_go
[[ "$do_npm" == 1 ]] && check_npm
[[ "$do_icons" == 1 ]] && check_icons

if [[ "$fails" -eq 0 ]]; then echo "OK: all licence checks passed"; exit 0; fi
printf '\n%d check(s) failed\n' "$fails"
exit 1
