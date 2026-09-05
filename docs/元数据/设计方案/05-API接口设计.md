# ITCodeX 元数据模块 - API 接口设计

> 版本: v1.2
> 日期: 2026-09-05
> 框架: GoFrame v2（`g.Meta` + `ghttp` + `MiddlewareHandlerResponse`）
>
> 本文是 **HTTP API**。[api/](../api/) 是 NocoBase JS SDK 参考，不在本服务中实现 JS 绑定。
> 路由与目录见 [06](./06-项目结构设计.md)。GoFrame 边界见 [README](./README.md#goframe-使用边界)。

## 1. API 总览

| 分类 | 前缀 | 说明 |
|------|------|------|
| 元数据管理 | `/api/meta/*` | Collection 和 Field 的增删改查 |
| 标准 CRUD | `/api/c/:collection/*` | 动态生成的数据表操作接口 |
| 自定义 API | `/api/custom/*` | Yaegi 脚本提供的自定义接口 |

### JS SDK 与 HTTP 对照

| JS Repository / Database | HTTP | 阶段 |
|--------------------------|------|------|
| `repository.find(opts)` | `GET /api/c/:collection` | 一 |
| `repository.findOne` / `filterByTk` | `GET /api/c/:collection/:id` | 一 |
| `repository.count` | `GET /api/c/:collection/count` | 一 |
| `repository.create` | `POST /api/c/:collection` | 一 |
| `repository.createMany` | `POST /api/c/:collection/batch` | 一 |
| `repository.update` + `filterByTk` | `PUT /api/c/:collection/:id` | 一 |
| `repository.update` + `filter` | `PUT /api/c/:collection?filter=` | 一 |
| `repository.destroy` | `DELETE /api/c/:collection/:id` | 一 |
| `fields` / `except` / `sort` / `page` | Query 同名参数 | 一 |
| `appends`、关联过滤 `posts.title` | 同左；实现见第三阶段 | 三 |
| 关联 add/set/remove | `/api/c/:collection/:id/:association` | 三 |
| `db.registerOperators` | 进程内 `RegisterOperator`，不单独暴露 HTTP | 一 |
| `db.addMigration` / `sync` 选项全集 | 启动 Bootstrap + 建表/加列，无迁移框架 | 一 |

### GoFrame 落地方式

| 接口 | 定义 | 路由 | 参数校验 |
|------|------|------|----------|
| `/api/meta/*` | `api/metadata/v1` 的 Req/Res，带 `g.Meta` | `group.Bind(controller)` | `v:` 标签（page、name 等） |
| `/api/c/:collection/*` | 不按表生成 `g.Meta` | `/c` 组显式注册 | Query/JSON 解析；**记录**由 CEL + 元数据校验 |
| `/api/custom/*` | Yaegi 脚本 | 通配路由 | 由脚本处理 |

Controller 签名：`(ctx context.Context, req *v1.XxxReq) (res *v1.XxxRes, err error)`。

统一响应由 `ghttp.MiddlewareHandlerResponse` 输出，对应 `ghttp.DefaultHandlerResponse`（`code` / `message` / `data`）。业务错误用 `gerror`；领域码（404/409/403/422）用 `gcode` 映射。不要在 Controller 里手写 `WriteHeader` + `WriteJson` 作为主路径。

OpenAPI / Swagger：`server.openapiPath`、`server.swaggerPath`，由 `g.Meta` 生成。

所有接口统一返回格式：

```json
// 成功响应
{
    "code": 0,
    "message": "success",
    "data": { ... }
}

// 错误响应
{
    "code": 错误码,
    "message": "错误描述",
    "data": null
}

// 校验错误响应 (code=422)
{
    "code": 422,
    "message": "数据校验失败",
    "data": {
        "fieldErrors": {
            "字段名": ["错误信息1", "错误信息2"]
        },
        "tableErrors": ["全局错误信息"]
    }
}

// 分页列表响应
{
    "code": 0,
    "message": "success",
    "data": {
        "list": [ ... ],
        "total": 100,
        "page": 1,
        "pageSize": 20,
        "totalPages": 5
    }
}
```

## 2. 元数据管理接口 (/api/meta)

### 2.1 Collection 管理

#### 2.1.1 获取 Collection 列表
```
GET /api/meta/collections
```
**Query 参数:**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 否 | 按分类筛选 |
| keyword | string | 否 | 关键词搜索 |
| page | int | 否 | 页码，默认1 |
| pageSize | int | 否 | 每页条数，默认20 |

**响应示例:**
```json
{
    "code": 0,
    "data": {
        "list": [
            {
                "id": 1,
                "name": "users",
                "displayName": "用户",
                "type": "general",
                "description": "系统用户表",
                "categories": ["系统"],
                "createdAt": "2026-09-01T10:00:00Z",
                "fieldCount": 8
            }
        ],
        "total": 1
    }
}
```

#### 2.1.2 获取单个 Collection 详情
```
GET /api/meta/collections/:name
```
**Path 参数:**
- `name`: Collection 标识名

**响应示例:**
```json
{
    "code": 0,
    "data": {
        "id": 1,
        "name": "users",
        "displayName": "用户",
        "tableName": "users",
        "type": "general",
        "description": "系统用户表",
        "categories": ["系统"],
        "options": {
            "simplePagination": false
        },
        "filterTargetKey": "id",
        "autoGenId": true,
        "fields": [ ... ],
        "indexes": [ ... ],
        "createdAt": "2026-09-01T10:00:00Z",
        "updatedAt": "2026-09-02T15:30:00Z"
    }
}
```

#### 2.1.3 创建 Collection
```
POST /api/meta/collections
```
**请求体:**
```json
{
    "name": "orders",
    "displayName": "订单",
    "type": "general",
    "description": "订单数据表",
    "categories": ["销售"],
    "options": {
        "simplePagination": false
    },
    "presetFields": ["id", "createdAt", "createdBy", "updatedAt", "updatedBy"],
    "fields": [
        {
            "name": "orderNo",
            "type": "string",
            "displayName": "订单号",
            "isRequired": true,
            "isUnique": true,
            "options": {
                "maxLength": 50
            }
        },
        {
            "name": "amount",
            "type": "number",
            "displayName": "订单金额",
            "dataType": "decimal",
            "isRequired": true,
            "validation": {
                "min": 0.01
            }
        },
        {
            "name": "status",
            "type": "select",
            "displayName": "订单状态",
            "options": {
                "enum": ["pending", "paid", "shipped", "completed", "cancelled"],
                "default": "pending"
            }
        }
    ],
    "indexes": [
        {
            "fields": ["orderNo"],
            "unique": true
        },
        {
            "fields": ["status", "createdAt"]
        }
    ]
}
```

#### 2.1.4 更新 Collection
```
PUT /api/meta/collections/:name
```
**说明:** `name` 创建后不可修改。

#### 2.1.5 删除 Collection
```
DELETE /api/meta/collections/:name
```
**Query 参数:**
- `cascade`: boolean，是否级联删除依赖对象，默认 false

#### 2.1.6 同步 Collection 结构到数据库
```
POST /api/meta/collections/:name/sync
```

### 2.2 Field 管理

#### 2.2.1 获取字段列表
```
GET /api/meta/collections/:collection/fields
```

#### 2.2.2 添加字段
```
POST /api/meta/collections/:collection/fields
```
**请求体:**
```json
{
    "name": "phone",
    "type": "phone",
    "displayName": "手机号",
    "isRequired": false,
    "isUnique": true,
    "defaultValue": null,
    "description": "联系手机号",
    "options": {
        "format": "phone"
    },
    "validation": {
        "pattern": "^1[3-9]\\d{9}$"
    },
    "validationRules": [
        {
            "name": "phoneFormat",
            "expression": "phone == null || phone.phone()",
            "errorMessage": "请输入有效的手机号"
        }
    ]
}
```

#### 2.2.3 更新字段
```
PUT /api/meta/collections/:collection/fields/:field
```

#### 2.2.4 删除字段
```
DELETE /api/meta/collections/:collection/fields/:field
```

### 2.3 Yaegi 脚本管理

#### 2.3.1 获取脚本列表
```
GET /api/meta/scripts?collection=xxx&hook=xxx
```

#### 2.3.2 创建/更新脚本
```
POST /api/meta/scripts
PUT /api/meta/scripts/:id
```
**请求体（钩子脚本）:**
```json
{
    "collectionName": "orders",
    "name": "自动生成订单号",
    "hookPoint": "beforeCreate",
    "content": "package main\n\nimport (\n    \"context\"\n    \"fmt\"\n    \"itcodex/utils\"\n)\n\nfunc BeforeCreate(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {\n    now := utils.Now()\n    datePart := utils.FormatTime(now, \"20060102\")\n    seq := utils.NanoID(6)\n    data[\"orderNo\"] = fmt.Sprintf(\"ORD-%s-%s\", datePart, seq)\n    return data, nil\n}",
    "priority": 0,
    "enabled": true
}
```

**请求体（自定义API脚本）:**
```json
{
    "collectionName": "users",
    "name": "修改密码",
    "hookPoint": "customAPI",
    "apiPath": "/c/users/:id/change-password",
    "httpMethod": "POST",
    "content": "package main\n\nimport (\n    \"itcodex/context\"\n)\n\nfunc Handle(ctx *context.YaegiHTTPContext) {\n    // ... 脚本逻辑\n}",
    "enabled": true
}
```

#### 2.3.3 验证脚本语法
```
POST /api/meta/scripts/validate
```
**请求体:** `{ "content": "..." }`

#### 2.3.4 启用/禁用脚本
```
PUT /api/meta/scripts/:id/toggle
```

#### 2.3.5 删除脚本
```
DELETE /api/meta/scripts/:id
```

## 3. 标准 CRUD 接口 (/api/c/:collection)

所有动态生成的接口都会经过：
1. 元数据解析（根据 collection 名获取字段定义）
2. Yaegi before 钩子执行
3. CEL 校验（单字段 + 多字段）
4. 数据库操作
5. Yaegi after 钩子执行

### 3.1 查询列表
```
GET /api/c/:collection
```
**Query 参数:**

| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| filter | JSON | 过滤条件（使用NocoBase风格操作符） | `{"status":"active","age":{"$gt":18}}` |
| sort | string | 排序，-前缀表示降序 | `-createdAt,name` |
| fields | string | 指定返回字段，逗号分隔 | `id,name,email` |
| except | string | 排除字段，逗号分隔 | `password` |
| appends | string | 追加关联字段 | `profile,posts` |
| page | int | 页码 | 1 |
| pageSize | int | 每页条数 | 20 |
| limit | int | 限制数量（不分页） | 100 |
| offset | int | 偏移量 | 0 |

**Filter 操作符示例:**
```json
// 简单条件
{
    "status": "active",
    "age": { "$gt": 18, "$lt": 60 }
}

// AND 条件
{
    "$and": [
        { "status": "active" },
        { "age": { "$gte": 18 } }
    ]
}

// OR 条件
{
    "$or": [
        { "role": "admin" },
        { "department": "IT" }
    ]
}

// 字符串模糊匹配
{
    "name": { "$like": "%张%" }
}

// IN 查询
{
    "id": { "$in": [1, 2, 3] }
}

// 关联字段过滤（第三阶段）
{
    "posts.title": { "$like": "%公告%" }
}
```

### Filter 操作符分期

一期实现（与当前代码对齐）：

`$eq` `$ne` `$gt` `$gte` `$lt` `$lte` `$in` `$notIn` `$like` `$notLike` `$isNull` `$notNull` `$between` `$notBetween` `$startsWith` `$endsWith`，以及逻辑 `$and` `$or` `$not`。

二期：`$empty` `$notEmpty` `$includes` `$notIncludes`。

**不做**（除非单独立项）：`$col` `$is`/`$not`（列比较语义）`$exists` `$dateOn` 等日期族、`$match`/`$anyOf` 等数组族、`$iLike`/`$regexp`。

`appends` 与 `posts.title` 关联过滤：第三阶段；接口参数先保留，未实现时忽略或返回明确错误。

**响应示例:**
```json
{
    "code": 0,
    "data": {
        "list": [
            {
                "id": 1,
                "name": "张三",
                "email": "zhangsan@example.com",
                "age": 28,
                "createdAt": "2026-09-01T10:00:00Z"
            }
        ],
        "total": 100,
        "page": 1,
        "pageSize": 20,
        "totalPages": 5
    }
}
```

### 3.2 查询单条
```
GET /api/c/:collection/:id
```
**Query 参数:** `fields`, `except`, `appends`

### 3.3 创建记录
```
POST /api/c/:collection
```
**请求体:**
```json
{
    "name": "李四",
    "email": "lisi@example.com",
    "age": 25,
    "profile": {
        "bio": "这是简介",
        "avatar": "https://..."
    }
}
```

### 3.4 批量创建
```
POST /api/c/:collection/batch
```
**请求体:**
```json
[
    { "name": "王五", "email": "wangwu@example.com" },
    { "name": "赵六", "email": "zhaoliu@example.com" }
]
```

### 3.5 更新记录
```
PUT /api/c/:collection/:id
```
**Query 参数:**
- `whitelist`: 只更新指定字段，逗号分隔
- `blacklist`: 排除指定字段，逗号分隔

**请求体:**
```json
{
    "age": 26,
    "profile": {
        "bio": "更新后的简介"
    }
}
```

### 3.6 批量更新
```
PUT /api/c/:collection
```
**Query 参数:**
- `filter`: 过滤条件，JSON
- `whitelist`, `blacklist`

### 3.7 删除记录
```
DELETE /api/c/:collection/:id
```

### 3.8 批量删除
```
DELETE /api/c/:collection?filter={...}
```
**注意:** 不带 filter 时视为 truncate。本模块无独立权限体系，由网关/调用方约束；服务端可加配置开关禁止无条件清空。

### 3.9 关联操作（第三阶段）
```
// 查询关联数据
GET /api/c/:collection/:id/:association

// 添加关联
POST /api/c/:collection/:id/:association
Body: { "id": 123 } 或 [ { "id": 1 }, { "id": 2 } ]

// 更新关联（替换）
PUT /api/c/:collection/:id/:association
Body: [ { "id": 1 }, { "id": 3 } ]

// 移除关联
DELETE /api/c/:collection/:id/:association
Body: { "id": 123 }
```

### 3.10 统计数量
```
GET /api/c/:collection/count?filter={...}
```
**响应:**
```json
{
    "code": 0,
    "data": {
        "count": 42
    }
}
```

## 4. 自定义 API 路由 (/api/custom)

由 Yaegi 脚本动态注册，示例：

### 4.1 表级自定义 API
```
POST /api/custom/c/orders/generate-order-no
POST /api/custom/c/users/:id/change-password
GET  /api/custom/c/products/:id/stock-history
```

### 4.2 全局自定义 API
```
POST /api/custom/global/data-import
GET  /api/custom/global/health
POST /api/custom/global/export
```

自定义 API 的请求和响应完全由 Yaegi 脚本控制，但建议遵循统一的 JSON 响应格式。

## 5. 字段类型与表单映射参考

下表是 **目标目录**（对齐需求 Interface），按 [设计方案总览](./README.md#需求--api--设计决策--阶段) 分期实现，不是当前已全部落地。

| Interface类型 | 数据类型 | 表单组件 | 说明 |
|--------------|----------|----------|------|
| input | string | 单行输入框 | 单行文本 |
| textarea | text | 多行输入框 | 多行文本 |
| phone | string | 手机号输入 | 手机号 |
| email | string | 邮箱输入 | 邮箱 |
| url | string | URL输入 | URL |
| number | double/integer | 数字输入 | 数字/整数 |
| percent | double | 百分比输入 | 百分比 |
| color | string | 颜色选择器 | 颜色 |
| icon | string | 图标选择器 | 图标 |
| password | string | 密码输入 | 密码（哈希存储） |
| checkbox | boolean | 勾选框 | 勾选 |
| select | string/integer | 下拉单选 | 下拉单选 |
| radioGroup | string/integer | 单选框组 | 单选框组 |
| multipleSelect | json/array | 下拉多选 | 下拉多选 |
| checkboxGroup | json/array | 复选框组 | 复选框组 |
| chinaRegion | string/json | 行政区选择 | 中国行政区 |
| markdown | text | Markdown编辑器 | Markdown |
| richText | longtext | 富文本编辑器 | 富文本 |
| attachment | relation | 附件上传 | 附件（关联文件表） |
| attachmentUrl | string/json | URL附件 | 附件URL |
| datetime | datetime | 日期时间选择 | 日期时间无时区 |
| date | date | 日期选择 | 日期 |
| time | time | 时间选择 | 时间 |
| unixTimestamp | bigint | 日期时间选择 | Unix时间戳 |
| belongsTo | foreignKey | 数据选择器 | 多对一关系 |
| hasMany | - | 子表格/子表单 | 一对多关系 |
| hasOne | - | 子表单 | 一对一关系 |
| belongsToMany | through table | 数据选择器（多选） | 多对多关系 |
| point | json | 地图选点 | 点 |
| uuid | string/uuid | 隐藏/只读 | UUID |
| nanoId | string | 隐藏/只读 | NanoID |
| sort | integer | 排序组件 | 排序 |
| formula | virtual/double | 只读展示 | 公式 |
| sequence | string/integer | 只读展示 | 自动序列 |
| json | json/jsonb | JSON编辑器 | JSON |
| tableSelector | string | 表选择器 | 表选择器 |
| encrypted | text | 加密输入 | 加密字段 |
| createdAt | datetime | 只读 | 创建时间 |
| createdBy | bigint | 只读 | 创建人 |
| updatedAt | datetime | 只读 | 更新时间 |
| updatedBy | bigint | 只读 | 更新人 |
