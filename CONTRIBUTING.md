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

Backend tests run on a real database, not an in-memory stand-in: every test
that touches the database gets its own freshly migrated one.

1. Start PostgreSQL (and LocalStack for the storage tests) from the repository root:

   ```bash
   docker compose up -d postgres localstack localstack-init
   export TEST_DB_DSN="host=localhost user=mansooba password=mansooba dbname=mansooba_test port=5432 sslmode=disable"
   ```

   Use another port with `POSTGRES_PORT=15432 docker compose up -d postgres`
   (and the same port in `TEST_DB_DSN`) if 5432 is taken. No container runtime?
   Point `TEST_DB_DSN` at any PostgreSQL you already have, with a user that is
   allowed to create and drop databases.
2. `cd backend && go test ./...` (add `-race` as CI does).

Each test gets its own database (named `test_<time>_<suffix>`, dropped when the
test ends; leftovers from a crashed run are removed after an hour). The first run
after a change to the models also builds a small template database
(`mansooba_tmpl_<hash>`) that later runs copy; old ones can be dropped or the
volume wiped (`docker compose down -v`).

A database test fails, with these instructions in the message, when no database
can be reached; it never silently skips. Set `REQUIRE_LOCALSTACK=1` to make a
missing LocalStack fail the storage tests too instead of skipping them.

**MariaDB and MySQL** are officially supported but are not run on every change.
Check them on demand, and before each release (quote the result in the release
notes), with the same tests:

```bash
docker run -d --name mdb -p 13306:3306 -e MARIADB_ROOT_PASSWORD=test mariadb:11
TEST_DB_DRIVER=mariadb TEST_DB_DSN='root:test@tcp(localhost:13306)/?parseTime=true' go test ./...
```

For MySQL use a `mysql:8.4` container and `TEST_DB_DRIVER=mysql`. Or start the
"Database matrix (on demand)" workflow from the Actions tab.

- Frontend: `cd frontend && npm test` and `npm run typecheck`.
- Licensing and version checks: `scripts/check-license-statements.sh` and
  `scripts/verify-versions.sh` (both read-only, no network).

See the [README](README.md) for how to run the whole stack locally.
