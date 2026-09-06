# ITCodeX 元数据管理端

基于 `web/docs/元数据管理端-产品需求文档.md` 实现的 **React 纯前端**管理控制台。

仅调用 `server` 已有接口：

- `/api/meta/*` — Collection / Field / Index / Script
- `/api/c/:collection/*` — 业务记录 CRUD

## 技术栈

- React 18 + TypeScript + Vite
- Ant Design 5
- TanStack Query
- React Router 6

## 快速开始

1. 安装 [Node.js 18+](https://nodejs.org/)
2. 启动后端（默认 `http://127.0.0.1:8000`）
3. 安装并启动前端：

```bash
cd web
npm install
npm run dev
```

浏览器打开 http://127.0.0.1:5173

开发环境已配置 Vite 代理：`/api` → `http://127.0.0.1:8000`。也可在「设置」页填写 API Base URL。

## 功能（MVP）

| 模块 | 能力 |
|------|------|
| 数据表广场 | 卡片列表、搜索、类型/分类筛选、删除、同步 |
| 创建向导 | 五类表、基本信息、预设字段、创建后 Sync |
| 工作室 · 概览 | 记录数、编辑显示名/分类/描述 |
| 工作室 · 结构 | 字段增删改、类型分组、校验基础项 |
| 工作室 · 数据 | 动态表格、搜索、排序、分页、增删改查抽屉 |
| 工作室 · 索引 | 创建/删除索引 |
| 工作室 · 脚本 | 钩子与自定义 API、模板、语法校验 |
| 工作室 · 高级 | Sync、危险删除、类型专属 options |
| 全局脚本 | 同上 |
| 设置 | API 地址、主题、页大小（localStorage） |
| 命令面板 | `Ctrl+K` / `⌘K` |

## 目录结构

```
web/
├── docs/                         # 产品需求文档
├── src/
│   ├── api/                      # HTTP 客户端与资源 API
│   ├── components/               # FieldRenderer、命令面板
│   ├── layouts/                  # 侧栏布局
│   ├── lib/                      # 设置、字段类型常量
│   ├── pages/                    # 页面
│   ├── router/                   # 路由
│   ├── styles/                   # 全局样式
│   └── types/                    # 与后端对齐的类型
├── package.json
└── vite.config.ts
```

## 说明

- 无登录页（后端暂无用户体系）
- Schema 驱动：不为具体业务表写死页面
- 关系字段 MVP 以降级为 ID/JSON 输入；完整记录选择器见需求文档二期
