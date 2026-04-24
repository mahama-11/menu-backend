# Studio Frontend Integration

## 1. Product Intent

Menu Studio is not modeled as a loose prompt tool.

The frontend should treat it as a stable product domain built around:

- `StudioAsset`
- `StylePreset`
- `GenerationJob`
- `GenerationVariant`

This is the core path:

1. user prepares assets
2. user selects a style preset
3. frontend creates a generation job
4. backend enqueues provider execution
5. frontend observes staged job progress
6. user selects one variant or continues editing

## 2. Core Domain Objects

### 2.1 StudioAsset

Represents source images, generated images, masks, and references.

Important fields:

- `id`
- `asset_type`: `source | generated | mask | reference`
- `source_type`: `upload | import | generated`
- `status`
- `storage_key`
- `source_url`
- `preview_url`
- `metadata`

Important access rule:

- For platform-stored assets, frontend should treat `source_url` / `preview_url` as Menu-owned signed content routes.
- Browser clients should not assume a public `/storage/*` path exists on platform.

### 2.2 StylePreset

Represents a reusable style definition, not just a raw prompt string.

Important fields:

- `style_id`
- `name`
- `dimensions[]`
- `tags[]`
- `visibility`
- `version`
- `parent_style_id`
- `execution_profile`

Frontend should display style cards using:

- `name`
- `description`
- `dimensions`
- `tags`
- `preview_asset_id`

Frontend should not assume that `execution_profile` is user-editable in every stage.

### 2.3 GenerationJob

Represents one execution request.

Important fields:

- `job_id`
- `mode`: `single | batch | variation | refinement`
- `input_mode`: `text_to_image | image_to_image`
- `status`
- `stage`
- `stage_message`
- `provider`
- `source_asset_ids[]`
- `params_snapshot.prompt` when text-driven generation is used
- `style_preset_id`
- `requested_variants`
- `progress`
- `queue_position`
- `eta_seconds`
- `selected_variant_id`
- `charge`
- `child_jobs[]`
- `variants[]`

`charge` is the frontend-facing commercial snapshot for a job.

Important fields:

- `billable`
- `charge_mode`
- `resource_type`
- `billable_item_code`
- `status`
- `estimated_units`
- `final_units`
- `charge_priority_asset_codes[]`
- `wallet_asset_code`
- `wallet_debited`
- `credits_consumed`
- `quota_consumed`
- `gross_amount`
- `net_amount`
- `failure_code`
- `failure_message`

### 2.4 GenerationVariant

Represents one output result of a job.

Important fields:

- `variant_id`
- `asset_id`
- `status`
- `index`
- `score`
- `is_selected`
- `parent_variant_id`

## 3. Job State Model

Frontend should use both `status` and `stage`.

### 3.1 Status

- `queued`
- `processing`
- `completed`
- `failed`
- `canceled`

### 3.2 Stage

Current meaningful stages:

- `queued`
- `dispatching`
- `provider_accepted`
- `running`
- `retry_scheduled`
- `completed`
- `failed`
- `canceled`

### 3.3 Rendering Guidance

Frontend should not render only `progress`.

Recommended UI priority:

1. `stage_message`
2. `status`
3. `progress`
4. `queue_position`
5. `eta_seconds`

Example mapping:

- `queued`: show waiting state and queue hint
- `dispatching`: show “submitting to generation engine”
- `provider_accepted`: show “accepted by provider”
- `running`: show “AI is generating”
- `retry_scheduled`: show temporary recovery state rather than failure
- `completed`: enable result comparison and selection
- `failed`: show retry-friendly error copy

## 4. API Contract

### 4.1 User-facing APIs

- `GET /api/v1/menu/studio/assets`
- `GET /api/v1/menu/studio/library/assets`
- `POST /api/v1/menu/studio/assets`
- `GET /api/v1/menu/studio/styles`
- `POST /api/v1/menu/studio/styles`
- `GET /api/v1/menu/studio/styles/:styleID`
- `POST /api/v1/menu/studio/styles/:styleID/fork`
- `GET /api/v1/menu/studio/jobs`
- `GET /api/v1/menu/studio/history/jobs`
- `POST /api/v1/menu/studio/jobs`
- `GET /api/v1/menu/studio/jobs/:jobID`
- `POST /api/v1/menu/studio/jobs/:jobID/select-variant`
- `POST /api/v1/menu/studio/jobs/:jobID/cancel`

### 4.2 Internal execution APIs

These are not for browser clients.

- `POST /internal/v1/menu/studio/jobs/:jobID/runtime`
- `POST /internal/v1/menu/studio/jobs/:jobID/results`

