#!/usr/bin/env bash
# Stand-in for `lichen` in fixture tests (check-licenses_test.sh). Ignores
# its real arguments (the binary path, --config) and instead prints the
# fixed scenario named by LICHEN_STUB_OUTPUT, exiting LICHEN_STUB_EXIT.
# Never touches the network.
set -uo pipefail
: "${LICHEN_STUB_OUTPUT:?LICHEN_STUB_OUTPUT must be set}"
: "${LICHEN_STUB_EXIT:?LICHEN_STUB_EXIT must be set}"
cat "$LICHEN_STUB_OUTPUT"
exit "$LICHEN_STUB_EXIT"
