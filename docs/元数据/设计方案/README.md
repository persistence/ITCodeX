# ITCodeX 元数据模块 - 设计方案

> 版本: v1.2
> 日期: 2026-09-05
> 技术栈: GoFrame v2 + MySQL 8 + CEL-Go + Yaegi
> **代码根路径**: [d:\code\ITCodeX\server](file:///d:/code/ITCodeX/server)

`需求描述/` 与 `api/` 来自 NocoBase，用作能力参考，**不是**本项目必须全部实现的规格。本目录才是 ITCodeX 的实现规格。

## 文档目录

| 序号 | 文档 | 说明 |
|------|------|------|
| 01 | [总体架构设计](./01-总体架构设计.md) | 整体架构分层、模块职责、启动流程、系统表结构 |
| 02 | [数据模型设计](./02-数据模型设计.md) | Database/Collection/Field/Repository 等核心结构体定义、内置字段类型 |
| 03 | [CEL校验设计](./03-CEL校验设计.md) | CEL-Go 环境配置、单字段/多字段联合校验、校验规则缓存、执行流程 |
| 04 | [Yaegi二开设计](./04-Yaegi二开设计.md) | Yaegi 脚本引擎、CRUD钩子、自定义API、沙箱环境、Path规范、脚本示例 |
| 05 | [API接口设计](./05-API接口设计.md) | HTTP 接口、与 NocoBase JS API 的对照、Filter 分期 |
| 06 | [项目结构设计](./06-项目结构设计.md) | `gf init` 目录、`g.Meta` 路由、Controller/Service、系统表 DAO |
| 07 | [MySQL集成设计](./07-MySQL集成设计.md) | Docker 本机 MySQL、类型映射、建表SQL、Schema同步、Filter到SQL翻译 |
| 08 | [测试方案](./08-测试方案.md) | 测试策略、测试框架、各模块测试用例、基准测试、覆盖率目标 |

## 对标 NocoBase 的范围

| 参考文档 | 本质 | 本项目用法 |
|----------|------|------------|
| [需求描述](../需求描述/) | NocoBase 产品需求（含页面、权限、工作流、外部数据源） | 指导字段/表类型范围，**不实现**页面与工作流 |
| [api/](../api/) | `@nocobase/database` **JS SDK** | 指导 Repository/Filter 语义，对外暴露为 **HTTP**，不做 JS 兼容层 |

### 做

- 主库 MySQL 8（Docker：root / 123456，库 `itcodex`）上的动态 Collection / Field
- 普通表的运行时建模与 schema 同步（`CREATE` / `ALTER`）
- HTTP：`/api/meta/*`、`/api/c/:collection/*`、`/api/custom/*`
- NocoBase 风格 Filter 的**子集**（见下方分期）
- CEL 单字段 / 多字段校验（用 CEL 覆盖需求里的 Joi 规则，不引入 Joi）
- Yaegi 钩子与自定义 API
- 关系字段与关联查询（第三阶段）
- 树 / 日历 / 评论 / 文件表的**语义增强**（第五阶段）

### 明确不做（至少 v1）

| 参考来源 | 能力 | 原因 |
|----------|------|------|
| 需求-数据表 | 数据库视图、SQL 表、继承表、外部表/FDW | 超出主库动态建模范围 |
| 需求-数据表 | 页面区块、权限、工作流 | 不属于本模块 |
| api-Database | 多方言（SQLite/PostgreSQL）、Umzug 式迁移 | 只做 MySQL + 启动时 sync |
| api-Database | 完整 JS 事件/插件生态 | 生命周期只服务 CEL / Yaegi |
| api-Operators | `$col`、`$exists`、日期族、数组族等 | 见 Filter 分期；未列入的默认不做 |
| 需求-文件表 | 完整对象存储引擎 | 第五期只存文件元数据，存储后端另议 |
| — | 独立用户体系 | `createdBy`/`updatedBy` 由请求上下文注入，无则留空 |

### 需求 / API → 设计决策 → 阶段

| 参考项 | 设计决策 | 阶段 |
|--------|----------|------|
| 普通表 + 预设系统字段 | 实现 `general`，预设 id/createdAt/updatedAt | 一 |
| 基础字段（文本/数字/布尔/日期） | Interface 与存储类型可配置，默认映射见 `02`/`05` | 一～二 |
| 选择 / 媒体字段 | 作为 Interface，存储为 string/json/text | 二 |
| Joi 必填/长度/范围/正则/邮箱/URL | 内置规则编译为 CEL | 一～二 |
| Joi 精度、关系必填限制 | CEL 表达式或字段 option | 二～三 |
| Filter 通用/比较/逻辑/LIKE | 实现，见 `05` | 一 |
| Filter 字符串增强、空值、between | 实现或紧随其后 | 一～二 |
| Filter `$col` / `$exists` / `$date*` / 数组算子 | **不做**，需要时再开 | — |
| `find/create/update/destroy` | HTTP CRUD，见 `05` 对照表 | 一 |
| `fields` / `except` / `sort` / 分页 | Query 参数 | 一 |
| `appends`、嵌套 `posts.title`、关联写入 | 第三阶段专项 | 三 |
| belongsTo / hasOne / hasMany / belongsToMany | 外键、中间表、级联策略见第三阶段 | 三 |
| 多对多（数组） | 第五阶段或与关系一并评估 | 五 |
| 树 / 日历 / 评论 / 文件表 | 仅先占类型枚举与 options，行为第五阶段 | 五 |
| 公式 / 加密 / 几何 / 排序 / 序列 | 第五阶段；公式用 CEL，加密密钥走配置 | 五 |
| Yaegi 钩子与自定义 API | 第四阶段 | 四 |
| 视图 / SQL 表 / 继承 / 外部数据源 | 不做 | — |

## GoFrame 使用边界

本模块**使用 GoFrame v2，但不把业务动态表交给 `gdb` / `gf gen dao`**。

| 能力 | 用法 | 不这样用 |
|------|------|----------|
| HTTP / 中间件 / 配置 | `ghttp`、`g.Cfg()`、`gcmd`、`MiddlewareHandlerResponse` | 手写 `WriteHeader` + `WriteJson` 作为主路径 |
| 静态 Meta API | `api/*/v1` + `g.Meta` + `(ctx, req) (res, err)` | 为每张业务表生成 `g.Meta` |
| 系统表 `collections` / `fields` / `indexes` / `yaegi_scripts` | `gf gen dao` 的 `dao` / `do` / `entity`，禁止手改 | 用 `g.Map` 写系统表 |
| 业务 Collection 行数据 | `database/sql` + 动态 DDL + Repository | `gf gen dao`、把动态行当成 DO |
| 请求参数校验 | GoFrame `v:` 标签 | 用 CEL 校验 page/collectionName |
| 记录校验 | 元数据规则 + CEL | 用编译期 struct tag 描述运行时字段 |

分层约定：业务写在 `internal/service/`，**不使用 `logic/`**。详见 [06](./06-项目结构设计.md)。

## 技术选型

| 组件 | 技术 | 版本 | 说明 |
|------|------|------|------|
| Web框架 | GoFrame | v2 | HTTP、配置、OpenAPI、系统表 DAO；**不是**业务表 ORM |
| 数据库 | MySQL 8 | Docker | 本机 Docker，账号 root / 123456，库名 itcodex |
| 系统表访问 | gdb + dao/do | v2 | 仅固定系统表；`created_at`/`updated_at` 由 ORM 维护 |
| 业务表访问 | `database/sql` + MySQL 驱动 | - | 运行时建表，无法 codegen |
| 校验引擎 | cel-go | latest | 业务记录的单字段 / 多字段联合校验 |
| 脚本引擎 | Yaegi | latest | 运行时二开 |
| 主键策略 | Snowflake（默认）/ UUID / NanoID | - | 业务表默认 Snowflake 写入 `BIGINT` |

## 核心特性

1. **动态数据表**：运行时创建/修改/删除普通表和字段，无需重启
2. **字段类型**：目标覆盖 8 大类 Interface；**当前按阶段交付**，不是一次做完 40+ 种
3. **CEL校验**：单字段/多字段联合校验，编译缓存
4. **Yaegi二开**：CRUD 钩子 + 自定义 API + 沙箱（第四阶段）
5. **类 NocoBase Filter**：兼容 `$` 操作符语法的**已声明子集**
6. **MySQL 存储**：Docker MySQL 8，持久化元数据与业务数据

## 当前实现 vs 目标

| 能力 | 当前代码 | 目标 |
|------|----------|------|
| MySQL 连接与系统表 sync | 已具备（`database/sql` + 差量 ADD COLUMN） | 系统表迁到 `g.DB()` + dao/do（后续） |
| 工程结构 | `service` + `g.Meta` + `internal/cmd` | 保持 |
| 普通表 CRUD + 基础 Filter | 已具备（含 `$empty/$notEmpty/$includes`） | 保持 |
| Collection / Field / Index 管理 API | 已具备 | 保持 |
| CEL 联合校验 | 已具备 | 保持 |
| 关系 / appends / 关联 HTTP | 已具备（belongsTo/hasOne/hasMany/belongsToMany） | 保持 |
| Yaegi | 已具备（钩子、自定义 API、沙箱导出、启动加载） | 保持 |
| 特殊表、公式、加密、几何、序列 | 已具备语义增强 | 对象存储等后续另议 |

## API Path 规范

| 接口类型 | Path 前缀 | 说明 |
|----------|-----------|------|
| 元数据管理 | `/api/meta/*` | Collection/Field/脚本管理 |
| 标准 CRUD | `/api/c/:collection/*` | 动态生成的数据增删改查 |
| 表级自定义API | `/api/custom/c/:collection/*` | Yaegi提供的单表自定义接口 |
| 全局自定义API | `/api/custom/global/*` | Yaegi提供的全局自定义接口 |

JS SDK 方法与 HTTP 的对应关系见 [05-API接口设计](./05-API接口设计.md#js-sdk-与-http-对照)。

## 快速开发指引

### 1. 第一阶段（核心可用）
- Database 管理器 + MySQL 连接
- 基础字段：string/text/integer/bigint/float/double/boolean/datetime
- 系统表自动创建和同步
- Repository 基础 CRUD（find/findOne/create/update/destroy）
- Filter：`$eq/$ne/$gt/$gte/$lt/$lte/$in/$notIn/$and/$or/$like/$notLike/$isNull/$notNull/$between/$startsWith/$endsWith`
- 简单单字段校验（required/min/max/pattern）

### 2. 第二阶段（元数据管理）
- 选择 / 媒体等 Interface 字段
- CEL 单字段校验与需求规则模板
- Collection 和 Field 元数据管理 API
- `$empty/$notEmpty/$includes` 等查询增强

### 3. 第三阶段（高级查询和校验）
- CEL 多字段联合校验
- 关系字段（belongsTo/hasOne/hasMany/belongsToMany）
- `appends`、关联过滤、关联写入
- 索引管理

### 4. 第四阶段（Yaegi二开）
- Yaegi 解释器初始化和沙箱
- CRUD before/after 钩子执行
- 自定义 API 路由注册和执行
- 脚本管理 API（增删改查、热加载）

### 5. 第五阶段（增强）
- 公式 / 加密 / 几何 / 排序 / 序列
- 树表 / 日历表 / 评论表 / 文件表行为
- 性能优化、错误处理、测试覆盖