### 4.3 Related wallet / referral APIs

These APIs are important for frontend commercial UX around Studio:

- `GET /api/v1/menu/user/wallet-summary`
- `GET /api/v1/menu/user/wallet-history`
- `GET /api/v1/menu/user/audit-history`
- `GET /api/v1/menu/referrals/me/codes`
- `POST /api/v1/menu/referrals/me/codes/ensure`
- `GET /api/v1/menu/referrals/me/commissions`
- `POST /api/v1/menu/referrals/me/commissions/redeem`
- `GET /api/v1/menu/share/posts`
- `POST /api/v1/menu/share/posts`
- `GET /api/v1/menu/share/posts/:shareID`
- `GET /api/v1/menu/share/public/:token`
- `GET /api/v1/menu/share/public`
- `POST /api/v1/menu/share/public/:token/view`
- `GET /api/v1/menu/share/me/favorites`
- `GET /api/v1/menu/share/posts/:shareID/engagement`
- `POST /api/v1/menu/share/posts/:shareID/like`
- `POST /api/v1/menu/share/posts/:shareID/favorite`

## 4.4 Share Domain Boundary

`share/posts` is the product sharing boundary for Studio outputs.

The important rule is:

- `StudioAsset` remains the source-of-truth media object.
- `SharePost` is a publishable product object built on top of an asset.
- likes / favorites / views belong to `SharePost`, not to `StudioAsset`.

This keeps social-style engagement, visibility, and share copy from polluting the asset truth model.

Recommended frontend layering:

1. create or list `share/posts` inside authenticated product pages
2. use `GET /share/public` as the product-owned public feed for discovery
3. use `GET /share/me/favorites` as the authenticated saved-view for internal re-engagement
4. open public share pages through `share_url`
5. use browser-native share / copy-link / download as the first universal distribution path
6. treat external platform publishing as adapters layered on top of `SharePost`, not as fields inside `StudioAsset`

## 5. Frontend Flow Guidance

### 5.1 Single Job

Recommended flow:

1. if `image_to_image`, register source asset
2. if `text_to_image`, collect prompt text first
3. load style presets
4. load wallet summary
5. create generation job
6. poll `GET /studio/jobs/:jobID`
7. show variants when completed
8. call `select-variant` for final choice

Current backend truth for source/result media is:

- source uploads are first stored through platform storage and persisted by `storage_key`
- runtime/provider input is derived from stored bytes instead of relying on a public asset URL round-trip
- generated result assets are imported back into platform storage before Menu records them
- browser-facing asset access should flow through signed Menu content URLs

`/studio` is now treated as an authenticated product workspace. Frontend should redirect unauthenticated users to `/login` before attempting protected Studio APIs.

For retrieval UX, frontend should also use:

- `GET /studio/history/jobs` for process-oriented history
- `GET /studio/library/assets` for asset-oriented library

### 5.2 Batch Job

Frontend should treat batch as:

- one root job
- multiple child jobs

Do not assume batch is a single giant result set.

Recommended UI:

- root progress summary
- child job grid/list
- per-child state
- final aggregate status

### 5.3 Refinement / Variation

Frontend should create another `GenerationJob` and pass:

- `parent_job_id`
- or `parent_variant_id`

This keeps multi-round editing inside the same stable job model.

### 5.4 Library / Retrieval

Frontend should not force users to recover past work only through job lists.

Recommended split:

- `job history` page:
  - use `GET /studio/history/jobs`
  - focuses on process, stage, billing, and selected result
- `asset library` page:
  - use `GET /studio/library/assets`
  - focuses on reusable inputs/outputs, latest job context, and publish/share state

Recommended library filters:

- asset type
- status
- keyword query
- generated only
- published only

## 6. Polling / Refresh Strategy

Frontend does not need to know queue internals.

Recommended job refresh strategy:

- first 30 seconds: poll every `2s`
- after 30 seconds: poll every `5s`
- after 2 minutes: poll every `8-10s`
- stop polling when:
  - `status=completed`
  - `status=failed`
  - `status=canceled`
- stop polling after repeated transport failures as well; frontend should not retry forever on `GET /studio/jobs/:jobID`
- when repeated polling fails, surface a recoverable sync error and require explicit user refresh / retry

Optional optimization later:

- upgrade to SSE or websocket
- keep current API model unchanged

## 7. Commercial Model

Frontend should treat job charging as:

1. create job
2. backend attempts a reservation
3. if reservation succeeds, job is created and queued
4. if job completes, backend finalizes metering and settlement
5. if job fails or is canceled, backend releases the reservation

This means:

- frontend should not mark a request as finally charged when `POST /studio/jobs` returns success
- final charging truth is attached to job detail through `charge`
- failed or canceled jobs should be explained as "not finally charged" / "reservation released"

### 7.1 Recommended user-facing asset buckets

- `MENU_MONTHLY_ALLOWANCE`: monthly included allowance
- `MENU_PROMO_CREDIT`: promotional / referral reward balance
- `MENU_CREDIT`: main paid balance or redeemed commission balance
- `CommissionLedger`: earned commission that is not directly consumable until redeemed

### 7.2 Recommended consume priority

Frontend copy and help text should align with this order:

1. `MENU_MONTHLY_ALLOWANCE`
2. `MENU_PROMO_CREDIT`
3. `MENU_CREDIT`

`CommissionLedger` should not be treated as directly spendable balance in Studio create flow.

### 7.5 Audit / Operation Center

Frontend can use `GET /user/audit-history` to build a human-readable operation center:

- uploads
- style creation/fork
- job creation
- result writeback
- share post creation
- redeem actions

This should be presented as an operation log, not as a billing ledger.

### 7.6 Share Posts

Publishing is intentionally modeled outside core Studio assets/jobs.

Frontend should use `share/posts` for:

- publish a generated asset
- retrieve published share state
- show current share URL
- display placeholder engagement counters

Do not overload `StudioAsset` itself with future social fields.

### 7.3 How to read `charge`

- `billable=false`: root batch jobs or billing-disabled environments
- `status=reserved`: reservation succeeded and job is allowed to run
- `status=settled`: final charge is completed
- `status=released`: reservation was released because the job failed or was canceled
- `status=failed_need_reconcile`: reservation / finalize / release hit an abnormal path and needs operator attention

If settlement is completed, frontend can also read:

- `wallet_asset_code`
- `wallet_debited`
- `credits_consumed`
- `quota_consumed`
- `gross_amount`
- `net_amount`

These fields should be used for detail pages, billing drawers, and post-generation receipts.

### 7.4 Invite and redeem flows

Commercial UX for Studio is not complete unless frontend also supports referral and redeem entry points.

Recommended frontend capabilities:

1. load `GET /referrals/me/codes` or call `POST /referrals/me/codes/ensure`
2. display `invite_url` / `signup_url`
3. support copy/share CTA using `share_text`
4. on signup landing pages, parse `referral_code` from URL and call `GET /referrals/codes/:code/resolve`
5. after login, show `redeemable_commission` and call `POST /referrals/me/commissions/redeem`

`CommissionLedger` should not be treated as directly spendable Studio balance.
Only redeemed balance becomes consumable asset balance.

## 8. Create Failure Semantics

`POST /api/v1/menu/studio/jobs` no longer has only one generic create-failed meaning.

Frontend should handle these semantic `error_code` values:

- `STUDIO_BILLING_ALLOWANCE_INSUFFICIENT`
- `STUDIO_BILLING_CREDITS_INSUFFICIENT`
- `STUDIO_BILLING_WALLET_INSUFFICIENT`
- `STUDIO_BILLING_CONFIG_MISSING`
- `STUDIO_BILLING_UPSTREAM_FAILED`
- `STUDIO_JOB_CREATE_FAILED`

Recommended UX mapping:

- insufficient allowance / credits / wallet:
  - explain balance is not enough
  - show recharge or redeem entry
- config missing:
  - show service unavailable copy
  - do not ask user to retry repeatedly
- upstream failed:
  - show retryable service error
- generic create failed:
  - keep fallback error handling

## 9. What Frontend Should Not Assume

- do not assume style equals one category
- do not assume provider is fixed
- do not assume one job always means one output image
- do not assume batch returns only root-level variants
- do not assume `execution_profile` is always editable for end users

## 10. Recommended Frontend State Model

Suggested state buckets:

- `assetStore`
- `stylePresetStore`
- `generationJobStore`
- `selectedVariantStore`
- `walletSummaryStore`
- `walletHistoryStore`
- `commissionStore`
- `referralCodeStore`
- `assetLibraryStore`
- `jobHistoryStore`
- `auditHistoryStore`
- `sharePostStore`

Suggested job cache shape:

- list cache for `/studio/jobs`
- detail cache for `/studio/jobs/:jobID`
- optimistic variant selection state
- charge snapshot should come from job detail rather than client-side balance math

## 11. Near-term Frontend Priorities

Recommended order:

1. wallet summary + charge priority display
2. single-job create + detail page
3. create-failure semantic handling
4. variant selection flow
5. batch job detail UI
6. referral commission redeem entry
7. style fork/create UI
8. refinement flow

This order aligns with the current backend capability.
