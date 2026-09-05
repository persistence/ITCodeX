# ITCodeX 元数据模块 - 测试 Client 设计方案

> 版本: v2.0
> 日期: 2026-09-05
> 技术栈: Go + net/http + testing + testify
> 对齐规格: [docs/元数据/设计方案](../../docs/元数据/设计方案/)（五阶段能力闭环）

## 1. 概述

本测试 Client 对 ITCodeX 元数据服务做 **HTTP 端到端集成测试**，覆盖设计方案一～五阶段已落地的公开 API（Meta / CRUD / 关联 / Index / Yaegi / 特殊表语义）。

参考文档优先级：

1. [设计方案 README](../../docs/元数据/设计方案/README.md) — 范围与阶段
2. [05-API接口设计](../../docs/元数据/设计方案/05-API接口设计.md) — HTTP 契约
3. [08-测试方案](../../docs/元数据/设计方案/08-测试方案.md) — 分层策略（本 Client 对应其中的 E2E/集成层）
4. `docs/元数据/api/` — NocoBase JS SDK **语义参考**（本 Client 不实现 JS，只验证等价 HTTP 行为）

### 1.1 设计目标

- 覆盖服务端公开 HTTP API（按五阶段能力矩阵）
- 用例可独立运行、自动清理测试 Collection
- 错误码与校验失败可断言（404/409/422 等）
- 支持 Yaegi 钩子与自定义 API 冒烟
- 通过环境变量切换服务地址

### 1.2 服务端信息

| 项 | 值 |
|----|-----|
| 默认地址 | `http://localhost:8000`（`TEST_SERVER_URL` 可覆盖） |
| Meta | `/api/meta/*` |
| CRUD | `/api/c/:collection/*` |
| 自定义 API | `/api/custom/*` |
| 响应 | `{ code, message, data }`（GoFrame `DefaultHandlerResponse`） |

### 1.3 与服务端单元测试的分工

| 层级 | 位置 | 职责 |
|------|------|------|
| 单元/仓储 | `server/internal/service/metadata/*_test.go` | Filter/CEL/Repository/关系内存逻辑 |
| HTTP E2E | `client/internal/tests/` | 真实 HTTP + MySQL，验证契约与联通性 |

## 2. 项目结构

```
client/
├── cmd/test-client/main.go          # 可选：冒烟入口
├── internal/
│   ├── client/
│   │   ├── client.go                # HTTP 基础
│   │   ├── types.go                 # 请求/响应类型
│   │   ├── meta.go                  # Collection/Field/Index/Script/Sync
│   │   ├── crud.go                  # CRUD + count + batch
│   │   └── association.go           # 关联 appends / association API
│   └── tests/
│       ├── suite_test.go            # 套件与清理
│       ├── meta_test.go             # 元数据管理（一/二期）
│       ├── crud_test.go             # 基础 CRUD（一期）
│       ├── filter_test.go           # Filter/Sort/Fields（一/二期）
│       ├── validation_test.go       # 校验（一/二期）
│       ├── relation_test.go         # 关系/appends/关联 HTTP（三期）
│       ├── index_test.go            # 索引 Meta API（三期）
│       ├── script_test.go           # Yaegi 脚本（四期）
│       └── enhanced_test.go         # 公式/加密/序列/特殊表（五期）
├── docs/测试Client设计方案.md       # 本文档
├── go.mod
└── go.sum
```

## 3. 能力矩阵（按设计方案五阶段）

| 阶段 | 能力 | Client 封装 | 测试文件 |
|------|------|-------------|----------|
| 一 | Collection/Field CRUD、记录 CRUD、基础 Filter、分页排序 | `meta` / `crud` | `meta` / `crud` / `filter` / `validation` |
| 二 | 选择/媒体字段、`$empty/$notEmpty/$includes`、CEL 规则 | `meta` / `crud` | `filter` / `validation` / `enhanced` |
| 三 | 关系字段、`appends`、关联过滤、关联 HTTP、Index API | `association` / `meta` | `relation` / `index` |
| 四 | Script list/save/disable/delete/validate、钩子、自定义 API | `meta` | `script` |
| 五 | formula/encrypted/sequence/uuid、tree/calendar/comment/file | `meta` / `crud` | `enhanced` |

**明确不做（与设计 README 一致）**：页面/权限/工作流、对象存储上传流、`$col`/`$exists`/日期族算子、dao/do 迁移相关断言。

## 4. 核心模块设计

### 4.1 基础 HTTP 客户端 (`client.go`)

