# Contributing to Mansooba

## Bug reports, questions and ideas

Always welcome — please open an issue. No agreement needed for these.

## Code contributions

Mansooba is licensed under AGPL-3.0-only, with the option to offer a
commercial license in future (see [NOTICE](NOTICE)). Keeping
that option open means every code contribution needs a one-time signature
of [CLA.md](CLA.md) before it can be merged — get your employer's
permission first if their policies require it.

To sign: open your pull request as usual. The `CLA` check will comment
with the exact sentence to post as a comment on that pull request:

> I have read the CLA Document and I hereby sign the CLA

You only need to do this once; later pull requests won't ask again unless
the agreement's version changes. Issues and ideas need no agreement.

## Running the tests

- Backend: `cd backend && go test ./...` (set `REQUIRE_LOCALSTACK=1` and
  start LocalStack first — `docker compose up -d localstack localstack-init`
  — to run the storage tests for real rather than skipping them).
- Frontend: `cd frontend && npm test` and `npm run typecheck`.
- Licensing and version checks: `scripts/check-license-statements.sh` and
  `scripts/verify-versions.sh` (both read-only, no network).

See the [README](README.md) for how to run the whole stack locally.
