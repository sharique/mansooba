# Moving off SQLite

Mansooba no longer supports SQLite. The server refuses to start when `DB_DRIVER` is `sqlite`, and it never opens or changes your SQLite file when it does. This page explains how to tell whether you are affected and how to get to a supported database.

Supported databases: **PostgreSQL** (preferred) and **MySQL / MariaDB**. The versions the project tests are PostgreSQL 18, MariaDB 11 and MySQL 8.4.

## Are you affected?

You are if any of these is true:

- Your settings (`.env`, your compose file, or the environment) contain `DB_DRIVER=sqlite`, or a `DB_DSN` that is a file path such as `./dev.db` or `/data/dev.db`.
- You started Mansooba from an older `compose.yml` or quick-start file that mounts a volume called `sqlite_data`.
- You ran the backend with `go run` and never set `DB_DRIVER` (older versions defaulted to SQLite). The server now needs both `DB_DRIVER` and `DB_DSN` to be set.

If your settings already say `postgres`, `mysql` or `mariadb`, nothing changes for you, except that the connection pool now defaults to 25 open and 5 idle connections (it was unlimited). Set `DB_MAX_OPEN_CONNS` and `DB_MAX_IDLE_CONNS` if you need other values.

## What the new version does

It stops at start-up with a message that names the supported databases and points here. Your SQLite file is left exactly as it was, so nothing is lost by trying.

## Option 1 — Start fresh (a development or evaluation setup)

This is what most people want. You lose the data in the SQLite file.

1. Stop the old stack.
2. Remove the old data if you no longer want it: `docker volume rm <project>_sqlite_data` (find the exact name with `docker volume ls`), or delete the `.db` file.
3. Get the new `compose.yml` (it now includes a `postgres` service) or add a PostgreSQL service to your own file, and set `DB_DRIVER=postgres` with a matching `DB_DSN`. See [running-locally-using-docker.md](running-locally-using-docker.md) or [running-from-ghcr.md](running-from-ghcr.md) for complete files.
4. Start the stack. The first start creates the tables and shows the setup wizard, where you create the first administrator.

Once the images are on your machine the stack is healthy in about a minute (34 seconds when this guide was walked through). The first run also downloads the images, which takes a few minutes on a typical connection.

## Option 2 — Keep your data (move it by hand)

There is no export or import command, and none is planned. If you need the data, move it yourself:

1. **Create the schema on the new database first.** Start the new version once against an empty PostgreSQL (or MySQL / MariaDB) database and stop it again. This builds the tables, indexes and constraints the application expects. Do not create them by hand.
2. **Copy the rows.** Dump the data from SQLite (`sqlite3 mansooba.db .dump` writes SQL, or `.mode csv` per table writes CSV) and load it table by table into the new database. A raw SQLite dump will not load into PostgreSQL unchanged. Types differ (booleans, timestamps), and statements that create tables must be removed because the tables already exist. CSV per table is usually the simpler route.
3. **Load tables in this order** so references line up: `users`, `projects`, `project_members`, `sprints`, `labels`, `issues`, `issue_labels`, `comments`, `activity_events`, `issue_relations`, `notifications`, `attachments`, `global_settings`. The `revoked_tokens` and `password_reset_tokens` tables can stay empty: people sign in again and request new reset links. Issues point at sprints through a real foreign key, so load `sprints` before `issues`.
4. **Reset the id counters.** After loading explicit ids into PostgreSQL, set each table's sequence to its highest id (`SELECT setval(pg_get_serial_sequence('issues', 'id'), (SELECT MAX(id) FROM issues));`), or new rows will collide with old ones.
5. **Check before you trust it:** sign in, open a project, and compare the issue counts with the old instance.

Attachments live in object storage (S3 or LocalStack), not in the database, so they are not affected.

## Staying on the old version

The images of earlier releases stay available in the container registry under their release tags. They are not maintained, so use one only while you move.

## Getting help

Open an issue in the project's repository with the message the server printed and which option you chose.