```go
type Client struct {
    BaseURL    string
    HTTPClient *http.Client
}
```

方法：`Get` / `Post` / `Put` / `Delete`（内部 `doRequest`）。

约定：

- `code == 0`（或兼容 200/201）为成功，否则返回 `*APIError`
- HTTP 状态 ≥ 400 同样映射为 `*APIError`
- JSON number 归一化为 `int64`/`float64`，便于断言

### 4.2 元数据管理 (`meta.go`)

| 接口 | 方法 | 说明 |
|------|------|------|
| `GET /api/meta/collections` | ListCollections | data 为 `{list,total}` |
| `POST /api/meta/collections` | CreateCollection | 支持 general/tree/calendar/comment/file |
| `GET/PUT/DELETE .../collections/:name` | Get/Update/Drop | |
| `POST .../collections/:name/sync` | SyncCollection | 差量 ADD COLUMN |
| `GET/POST/PUT/DELETE .../fields` | List/Add/Update/RemoveField | |
| `GET/POST/DELETE .../indexes` | List/Create/DeleteIndex | 三期 |
| `GET/POST /api/meta/scripts` | List/SaveScript | |
| `POST .../scripts/:id/disable` | DisableScript | |
| `DELETE .../scripts/:id` | DeleteScript | |
| `POST .../scripts/validate` | ValidateScript | |

`CreateFieldInput` 需支持关系与增强字段选项：`target` / `foreignKey` / `through` / `expression` / `pattern` 等（对齐服务端 `CreateFieldInput`）。

### 4.3 数据 CRUD (`crud.go`)

