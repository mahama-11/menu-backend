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
- Menu observability baseline is now aligned with platform expectations: frontend-facing core handlers emit request-correlated OTel spans, and `/metrics` is backed by the Prometheus official registry/counter/histogram pipeline instead of ad-hoc text rendering.
- Error handling should now prioritize semantic fields instead of guessing from generic status text alone: `error_code` is the stable machine-facing key (for example `INVALID_CREDENTIALS`, `TOKEN_EXPIRED`, `PROFILE_UPDATE_FAILED`) and `error_hint` is the user-facing recovery hint that frontend can display directly or localize.
- Referral and commission product APIs are now exposed from Menu itself rather than requiring frontend to call platform internal incentive routes directly: use Menu routes for `referrals/programs`, `referrals/me/overview`, `referrals/me/codes`, `referrals/me/conversions`, and `referrals/me/commissions`.
