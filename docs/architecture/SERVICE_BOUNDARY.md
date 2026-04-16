# Menu Backend Service Boundary

## 1. Menu-Owned Concerns

The following concerns are product-owned:

- Media asset lifecycle
- AI workflow orchestration
- Provider routing strategy
- Export formats and templates
- Product-specific credits deduction rules
- Product-specific dashboard and KPI semantics

## 2. Platform-Consumed Concerns

The following concerns should be consumed from the platform service:

- User identity
- Organization context
- Membership
- RBAC base
- Entitlement base
- Subscription base
- Payment / refund infrastructure base
- Metering and billing base

## 3. Rule Of Thumb

If a capability can be reused unchanged by KYC, Attendance, and Menu AI, it likely belongs to the platform.
If it depends on Menu AI product semantics, it belongs here.

## 4. Service Interaction

Recommended interaction model:

- `v-menu-frontend` -> `v-platform-backend` for login, register, `/me`, and org switch.
- `v-menu-frontend` -> `v-menu-backend` for menu product APIs.
- `v-menu-backend` -> `v-platform-backend` internal APIs for membership, org context, entitlement, or permission checks when required.

Do not turn `v-menu-backend` into a generic proxy for browser login flows.
