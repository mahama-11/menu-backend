# Menu Backend Guide

## 1. Purpose

This guide defines the initial scope of `v-menu-backend`.

## 1.1 Documentation Index

- [openapi/README.md](openapi/README.md): Swagger/OpenAPI generation for frontend-facing Menu APIs

## 2. Intended Capability Scope

Should belong here:

- `/menu/assets/*`
- `/menu/jobs/*`
- `/menu/providers/*`
- `/menu/templates/*`
- `/menu/exports/*`
- `/menu/history/*`
- `/menu/workspaces/*`
- `/menu/credits/*` (product-specific consumption rules)
- `/menu/reports/*` (product-owned reporting semantics)

Should not belong here:

- Shared login/register/user/org truth
- Shared entitlement/subscription base
- Shared payment gateway adapters
- Shared wallet / coupon / reward ledgers
- Shared metering event ingestion

## 3. Integration Principle

When Menu AI needs shared identity or platform data, call the platform service through stable APIs.
Do not duplicate platform tables as a second source of truth.

## 4. Engineering Status

- Go service skeleton initialized
- Product API contracts not finalized
- Current implementation intentionally stays minimal until platform-service contracts stabilize
- Database and Redis initialization are now wired through the same infrastructure-style config shape used across backend services.
- Platform integration is no longer limited to access context lookup. `internal/platform/client.go` now starts exposing commercialization-facing internal API methods for reservations, metering, settlement queries, wallet queries, incentive queries, and commercial route resolution, so Menu workflows can orchestrate platform billing truth instead of duplicating it locally.
- Menu P0 registration is designed as an orchestration boundary, not a second source of truth: Menu accepts frontend-facing register/login requests, maps `Restaurant Name` to the platform org creation input, and lets platform own the actual user/org/wallet/reward truth. Signup bonus policy (`app.signup_bonus_credits`) stays in Menu, but reward and wallet ledgers are issued through platform APIs.
- Menu backend now also follows the same response/error-code normalization direction as platform: `pkg/response` provides unified success/error envelopes, field-aware bind validation responses, and shared code-to-status mapping so auth, authz, middleware, and protected Menu APIs do not handcraft JSON responses independently.
- Frontend P0 auth/credits flow now has concrete Menu-facing endpoints under the long-term product namespace: `POST /api/v1/menu/auth/register`, `POST /api/v1/menu/auth/login`, `GET /api/v1/menu/auth/session`, and `GET /api/v1/menu/user/credits`. Menu owns the frontend contract and orchestration, while platform remains the source of truth for identity, org membership, rewards, and wallet balances.
- Menu now also owns product-facing user state beyond auth bootstrap: `GET /api/v1/menu/user/profile`, `PATCH /api/v1/menu/user/profile`, and `GET /api/v1/menu/user/activities` are backed by Menu-local preference/activity persistence plus platform user/org aggregation. This keeps frontend APIs product-oriented while preserving clear truth boundaries.
- Swagger/OpenAPI annotations now cover the first frontend P0 endpoints, and `./scripts/gen-swagger.sh` can generate the initial Menu API spec for frontend integration.
- Backend-owned frontend follow-up logic is now landing directly: Menu persists product-side `language_preference` and `activities`, aggregates platform user/org/wallet truth for `profile` and `credits`, and updates platform user/org profile through internal APIs instead of pushing these modeling decisions to frontend code.
- Auth/session contract should now be treated as authoritative for product access: `register`, `login`, and `session` return an explicit `access` object from Menu authz resolution, so frontend should stop inferring Menu access from `org_id`, `orgs`, or absent entitlement arrays.
- Error semantics are also part of the contract now: frontend should continue using numeric `code` for broad class handling, but it should prefer `error_code` for stable branch logic and `error_hint` for friendly user messaging instead of mapping every `401` or `1001` to a single hardcoded string.
- Engineering baseline is now treated as non-optional infrastructure rather than future polish: Menu initializes structured JSON logging, tracing provider bootstrap, request-scoped `request_id` / `trace_id`, Prometheus-style metrics exposure, structured access logs, and audit persistence for key mutations such as register, login, and profile update.
- Database governance is now expected to follow the same baseline: Menu schema migration should be driven from `internal/storage` under `database.auto_migrate_enabled`, and all Menu-owned tables should use the configured `database.table_prefix` (default `menu_`) so cross-project shared databases stay understandable and operable over time.
- To keep that rule enforceable rather than aspirational, CI guardrails should block new `gorm.Open(...)` usage outside `internal/storage` and block custom `TableName()` overrides inside `internal/models`. Table naming must stay centrally controlled by the Menu naming strategy and migration entrypoint.
