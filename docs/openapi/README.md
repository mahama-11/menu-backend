# Menu OpenAPI

## Current Scope

The first Swagger / OpenAPI scope focuses on frontend P0 integration:

- `POST /api/v1/menu/auth/register`
- `POST /api/v1/menu/auth/login`
- `GET /api/v1/menu/auth/session`
- `GET /api/v1/menu/user/profile`
- `PATCH /api/v1/menu/user/profile`
- `GET /api/v1/menu/user/credits`
- `GET /api/v1/menu/user/wallet-summary`
- `GET /api/v1/menu/user/wallet-history`
- `GET /api/v1/menu/user/audit-history`
- `GET /api/v1/menu/user/activities`
- `GET /api/v1/menu/referrals/programs`
- `GET /api/v1/menu/referrals/codes/:code/resolve`
- `GET /api/v1/menu/referrals/me/overview`
- `GET /api/v1/menu/referrals/me/codes`
- `POST /api/v1/menu/referrals/me/codes/ensure`
- `POST /api/v1/menu/referrals/me/codes`
- `GET /api/v1/menu/referrals/me/conversions`
- `GET /api/v1/menu/referrals/me/commissions`
- `POST /api/v1/menu/referrals/me/commissions/redeem`
- `GET /api/v1/menu/channel/current-binding`
- `GET /api/v1/menu/channel/me/overview`
- `GET /api/v1/menu/channel/me/commissions`
- `GET /api/v1/menu/channel/me/settlements`
- `POST /api/v1/menu/channel/me/preview`
- `GET /api/v1/menu/channel/me/adjustments`
- `POST /api/v1/menu/channel/me/adjustments`
- `GET /api/v1/menu/share/posts`
- `POST /api/v1/menu/share/posts`
- `GET /api/v1/menu/share/posts/:shareID`

These endpoints are the recommended starting point for frontend integration.

## Generate OpenAPI

Install `swag` first:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Then run:

```bash
./scripts/gen-swagger.sh
```

Generated output will be written to:

```bash
docs/openapi/
```

## Notes

- Menu backend owns the frontend-facing API contract.
- Frontend should align to the long-term Menu route namespace: `/api/v1/menu/*`.
- Shared identity, org, reward, and wallet truth still live in `v-platform-backend`.
- Frontend should consume Menu APIs rather than calling platform internal APIs directly.
- All API responses now include request-scoped metadata such as `request_id`, and the service also emits `X-Request-ID` / `X-Trace-ID` response headers for correlation during frontend and backend debugging.
- Frontend auth bootstrap should now rely on the explicit `access` object returned by Menu auth/session APIs: `access.has_menu_access`, `access.menu_roles`, and `access.menu_permissions` replace any old client-side inference based on `user.orgs`, `organizations`, or missing entitlements.
- Menu registration now accepts optional `referral_code` for signup attribution. Register success is still owned by the main auth flow; referral conversion is a best-effort follow-up that should not block successful account creation.
- Referral product APIs now support the core frontend workflows directly:
  - pre-register referral code validation via `GET /referrals/codes/:code/resolve`, including reward policy description fields such as `reward_policy_desc`
  - idempotent org code provisioning via `POST /referrals/me/codes/ensure`
  - optional `status` filters on conversions and commissions
  - optional `program_code` / `status` filters on code listing
