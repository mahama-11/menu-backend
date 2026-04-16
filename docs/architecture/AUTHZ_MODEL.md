# Menu Authz Model

## 1. Goal

Define a product-owned authorization model for `v-menu-backend` that can evolve independently from the shared platform identity layer.

## 2. Layer Split

- **Platform authentication**: `v-platform-backend` signs the token and owns identity, org membership, and product-entry entitlement.
- **Menu authorization**: `v-menu-backend` owns product-local roles, product-local permissions, and future resource-level access rules.

## 3. Current Foundation

Current request path for protected menu APIs:

1. Browser sends the platform-issued bearer token.
2. `v-menu-backend` verifies the JWT locally.
3. `v-menu-backend` calls platform internal access API to confirm user/org context.
4. `v-menu-backend` resolves menu roles and menu permissions locally.
5. The route or middleware checks the required menu permission.

## 4. Current Data Model

The current foundation tables are:

- `menu_roles`
- `menu_permissions`
- `menu_role_permissions`
- `menu_subject_roles`

This allows the product to grow from simple role mapping into explicit product-owned authorization.

## 5. Default Mapping

The first version uses platform org roles as the default source of menu roles:

- `owner/admin` -> `menu.workspace_admin`
- `viewer` -> `menu.viewer`
- everything else -> `menu.editor`

This keeps onboarding simple while leaving room for product-specific overrides in `menu_subject_roles`.

## 6. Why Menu Permissions Stay Out Of The Platform Token

Menu fine-grained permissions should not be embedded into the platform token because:

- product-specific permissions would pollute the shared platform token
- permission changes would become stale until token refresh
- different products would compete for token space and semantics

The platform token should carry stable identity context, not the full product ACL.

## 7. Future Evolution

Planned next steps:

- add org-scoped admin APIs for assigning `menu_subject_roles`
- support workspace/project/resource scopes in addition to org scope
- allow explicit product roles to override default platform-role mapping
- add entitlement checks for product packaging and monetization gates