| 接口 | 方法 |
|------|------|
| `GET /api/c/:collection` | List（filter/sort/fields/except/**appends**/page） |
| `GET /api/c/:collection/:id` | FindOne（fields/except/**appends**） |
| `POST /api/c/:collection` | Create |
| `POST /api/c/:collection/batch` | CreateMany |
| `PUT /api/c/:collection/:id` | Update |
| `PUT /api/c/:collection?filter=` | UpdateMany |
| `DELETE /api/c/:collection/:id` | DeleteOne |
| `DELETE /api/c/:collection?filter=` | BulkDelete |
| `GET /api/c/:collection/count` | Count（**不要**用 List.Total 凑数） |

`FindOptions` / `FindOneOptions` 增加 `Appends []string`。

### 4.4 关联 API (`association.go`)

| 接口 | 方法 |
|------|------|
| `GET /api/c/:c/:id/:association` | ListAssociation |
| `POST ...` | AddAssociation |
| `PUT ...` | SetAssociation |
| `DELETE ...` | RemoveAssociation |

Body 支持 `{ "id": 1 }`、`[{ "id": 1 }, { "id": 2 }]`、或纯 ID 数组（与设计 05 §3.9 一致）。

### 4.5 类型要点 (`types.go`)

```go
type CreateFieldInput struct {
    Name, DisplayName, Type string
    IsRequired, IsUnique, IsIndexed bool
    Options map[string]interface{}
    // 关系 / 增强
    Target, ForeignKey, SourceKey, Through, OtherKey, TargetKey string
    Expression, Pattern string
    AutoGenerate bool
    StartsAt, IncrementBy int
}

type FindOptions struct {
    Filter Filter
    Sort, Fields, Except, Appends []string
    Page, PageSize int
}

type Index struct {
    Name   string   `json:"name,omitempty"`
    Fields []string `json:"fields"`
    Unique bool     `json:"unique,omitempty"`
}
```

Meta 列表类接口统一从 `data.list` 取值，避免把 `{list,total}` 直接反序列化成切片。

## 5. 测试用例设计

### 5.1 套件 (`suite_test.go`)

- `setupTest`：读 `TEST_SERVER_URL`，必要时 `t.Skip`（服务不可达）
- `createTestCollection` / `createSpecialCollection(type)` + `t.Cleanup(DropCollection)`
- `createTestRecord`、`isAPIError`、`asFloat64` 等助手

### 5.2 一期/二期（保留并增强）

**meta_test.go**：Collection/Field CRUD、重名冲突、NotFound。

**crud_test.go**：单条/批量 CRUD、必填失败、分页、Count。

**filter_test.go**：

| 用例 | 说明 |
|------|------|
| `$eq/$ne/$gt/$gte/$lt/$lte/$in/$like` | 一期 |
| `$and/$or`、`$startsWith/$endsWith` | 一期 |
| `$empty/$notEmpty/$includes/$notIncludes` | **二期新增** |
| Sort / Fields / Except | 一期 |

**validation_test.go**：required / unique / length / range / pattern / 多字段错误。

### 5.3 三期 (`relation_test.go` / `index_test.go`)

| 用例 | 说明 |
|------|------|
| TestBelongsTo_Appends | 创建作者/文章，List/Get 带 `appends=author_id` |
| TestHasMany_SetAssociation | SetAssociation 后 appends 可见子记录 |
| TestBelongsToMany_Through | 中间表关联 add/set/remove |
| TestFilter_AssociationPath | `filter={"posts.title":{"$eq":"..."}}` |
| TestNestedCreate_Association | create values 嵌套 `{id}` / 解除 `null` |
| TestIndex_CreateListDelete | Meta Index API 闭环 |

### 5.4 四期 (`script_test.go`)

| 用例 | 说明 |
|------|------|
| TestScript_SaveValidateDisableDelete | 脚本生命周期 |
| TestScript_BeforeCreateHook | 保存钩子后 Create 观察副作用字段 |
| TestScript_CustomAPI | 注册 customAPI，HTTP 调 `/api/custom/...` |

### 5.5 五期 (`enhanced_test.go`)

| 用例 | 说明 |
|------|------|
| TestField_SequenceUUIDEncrypted | 自动序列 / UUID / 加密读写 |
| TestField_FormulaGeo | formula 写前计算、point GeoJSON |
| TestCollection_TreeChildren | type=tree，`appends=children` |
| TestCollection_FileDefaults | type=file 具备 name/url/mime/size |
| TestCollection_CalendarComment | 默认字段存在即可（语义冒烟） |

## 6. 测试执行

### 6.1 前置条件

1. Docker MySQL 8 可用（root/123456，库 `itcodex`）
2. 元数据服务已启动：`cd server && go run .`（监听 `:8000`）
3. Client 可访问服务

### 6.2 命令

```bash
# 全量 E2E
cd client
go test ./internal/tests/... -count=1 -v

# 按阶段
go test ./internal/tests/ -run "TestFilter_Empty|TestBelongsTo|TestIndex|TestScript|TestField_|TestCollection_Tree" -v

# 自定义地址
set TEST_SERVER_URL=http://127.0.0.1:8000
go test ./internal/tests/... -count=1
```

### 6.3 执行顺序建议

1. Meta → CRUD → Filter/Validation  
2. Relation / Index  
3. Script  
4. Enhanced / 特殊表  

用例彼此独立；清理靠 `t.Cleanup`。

## 7. 错误处理

| code / HTTP | 含义 | 断言 |
|-------------|------|------|
| 0 | 成功 | 解析 `data` |
| 404 | 不存在 | `*APIError` |
| 409 | 冲突 | 重名 Collection/Field |
| 422 | 校验失败 | `data.fieldErrors` / `tableErrors` |
| 403 | 禁止无条件清空等 | BulkDelete 无 filter |

## 8. 使用示例

```go
func TestExample_Relation(t *testing.T) {
    s := setupTest(t)
    s.createTestCollection(t, "authors_e2e",
        client.CreateFieldInput{Name: "name", Type: "string", IsRequired: true},
    )
    s.createTestCollection(t, "posts_e2e",
        client.CreateFieldInput{Name: "title", Type: "string", IsRequired: true},
        client.CreateFieldInput{Name: "author_id", Type: "belongsTo", Target: "authors_e2e", ForeignKey: "author_id"},
    )
    author := s.createTestRecord(t, "authors_e2e", map[string]interface{}{"name": "Alice"})
    post := s.createTestRecord(t, "posts_e2e", map[string]interface{}{
        "title": "Hello", "author_id": author["id"],
    })
    got, err := s.client.FindOne(s.ctx, "posts_e2e", fmt.Sprint(post["id"]), &client.FindOneOptions{
        Appends: []string{"author_id"},
    })
    require.NoError(t, err)
    _ = got
}
```

## 9. 后续扩展

1. 并发冒烟（同表多 goroutine Create）
2. Benchmark（List + Filter）
3. HTML/JSON 测试报告
4. 与 CI 联动：先起 Docker MySQL + server，再跑 `client` 测试

## 10. 注意事项

1. 仅连测试环境；会真实改库
2. Collection 名建议带随机后缀或固定前缀 `test_`，避免污染
3. 关联测试注意创建顺序（目标表先于关系表）
4. Yaegi 钩子测试若脚本编译失败，断言错误信息而非静默跳过
5. 本文档随 [设计方案 README](../../docs/元数据/设计方案/README.md)「当前实现 vs 目标」同步维护