- Referral semantics now expose clearer anti-abuse outcomes through error codes such as `REFERRAL_ALREADY_CLAIMED`, `REFERRAL_SELF_INVITE_BLOCKED`, and `REFERRAL_TRIGGER_NOT_ELIGIBLE` so frontend can show business-friendly guidance instead of generic server failures.
- Commission rewards can now stay in-product rather than moving toward payout flows: Menu exposes `POST /referrals/me/commissions/redeem` to convert earned referral commissions into the configured credits asset for the current organization.
- Menu wallet semantics now expose multi-asset summaries for frontend adaptation: `GET /user/wallet-summary` returns product-scoped balances split across permanent balance, expiring reward credits, and cycle-reset allowance buckets.
- Menu now also exposes a product-facing history feed for frontend wallet, growth, and billing pages: `GET /user/wallet-history` aggregates Studio charge records, reward issuance, commission earning/redeeming, recharge-like wallet adjustments, and expiration events into one chronological API so frontend does not need to stitch multiple platform ledgers on its own.
- Menu now also exposes `GET /user/audit-history` so frontend can build an operation log and user audit center without calling admin-only tooling or direct storage queries.
- Menu observability baseline is now aligned with platform expectations: frontend-facing core handlers emit request-correlated OTel spans, and `/metrics` is backed by the Prometheus official registry/counter/histogram pipeline instead of ad-hoc text rendering.
- Menu studio (AI style processing) product domain APIs are now available for engineering-first rollout. These endpoints are intentionally designed around stable domain objects (style presets, generation jobs, variants) so future batch and multi-round refinement can be layered on without breaking the core contract:
  - `GET/POST /studio/assets` (register metadata, list assets)
  - `GET /studio/library/assets` (asset library with latest job/share context)
  - `GET/POST /studio/styles` (list/create style presets)
  - `GET /studio/styles/:styleID` (get a preset)
  - `POST /studio/styles/:styleID/fork` (derive a new preset)
  - `GET/POST /studio/jobs` (list/create generation jobs)
  - `GET /studio/history/jobs` (job/result history for retrieval UX)
  - `GET /studio/jobs/:jobID` (get job + variants)
  - `POST /studio/jobs/:jobID/results` (worker-controlled result callback)
  - `POST /studio/jobs/:jobID/cancel` (cancel generation job)
  - `POST /studio/jobs/:jobID/select-variant` (user selects one variant)
- Studio job execution is now queue-driven rather than DB polling: job creation enqueues Redis-backed worker tasks, provider dispatch is adapter-based, retries/timeouts are scheduled as delayed tasks, and batch jobs aggregate child jobs without coupling the frontend contract to a specific model vendor.
- Internal execution callbacks are intentionally separated from browser APIs: worker/provider updates use `/internal/v1/menu/studio/jobs/:jobID/runtime` and `/internal/v1/menu/studio/jobs/:jobID/results`, while frontend integration guidance is documented in `docs/architecture/STUDIO_FRONTEND_INTEGRATION.md`.
- Error handling should now prioritize semantic fields instead of guessing from generic status text alone: `error_code` is the stable machine-facing key (for example `INVALID_CREDENTIALS`, `TOKEN_EXPIRED`, `PROFILE_UPDATE_FAILED`) and `error_hint` is the user-facing recovery hint that frontend can display directly or localize.
- Referral and commission product APIs are now exposed from Menu itself rather than requiring frontend to call platform internal incentive routes directly: use Menu routes for `referrals/programs`, `referrals/me/overview`, `referrals/me/codes`, `referrals/me/conversions`, and `referrals/me/commissions`.
- Channel revenue share product APIs are now exposed from Menu itself as a product aggregation layer over platform channel accounting: use Menu routes for `channel/current-binding`, `channel/me/overview`, `channel/me/commissions`, `channel/me/settlements`, `channel/me/preview`, and `channel/me/adjustments`.
- Referral code responses are now frontend-ready rather than backend-only: `referrals/me/codes`, `referrals/me/codes/ensure`, and `referrals/me/codes` create responses include `invite_url`, `signup_url`, and `share_text`, so frontend can directly render copy/share CTA without inventing invite link rules locally.
- Publishing/social semantics now have an explicit product boundary: use `share/posts` to create and query publishable share objects with stable `share_url`, visibility, and placeholder engagement counters (`view_count`, `like_count`, `favorite_count`) without leaking future social behavior into raw `StudioAsset` rows.
