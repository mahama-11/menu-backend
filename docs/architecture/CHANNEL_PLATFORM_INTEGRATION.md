# Channel Platform Integration

## 1. Goal

This document defines how `v-menu-backend` integrates with the platform-owned channel revenue share capability.

It answers:

- which Menu flows create or update channel bindings
- which Menu business events should be reported as channel charge / refund events
- which Menu-facing APIs should aggregate platform channel data for frontend consumption
- how Menu keeps product semantics without leaking platform settlement internals directly to frontend

## 2. Boundary

Menu owns:

- product signup flow
- product business charge lifecycle
- product-facing aggregation APIs
- product wording and dashboard semantics

Platform owns:

- channel partner / program / binding truth
- commission policy truth
- commission / clawback / settlement ledger truth
- settlement batch lifecycle

Menu must not duplicate platform settlement tables locally.

## 3. Integration Points

### 3.1 Signup Binding

Menu auth/register should:

1. accept optional `channel_code`
2. complete user + org creation first
3. best-effort call platform internal `channel-bindings`
4. never fail successful signup because channel binding follow-up fails

This mirrors the existing referral conversion follow-up pattern.

### 3.2 Charge Reporting

Menu Studio commercial flow should report channel charge events when the charge becomes effective.

Current preferred entry point:

- charge finalization / settlement success in Studio commercial flow

The first integration scope should cover:

- wallet / credit based studio charges
- generation job charge finalization

Future product modules can reuse the same reporting client for:

- subscriptions
- wallet recharge
- other billable product events

### 3.3 Refund / Reverse Reporting

Menu should report channel refund events when a previously effective charge is reversed or released after commitment.

Current preferred entry points:

- settled charge reverse path
- explicit release / reversal path when the product determines the commercial event should be canceled

## 4. Product-facing APIs

Menu should expose product routes instead of forwarding platform models directly:

- `GET /api/v1/menu/channel/me/overview`
- `GET /api/v1/menu/channel/me/commissions`
- `GET /api/v1/menu/channel/me/settlements`
- `GET /api/v1/menu/channel/current-binding`
- `POST /api/v1/menu/channel/me/preview`
- `GET /api/v1/menu/channel/me/adjustments`
- `POST /api/v1/menu/channel/me/adjustments`

Menu aggregation rules:

- keep product wording consistent with dashboard language
- reshape settlement batch / item into frontend-friendly summaries
- keep platform raw IDs available when useful for debugging, but do not force frontend to understand internal settlement transitions

## 5. Suggested Module Shape

Create a dedicated `channel` module in `v-menu-backend`.

Recommended responsibilities:

- `internal/modules/channel/service.go`
  - aggregate current binding, commissions, settlements
- `internal/modules/channel/handler.go`
  - frontend-facing APIs
- `internal/modules/channel/types.go`
  - product response structs

Extend existing modules instead of duplicating flows:

- `auth` calls platform binding helper
- `studio` calls platform charge / refund helpers
- `platform/client.go` owns internal channel API calls

## 6. Failure Handling

### 6.1 Binding

- binding calls are best-effort follow-up
- failures should be logged and audited
- successful signup must not be rolled back because of binding failure

### 6.2 Charge / Refund Events

- channel reporting should be idempotent by product event IDs
- if platform call fails, Menu should log enough context for replay
- if the product financial action already succeeded, do not roll back product success purely because channel reporting failed

## 7. Current Implementation Order

1. extend platform client with channel methods
2. add Menu `channel` aggregation module
3. integrate `auth/register` optional channel binding
4. integrate `studio commercial` charge event reporting
5. integrate `studio commercial` reverse / refund reporting
6. expose frontend-facing channel APIs
7. update OpenAPI / docs / tests together

## 7.1 Current Implementation Status

Current code reality already includes:

- platform client support for channel bindings, commissions, clawbacks, settlement batches, settlement items, charge events, and refund events
- dedicated Menu `channel` module with:
  - `GET /api/v1/menu/channel/current-binding`
  - `GET /api/v1/menu/channel/me/overview`
  - `GET /api/v1/menu/channel/me/commissions`
  - `GET /api/v1/menu/channel/me/settlements`
  - `POST /api/v1/menu/channel/me/preview`
  - `GET /api/v1/menu/channel/me/adjustments`
  - `POST /api/v1/menu/channel/me/adjustments`
- best-effort signup binding follow-up in `auth/register` via optional `channel_code`
- best-effort Studio charge reporting after successful charge finalization
- Menu authz bootstrap entries for `menu.channel.read` and `menu.channel.manage`

Current implementation also adds a product-owned operations surface over platform advanced accounting:

- `preview`
  - product-facing dry-run for channel policy resolution
  - returns partner/program enrichment plus policy version / assignment / profit snapshot result
- `adjustments`
  - product-facing list/create APIs for manual credit, manual debit, reprice delta, cost true-up, and dispute-resolution adjustments
  - keeps frontend away from direct platform internal route semantics

Current known gap in Menu integration:

- explicit settled charge reverse / refund reporting is not yet wired from a concrete Menu product flow because current Studio integration finalizes charges but does not yet expose a product-owned settled refund callback path

## 8. Final Rule

Menu is the product integration and aggregation layer.
Platform remains the source of truth for channel revenue share accounting and settlement.
