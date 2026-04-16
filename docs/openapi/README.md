# Menu OpenAPI

## Current Scope

The first Swagger / OpenAPI scope focuses on frontend P0 integration:

- `POST /api/v1/menu/auth/register`
- `POST /api/v1/menu/auth/login`
- `GET /api/v1/menu/auth/session`
- `GET /api/v1/menu/user/profile`
- `PATCH /api/v1/menu/user/profile`
- `GET /api/v1/menu/user/credits`
- `GET /api/v1/menu/user/activities`

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
- Error handling should now prioritize semantic fields instead of guessing from generic status text alone: `error_code` is the stable machine-facing key (for example `INVALID_CREDENTIALS`, `TOKEN_EXPIRED`, `PROFILE_UPDATE_FAILED`) and `error_hint` is the user-facing recovery hint that frontend can display directly or localize.
