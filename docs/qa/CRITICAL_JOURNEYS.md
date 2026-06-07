# Menu Critical Journeys QA

This document defines the backend/API side of Menu critical journey acceptance. It complements `menu-frontend/docs/acceptance-tdd-governance.md` and must stay aligned with frontend routes and Platform runtime contracts.

## Status vocabulary

- `PASS`: Menu API contract, product persistence, Platform contract, and external runtime dependency passed.
- `PASS_WITH_NOTES`: Menu contract passed; non-blocking notes remain.
- `PARTIAL_PASS`: unit/static/API checks passed, but live runtime/browser evidence is missing.
- `BLOCKED`: Menu is wired correctly but an external dependency such as ComfyUI 8188 prevents terminal provider completion.
- `FAIL`: Menu frontend-facing API, persistence, or fail-closed behavior is broken.

## CJ-01 Auth / package activation / allowance

Routes:

- `POST /api/v1/menu/auth/register`
- `POST /api/v1/menu/auth/login`
- `GET /api/v1/menu/auth/session`
- `GET /api/v1/menu/user/wallet-summary`
- `GET /api/v1/menu/user/quota-summary`

Required backend evidence:

- Registration and login call Platform package activation idempotently for `product_code=menu`.
- Activation grants `menu.render.call` allowance for the active organization.
- Session response carries `access.active_org_id`, `has_menu_access`, and product permissions.
- API responses do not expose internal package activation/provider/runtime implementation fields.

Existing tests:

- `internal/modules/auth/service_package_activation_test.go`
- `internal/modules/user/service_platform_mock_test.go`

## CJ-02 Template Center → Studio handoff

Routes:

- `GET /api/v1/menu/template-center/meta`
- `GET /api/v1/menu/template-center/catalog`
- `GET /api/v1/menu/template-center/catalog/:templateID`
- `POST /api/v1/menu/template-center/catalog/:templateID/use`

Required backend evidence:

- List/detail exposes structured `input_slots`, target outputs, plan lock, and strategy policy.
- Use Template returns `prefilled_job`, `resolved_strategy`, `template_id`, `template_version_id`, and target route.
- `ask_for_required_input` stays a readiness sentinel and never reaches runtime as an executable mode.

Existing tests:

- `internal/modules/templatecenter/service_test.go`
- `internal/modules/templatecenter/handler_test.go`

## CJ-03 Single image generation

Routes:

- `POST /api/v1/menu/studio/assets`
- `POST /api/v1/menu/studio/jobs`
- `GET /api/v1/menu/studio/jobs/:jobID`
- `GET /api/v1/menu/studio/history/jobs`

Required backend evidence:

- One source asset produces `input_mode=image_to_image`.
- Runtime manifest includes `source_asset_ids`, role-aware `source_assets`, prompt snapshot, and charge snapshot.
- Status/stage transitions are persisted and exposed as queued/running/completed/failed/canceled.
- Late non-terminal callbacks cannot overwrite terminal completed/failed jobs.

Existing tests:

- `internal/modules/studio/service_test.go` (history/library/readback cases)
- `internal/modules/studio/generation_strategy_test.go`

## CJ-04 Four-slot multi-image generation

Required roles:

- `dish_photo`
- `brand_logo`
- `menu_reference`
- `style_reference`

Required backend evidence:

- Exactly four role-aware source assets resolve to `input_mode=multi_image`.
- More than four source assets fails closed before runtime dispatch with typed error `STUDIO_SOURCE_ASSETS_LIMIT_EXCEEDED`.
- Runtime manifest preserves roles.
- Platform route policy must only choose providers that declare `multi_image`; `volcengine` must not silently consume multi-image jobs.
- Current expected provider for real multi-image route is `comfyui_bridge`; real completion may be `BLOCKED by ComfyUI 8188` when downstream core is unavailable.

Existing tests:

- `internal/modules/studio/generation_strategy_test.go`
- `internal/modules/studio/provider_normalize_test.go`
- Platform route policy tests in `platform-backend` should cover provider capability filtering.

## CJ-05 Insufficient allowance / settlement state

Required backend evidence:

- Reserve happens before runtime dispatch.
- Insufficient allowance prevents real runtime job creation.
- Consume/finalize/release are idempotent and observable in charge summary.
- Frontend-facing errors use stable `error_code` / `error_hint` while hiding raw internal/provider/runtime strings.

Existing tests:

- Studio billing reserve/finalize/release tests under `internal/modules/studio/service_test.go`.
- User commercial consumption tests under `internal/modules/user/service_test.go`.

## CJ-06 History / Library

Routes:

- `GET /api/v1/menu/studio/history/jobs`
- `GET /api/v1/menu/studio/library/assets`
- `GET /api/v1/menu/studio/assets/:assetID/content`

Required backend evidence:

- Source assets, result assets, and selected variant are shown through Menu-owned read models.
- Failed tasks expose understandable stage messages.
- Raw provider/runtime/callback payloads are not frontend-visible.

## Smoke command

Use the HTTP-only contract smoke when a Menu service is running:

```bash
MENU_BASE_URL=http://127.0.0.1:8196 node scripts/menu-contract-smoke.mjs
```

Optional authenticated mode:

```bash
MENU_SMOKE_AUTH=1 MENU_SMOKE_EMAIL=qa-$(date +%s)@example.com MENU_SMOKE_PASSWORD='...' node scripts/menu-contract-smoke.mjs
```

The smoke is intentionally no-repo-write and should not create temporary Go files.
