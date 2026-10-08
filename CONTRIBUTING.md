# Contributing to Mansooba

## Bug reports, questions and ideas

Always welcome — please open an issue. No agreement needed for these.

## Code contributions

**Not merged yet.** Mansooba is licensed under AGPL-3.0-only, with the
option to offer a commercial license in future (see
[NOTICE](NOTICE)). Keeping that option open means every code
contribution needs a signed contributor license agreement first, and that
mechanism is not yet operational. Open an issue to discuss an idea in the
meantime; once the agreement process is live, this section will explain how
to sign it and pull requests will be accepted.

## Running the tests

- Backend: `cd backend && go test ./...` (set `REQUIRE_LOCALSTACK=1` and
  start LocalStack first — `docker compose up -d localstack localstack-init`
  — to run the storage tests for real rather than skipping them).
- Frontend: `cd frontend && npm test` and `npm run typecheck`.
- Licensing and version checks: `scripts/check-license-statements.sh` and
  `scripts/verify-versions.sh` (both read-only, no network).

See the [README](README.md) for how to run the whole stack locally.
