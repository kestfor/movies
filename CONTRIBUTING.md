# Contributing to KinoKrug

Thank you for improving KinoKrug. Keep changes focused, tested, and consistent with the documented product decisions.

## Workflow

1. Create a branch from the latest `main` using a clear prefix such as `feat/`, `fix/`, or `chore/`.
2. Read the related product, API, migration, and neighboring code before editing.
3. Make the smallest change that fully solves the task.
4. Run the relevant checks locally.
5. Open a pull request into `main` and wait for the required CI quality gate.
6. Resolve review conversations and squash-merge the pull request.

Direct pushes, force-pushes, and deletion of `main` are blocked.

## Project conventions

- Use `title` for the shared movie/series entity. Use `movie` only for TMDB fields or `media_type` values.
- Keep backend responsibilities separated: HTTP transport handles protocol concerns, usecases enforce business rules, and repositories persist data.
- Keep public API routes under `/api/v1`, JSON fields in `snake_case`, and errors in the documented common shape.
- Preserve the mobile Telegram WebApp experience and existing visual patterns.
- Do not add dependencies without a concrete need.
- Add or update tests for changed business logic.

## Database and generated code

- Add schema changes as new tern migrations; do not rewrite applied migrations.
- Edit SQL under `backend/repo/postgres/queries`.
- Run `make generate` after SQL changes.
- Never edit `backend/internal/repo/postgres/gen` manually.
- Commit generated output together with its SQL source.

## Documentation

Update `docs/api.md` whenever a public HTTP contract changes. Update the relevant product documentation when a product decision or business rule changes.

## Checks

Run the complete local suite:

```bash
make check
```

For production-container changes, also run:

```bash
cp .env.example .env
docker compose config --quiet
docker compose build api frontend
```

Do not overwrite an existing local `.env` merely to run this check.

## Secrets

Never commit `.env`, API tokens, Telegram credentials, webhook URLs, database dumps, Coolify credentials, or private operational data. Use placeholders in examples and repository secrets for CI integrations.
