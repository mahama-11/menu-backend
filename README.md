# v-menu-backend

Menu AI 产品后端仓库的 Go 初始化工程。

## 目标

该仓库用于承载 `menu` 产品的独立业务后端能力，包括但不限于：

- 菜单图片/素材上传与管理
- AI 任务编排（增强、生图、文案、导出）
- 第三方视觉与生成 provider 适配
- 产品内任务历史、工作区、模板、导出能力
- menu 产品专属 credits 消耗与业务规则

## 当前状态

- 已选择 Go 作为当前实现技术栈。
- 已落最小可运行服务骨架与健康检查入口。
- 平台能力未来通过 `v-platform-backend` 接入，而不是复制用户/组织真相。

## 目录

```text
v-menu-backend/
├── AGENTS.md
├── README.md
├── cmd/
├── docs/
│   ├── BACKEND_GUIDE.md
│   └── architecture/
│       └── SERVICE_BOUNDARY.md
├── internal/
└── test/
```
