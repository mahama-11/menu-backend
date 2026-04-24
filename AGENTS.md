# V-Menu Backend - Agent Context

> **CRITICAL INSTRUCTION**: This repository owns Menu AI business workflows. Reuse platform capabilities through service contracts instead of copying identity or billing truth locally.

## 1. Purpose

`v-menu-backend` is the planned product backend for the Menu AI line.

It should host:

- Asset management
- Job orchestration
- Provider adapters
- Export and template workflows
- Product-owned credits consumption rules
- Product-owned admin and reporting semantics

## 1.1 Current Runtime Choice

- Language/runtime: Go
- Current implementation state: product backend already active with Menu-facing auth/session bootstrap, user/profile aggregation, Studio asset/job/history APIs, wallet-history aggregation, referral/redeem flows, share-post publishing boundary, Redis-backed job queue runtime, and generated OpenAPI for frontend-facing routes
- Platform dependency direction: consume shared auth/org/RBAC/subscription primitives from `v-platform-backend`

## 2. Boundary Rules

- Product workflows belong here.
- Shared identity, org, RBAC, subscription, payment base, and metering base do not.
- Prefer calling platform APIs over directly coupling to shared data stores.

## 3. Documentation Index

- [**Backend Guide**](docs/BACKEND_GUIDE.md): Entry guide for Menu backend development.
- [**OpenAPI README**](docs/openapi/README.md): Swagger/OpenAPI generation entry for frontend-facing Menu APIs.
- [**Service Boundary**](docs/architecture/SERVICE_BOUNDARY.md): What belongs here versus the platform service.
- [**Authz Model**](docs/architecture/AUTHZ_MODEL.md): Product-owned authorization model and platform-role mapping strategy.
- [**Channel Platform Integration**](docs/architecture/CHANNEL_PLATFORM_INTEGRATION.md): Menu-side integration boundary for channel binding, charge reporting, refund reporting, and product aggregation APIs.
