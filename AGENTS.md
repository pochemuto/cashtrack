# Agent Instructions (cashtrack)

## Project structure + tech
- `api/` — protobuf API definitions (ConnectRPC). Generated Go/TS code via Buf.
- `backend/` — Go service (ConnectRPC handlers, pgx, sqlc, google/wire).
  - Generated code in `backend/gen/` (protobuf + connect) and `backend/gen/db` (sqlc).
- `db/` — Postgres migrations (`migrations/`), `query.sql`, and generated `schema.sql`.
- `frontend/` — SvelteKit + Vite app (TypeScript, Tailwind CSS + DaisyUI). Generated TS clients in `frontend/src/lib/gen`.
- `docker/` — compose files for dev/prod and generation.

Core tools/tech:
- Go, ConnectRPC, Buf, sqlc, Goose, Postgres, google/wire
- SvelteKit, Vite, Tailwind CSS, DaisyUI, TypeScript

## Project structure (detailed)
- `backend/cmd/server/main.go` — server entry point: app initialization, ReportProcessor start, HTTP server start.
- `backend/server.go` — HTTP server, CORS, SPA file server, `/health`.
- `backend/handlers.go` — list of HTTP/ConnectRPC handlers mounted into the server.
- `backend/*_handlers.go`, `backend/*_service.go`, `backend/*.go` — business logic, ConnectRPC handlers, reports/transactions processing.
- `api/v1/*.proto` — protobuf contracts (Go/TS generated via Buf).
- `db/migrations/` — Postgres migrations; `db/query.sql` — sqlc queries.
- `frontend/src/routes/` — SvelteKit pages; `frontend/src/lib/` — API clients, stores, components.

## Frontend pages (routes)
- `/` — `frontend/src/routes/+page.svelte`: welcome and short app description.
- `/login` — `frontend/src/routes/login/+page.svelte`: Google Sign‑In, redirect to `/auth`.
- `/transactions` — `frontend/src/routes/transactions/+page.svelte`: transactions list, filters, summary, category updates, Sankey visualization.
- `/categories` — `frontend/src/routes/categories/+page.svelte`: categories & rules CRUD, apply and reorder rules.
- `/import` — `frontend/src/routes/import/+page.svelte`: CSV report upload, list, download, delete.
- `/todo` — `frontend/src/routes/todo/+page.svelte`: todo list, add/remove, random items.
- `/greet` — `frontend/src/routes/greet/+page.svelte`: greeting demo page.
- Layout — `frontend/src/routes/+layout.svelte`: global nav, user loading, Google GSI prompt, logout.
- `frontend/src/routes/+layout.ts`: `prerender = true`.

## Go handlers (backend)
- `backend/auth.go` — `AuthHandler`: HTTP `/auth` (Google credential → session → cookie → redirect).
- `backend/auth_service.go` — `AuthServiceHandler`: ConnectRPC AuthService (`Me`, `Logout`) + auth interceptor.
- `backend/todo.go` — `TodoHandler`: ConnectRPC TodoService (List/Add/Remove/AddRandom) + auth.
- `backend/greet.go` — `GreetHandler`: ConnectRPC GreetService (Greet).
- `backend/transaction_handlers.go` — `TransactionServiceHandler`: ConnectRPC TransactionService (ListTransactions, UpdateTransactionCategory) + auth.
- `backend/category_handlers.go` — `CategoryServiceHandler`: ConnectRPC CategoryService (categories & rules, Apply/Reorder) + auth.
- `backend/report_handlers.go` — `ReportServiceHandler`: ConnectRPC ReportService (Upload/List/Download/Delete) + auth.

## Protobuf services (api/v1/*.proto)
- `AuthService` (`api/v1/auth.proto`): `Me`, `Logout`.
- `CategoryService` (`api/v1/categories.proto`): categories CRUD, rules CRUD, `ApplyCategoryRules`, `ReorderCategoryRules`.
- `TransactionService` (`api/v1/transactions.proto`): `ListTransactions`, `UpdateTransactionCategory`.
- `ReportService` (`api/v1/reports.proto`): `UploadReport`, `ListReports`, `DownloadReport`, `DeleteReport`.
- `TodoService` (`api/v1/todo.proto`): `List`, `Add`, `Remove`, `AddRandom`.
- `GreetService` (`api/v1/greet.proto`): `Greet`.

## Run tests
- All tests/checks:
  - `make test`
    - Backend: `go test ./backend/...`
    - Frontend: `npm --prefix frontend run check`

## Regenerate API (ConnectRPC) and DB (sqlc)
Preferred (full regen via dockerized generator):
- `make generate`
  - Runs Buf codegen (ConnectRPC) for Go + TS
  - Runs migrations, dumps schema, regenerates `db/schema.sql`
  - Runs `sqlc generate`
  - Runs `wire ./...`

Fresh generator run (rebuild container):
- `make generate-fresh`

Notes:
- Protobuf sources: `api/v1/*.proto`
- Buf config: `buf.yaml`, `buf.gen.yaml`
- sqlc config: `sqlc.yaml`
