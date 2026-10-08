#!/usr/bin/env bash
# Stand-in for `license-checker-rseidelsohn` in fixture tests. Ignores its
# real arguments and prints the fixed scenario named by
# LICENSE_CHECKER_STUB_OUTPUT, exiting LICENSE_CHECKER_STUB_EXIT.
set -uo pipefail
: "${LICENSE_CHECKER_STUB_OUTPUT:?LICENSE_CHECKER_STUB_OUTPUT must be set}"
: "${LICENSE_CHECKER_STUB_EXIT:?LICENSE_CHECKER_STUB_EXIT must be set}"
cat "$LICENSE_CHECKER_STUB_OUTPUT"
exit "$LICENSE_CHECKER_STUB_EXIT"
