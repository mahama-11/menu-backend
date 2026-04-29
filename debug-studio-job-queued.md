# Debug Session: studio-job-queued

- Status: OPEN
- Symptom: Studio 发起任务后长期停留在 `status=queued`, `stage=queued`, `stage_message="Runtime job queued"`
- Environment: 服务器 `ssh mix`

## Hypotheses

1. Studio worker 未启动，任务入队后无人消费。
2. Worker 已启动，但 Redis/queue 配置错误，未消费 `studio:default`。
3. Worker 消费后立即失败，但状态推进或日志链路异常，前端只看到 queued。
4. 部署使用了错误配置文件，`studio` 相关配置与预期不一致。

## Evidence Log

- `mix` 上 `v-menu-backend-dev` 配置中未看到本地 studio worker 相关启动证据，且仓库内不存在独立的本地 studio worker 入口，排除 “本地 worker 没消费 Redis 队列” 为主因。
- `platform.runtime_jobs` 中 `source_id=job_1777212641964998195` 的记录状态为 `failed`，说明 platform 已经接单并结束，不是仍停留在平台队列。
- 当前需进一步验证 platform 失败后是否成功回调 `/studio/internal/runtime-update` 到 menu，以及 menu 是否接受并落库状态。
- `mix` 上 `v-platform-backend-dev` 的 `config.dev.yaml` 中 `runtime.product_endpoints.menu.base_url` 配成了 `http://v-menu-backend-dev:8395`，但 `v-menu-backend-dev` 实际端口来自 `MENU_PORT=8196`。
- `v-menu-backend-dev` 日志中仅看到前端对 `/api/v1/menu/studio/jobs/:jobID` 的轮询，没有任何 `/internal/v1/menu/studio/jobs/:jobID/runtime` 或 `/results` 命中记录，佐证 platform 回调未送达 menu。
- `menu` internal 路由使用 `RequireInternalService(cfg.Security.ServiceSecretKey)` 保护，当前 `service_secret_key=menu-service-secret`，与 platform `product_endpoints.menu.secret` 一致，排除 shared secret 不匹配为主因。

## Current Conclusion

1. 任务停在 `queued` 的直接原因不是未入队，而是 `platform -> menu` 内部回调地址端口配置错误，导致 menu 永远收不到失败/结果回写。
2. 对应的 platform runtime job 已经失败，因此修复回调后，menu 侧预期会从 `queued` 变成 `failed`，还需要继续调查 provider 失败原因。

## Fix Applied

- 将 `v-platform-backend/config.dev.yaml` 中 `runtime.product_endpoints.menu.base_url` 从 `http://v-menu-backend-dev:8395` 修正为 `http://v-menu-backend-dev:8196`。
- 将 `v-platform-backend/config.prod.yaml` 中 `runtime.product_endpoints.menu.base_url` 从 `http://v-menu-backend:8295` 修正为 `http://v-menu-backend:8096`。
