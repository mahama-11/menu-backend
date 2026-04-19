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

- `v-menu-frontend` -> `v-menu-backend` for Menu-owned browser-facing routes, including product auth/session bootstrap, profile/activities, Studio, wallet-history, referral, and share-post flows.
- `v-menu-backend` -> `v-platform-backend` internal APIs for identity truth, org membership, permission/access checks, wallet summaries, reward/commission settlement, route resolution, and other shared commercialization capabilities.
- `v-menu-frontend` should not stitch platform internal models directly for product pages; Menu owns the product contract exposed to the browser.

Do not turn `v-menu-backend` into a blind generic proxy. Menu should own product-shaped contracts and orchestration, while platform remains the truth source for reusable shared capabilities.
