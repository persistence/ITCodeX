# ITCodeX 元数据模块 - Yaegi 二次开发设计

> 版本: v1.2
> 日期: 2026-09-05
> 技术: Yaegi (Go 语言解释器)
> 钩子与自定义 API 中间件放在 `internal/service/middleware`，由 `internal/cmd` 注册。自定义 API 保持通配路由，不为脚本生成 `g.Meta`。

## 1. 设计目标

- 支持通过 Yaegi 在运行时执行 Go 脚本，无需重新编译主程序
- CRUD 接口支持生命周期钩子（before/after create/update/delete 等）
- 每个元数据表支持自定义 API 接口（自定义 path）
- 提供安全的沙箱环境，导出必要的 API 给脚本使用
- 脚本热加载，修改后无需重启即可生效
- 统一的 path 规范前缀

## 2. 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP 请求层                                │
├─────────────────────────────────────────────────────────────┤
│                    路由分发                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ 标准CRUD路由 │  │ 元数据管理   │  │ /api/custom/*       │  │
│  │ /api/collec │  │ /api/meta/* │  │ Yaegi自定义API路由   │  │
│  └──────┬──────┘  └─────────────┘  └──────────┬──────────┘  │
└─────────┼──────────────────────────────────────┼─────────────┘
          │                                      │
          ▼                                      ▼
┌─────────────────────────────────────────────────────────────┐
│                  Yaegi 脚本引擎                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    钩子执行器                           │  │
│  │  beforeCreate ──► create ──► afterCreate              │  │
│  │  beforeUpdate ──► update ──► afterUpdate              │  │
│  │  beforeDelete ──► delete ──► afterDelete              │  │
│  │  beforeValidate / afterValidate                       │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  自定义API处理器                        │  │
│  │  注册到 /api/custom/:collectionName/*                 │  │
│  │  脚本处理 HTTP 请求/响应                               │  │
│  └───────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                    Yaegi 沙箱环境                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ DB 访问API  │  │  工具函数库  │  │   HTTP上下文对象     │  │
│  │ (Repository)│  │ (strings/   │  │  (Request/Response) │  │
│  │  安全封装    │  │  json/time) │  │                     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                  脚本管理和缓存                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  脚本加载器  │  │  编译缓存    │  │   热更新监听        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 3. Path 规范

### 3.1 API Path 前缀规范

| 接口类型 | Path 前缀 | 说明 |
|----------|-----------|------|
| 标准 CRUD | `/api/c/:collectionName/*` | 自动生成的增删改查接口 |
| 元数据管理 | `/api/meta/*` | Collection/Field 管理接口 |
| 自定义 API（全局） | `/api/custom/global/*` | 全局 Yaegi 自定义接口 |
| 自定义 API（表级） | `/api/custom/c/:collectionName/*` | 单表 Yaegi 自定义接口 |

### 3.2 标准 CRUD Path 详细设计

```
# 查询列表（GET）
GET    /api/c/:collection
GET    /api/c/:collection?filter={...}&sort=-id&page=1&pageSize=20

# 查询单条（GET）
GET    /api/c/:collection/:id

# 创建（POST）
POST   /api/c/:collection
Body:  { "field1": "value1", ... }

# 批量创建（POST）
POST   /api/c/:collection/batch
Body:  [ { ... }, { ... } ]

# 更新（PUT/PATCH）
PUT    /api/c/:collection/:id
Body:  { "field1": "newValue" }

# 批量更新（PUT）
PUT    /api/c/:collection
Query: filter={...}
Body:  { "field1": "newValue" }

# 删除（DELETE）
DELETE /api/c/:collection/:id

# 批量删除（DELETE）
DELETE /api/c/:collection
Query: filter={...}

# 关联查询
GET    /api/c/:collection/:id/:association
POST   /api/c/:collection/:id/:association
PUT    /api/c/:collection/:id/:association
DELETE /api/c/:collection/:id/:association
```

### 3.3 自定义 API Path 示例

```
# 订单表的自定义接口：生成订单号
POST /api/custom/c/orders/generate-order-no

# 用户表的自定义接口：修改密码
POST /api/custom/c/users/:id/change-password

# 全局自定义接口：数据导入
POST /api/custom/global/data-import
```

## 4. 导出给 Yaegi 的 API（沙箱环境）

### 4.1 核心 API 包

为 Yaegi 脚本提供以下导出包，脚本可以直接 `import` 使用：

```go
// 导出包名: itcodex/metadata
package metadata

// DB 全局数据库对象（脚本入口注入）
var DB *YaegiDB

// YaegiDB 封装给脚本使用的数据库API
type YaegiDB struct {
    // 获取指定表的Repository
    func Collection(name string) *YaegiRepository
}

// YaegiRepository 数据仓库API（安全封装）
type YaegiRepository struct {
    // 查询
    func Find(filter map[string]interface{}, opts ...FindOption) ([]map[string]interface{}, error)
    func FindOne(filter map[string]interface{}) (map[string]interface{}, error)
    func FindByID(id interface{}) (map[string]interface{}, error)
    func Count(filter map[string]interface{}) (int, error)
    func FindAndCount(filter map[string]interface{}, opts ...FindOption) ([]map[string]interface{}, int, error)

    // 写入
    func Create(values map[string]interface{}) (map[string]interface{}, error)
    func CreateMany(records []map[string]interface{}) ([]map[string]interface{}, error)
    func Update(filter map[string]interface{}, values map[string]interface{}) (int, error)
    func UpdateByID(id interface{}, values map[string]interface{}) (map[string]interface{}, error)
    func Delete(filter map[string]interface{}) (int, error)
    func DeleteByID(id interface{}) error

    // 事务
    func Transaction(fn func(tx *YaegiTX) error) error
}

// YaegiTX 事务对象
type YaegiTX struct {
    *YaegiRepository
    func Commit() error
    func Rollback() error
}

// FindOption 查询选项
type FindOption func(*findOptions)
func WithFields(fields ...string) FindOption
func WithSort(sort ...string) FindOption
func WithLimit(limit int) FindOption
func WithOffset(offset int) FindOption
func WithPage(page, pageSize int) FindOption
func WithAppends(associations ...string) FindOption
```

```go
// 导出包名: itcodex/context
package context

// YaegiHTTPContext HTTP上下文对象（自定义API中使用）
type YaegiHTTPContext struct {
    // 请求
    Request *YaegiRequest
    Response *YaegiResponse

    // 路径参数
    Params map[string]string
    // 查询参数
    Query map[string]string
    // 请求头
    Headers map[string]string

    // 用户信息（认证后注入）
    User *YaegiUser

    // 原始请求/响应对象（高级用法）
    RawRequest  *ghttp.Request
}

type YaegiRequest struct {
    func Body() []byte
    func BindJSON(v interface{}) error
    func GetJSON(key string) interface{}
    func GetString(key string, def ...string) string
    func GetInt(key string, def ...int) int
    func GetForm(key string) string
    func GetCookie(key string) string
}

type YaegiResponse struct {
    func JSON(status int, data interface{})
    func JSONSuccess(data interface{})
    func JSONError(status int, msg string)
    func JSONValidationError(errors map[string][]string)
    func String(status int, text string)
    func Redirect(url string, status ...int)
    func SetHeader(key, value string)
}

type YaegiUser struct {
    ID   int64
    Name string
    // 其他用户信息...
}
```

```go
// 导出包名: itcodex/utils
package utils

// 字符串工具
func Contains(s, substr string) bool
func HasPrefix(s, prefix string) bool
func HasSuffix(s, suffix string) bool
func Trim(s string) string
func TrimSpace(s string) string
func ToLower(s string) string
func ToUpper(s string) string
func Split(s, sep string) []string
func Join(elems []string, sep string) string
func Replace(s, old, new string, n int) string

// JSON工具
func ToJSON(v interface{}) (string, error)
func FromJSON(s string, v interface{}) error
func ToMap(v interface{}) map[string]interface{}

// 时间工具
func Now() time.Time
func ParseTime(s string, layout ...string) (time.Time, error)
func FormatTime(t time.Time, layout ...string) string

// ID生成
func SnowflakeID() int64
func UUID() string
func NanoID(len ...int) string

// 加密工具
func HashPassword(password string) (string, error)
func VerifyPassword(password, hash string) bool
func MD5(s string) string
func SHA256(s string) string

// 日志
func LogInfo(v ...interface{})
func LogWarn(v ...interface{})
func LogError(v ...interface{})
```

```go
// 导出包名: itcodex/validation
package validation

// CEL表达式校验（在脚本中可直接使用）
func ValidateCEL(data map[string]interface{}, expression string) (bool, error)
func NewValidationError(msg string) error
func NewFieldValidationError(field, msg string) error
```

### 4.2 钩子函数签名

Yaegi 脚本需要导出特定签名的函数作为钩子：

```go
// 钩子函数签名规范

// BeforeCreateHook 创建前钩子
// 返回 error 会终止创建并返回错误
// 返回 modified data 可修改要创建的数据
func BeforeCreate(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)

// AfterCreateHook 创建后钩子
func AfterCreate(ctx context.Context, result map[string]interface{}) error

// BeforeUpdateHook 更新前钩子
func BeforeUpdate(ctx context.Context, oldData, newData map[string]interface{}) (map[string]interface{}, error)

// AfterUpdateHook 更新后钩子
func AfterUpdate(ctx context.Context, result map[string]interface{}) error

// BeforeDeleteHook 删除前钩子
// 返回 error 会终止删除
func BeforeDelete(ctx context.Context, data map[string]interface{}) error

// AfterDeleteHook 删除后钩子
func AfterDelete(ctx context.Context, data map[string]interface{}) error

// BeforeValidateHook 校验前钩子
func BeforeValidate(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)

// AfterValidateHook 校验后钩子
func AfterValidate(ctx context.Context, data map[string]interface{}) error

// CustomAPIHandler 自定义API处理函数
func Handle(ctx *context.YaegiHTTPContext)
```

## 5. Yaegi 脚本管理

### 5.1 脚本存储结构

脚本可以两种方式存储：
1. **数据库存储**：存储在 `yaegi_scripts` 系统表中，支持通过 API 管理
2. **文件系统存储**：存储在 `scripts/` 目录下，支持文件热加载

```go
package yaegi

// Script Yaegi脚本定义
type Script struct {
    ID            int64             `json:"id"`
    CollectionName string           `json:"collectionName,omitempty"` // 关联的表，空表示全局
    Name          string            `json:"name"`
    HookPoint     HookPointType     `json:"hookPoint"`
    Content       string            `json:"content"`
    APIPath       string            `json:"apiPath,omitempty"`     // 自定义API路径
    HTTPMethod    string            `json:"httpMethod,omitempty"`  // GET/POST/PUT/DELETE
    Enabled       bool              `json:"enabled"`
    Priority      int               `json:"priority"` // 执行顺序，越小越先执行
    Options       map[string]interface{} `json:"options,omitempty"`
    CreatedAt     *gtime.Time       `json:"createdAt"`
    UpdatedAt     *gtime.Time       `json:"updatedAt"`

    // 编译后的符号（运行时）
    symbols map[string]reflect.Value `json:"-"`
}

type HookPointType string

const (
    HookBeforeCreate   HookPointType = "beforeCreate"
    HookAfterCreate    HookPointType = "afterCreate"
    HookBeforeUpdate   HookPointType = "beforeUpdate"
    HookAfterUpdate    HookPointType = "afterUpdate"
    HookBeforeDelete   HookPointType = "beforeDelete"
    HookAfterDelete    HookPointType = "afterDelete"
    HookBeforeValidate HookPointType = "beforeValidate"
    HookAfterValidate  HookPointType = "afterValidate"
    HookCustomAPI      HookPointType = "customAPI"
)
```

### 5.2 YaegiManager 实现

```go
package yaegi

import (
    "sync"

    "github.com/traefik/yaegi/interp"
    "github.com/traefik/yaegi/stdlib"
)

// YaegiManager Yaegi脚本管理器
type YaegiManager struct {
    mu sync.RWMutex

    // 解释器实例（每个脚本独立interpreter避免污染）
    interpreters map[int64]*interp.Interpreter

    // 脚本注册表: collectionName:hookPoint -> []*Script
    hooks map[string][]*Script

    // 自定义API注册表: method:path -> *Script
    customAPIs map[string]*Script

    // 导出给Yaegi的符号
    exportedSymbols interp.Exports

    // 是否启用文件热加载
    watchFiles bool
    scriptsDir string
}

// NewYaegiManager 创建Yaegi管理器
func NewYaegiManager(scriptsDir string) (*YaegiManager, error) {
    m := &YaegiManager{
        interpreters: make(map[int64]*interp.Interpreter),
        hooks:        make(map[string][]*Script),
        customAPIs:   make(map[string]*Script),
        scriptsDir:   scriptsDir,
    }

    // 注册标准库和自定义包
    m.exportedSymbols = m.buildExports()

    return m, nil
}

// buildExports 构建导出符号表
func (m *YaegiManager) buildExports() interp.Exports {
    exports := interp.Exports{}

    // 合并标准库
    for k, v := range stdlib.Symbols {
        exports[k] = v
    }

    // 导出自定义包: itcodex/metadata, itcodex/context, itcodex/utils
    // ... 注册自定义符号

    return exports
}

// LoadScript 加载并编译脚本
func (m *YaegiManager) LoadScript(script *Script) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 创建新的解释器实例
    i := interp.New(interp.Options{
        GoPath: "./yaegi-gopath",
    })

    // 注入导出符号
    if err := i.Use(m.exportedSymbols); err != nil {
        return fmt.Errorf("注入符号失败: %w", err)
    }

    // 注入DB实例
    if _, err := i.Eval(`package main

import (
    "itcodex/metadata"
)

var DB = metadata.DB
`); err != nil {
        return fmt.Errorf("注入DB失败: %w", err)
    }

    // 执行脚本
    if _, err := i.Eval(script.Content); err != nil {
        return fmt.Errorf("脚本编译失败: %w", err)
    }

    // 提取符号
    symbols := make(map[string]reflect.Value)

    // 根据钩子点提取对应函数
    switch script.HookPoint {
    case HookBeforeCreate:
        v, err := i.Eval("BeforeCreate")
        if err == nil {
            symbols["BeforeCreate"] = v
        }
    case HookAfterCreate:
        v, err := i.Eval("AfterCreate")
        if err == nil {
            symbols["AfterCreate"] = v
        }
    // ... 其他钩子
    case HookCustomAPI:
        v, err := i.Eval("Handle")
        if err == nil {
            symbols["Handle"] = v
        }
    }

    script.symbols = symbols
    m.interpreters[script.ID] = i

    // 注册到相应的映射
    m.registerScript(script)

    return nil
}

// ExecuteHook 执行钩子
func (m *YaegiManager) ExecuteHook(ctx context.Context, collection, hook string, data interface{}) (interface{}, error) {
    m.mu.RLock()
    key := collection + ":" + hook
    scripts := m.hooks[key]
    m.mu.RUnlock()

    // 按优先级排序
    sort.Slice(scripts, func(i, j int) bool {
        return scripts[i].Priority < scripts[j].Priority
    })

    var result interface{} = data
    for _, script := range scripts {
        if !script.Enabled {
            continue
        }

        fn, ok := script.symbols[hookMapFuncName(hook)]
        if !ok {
            continue
        }

        // 根据钩子类型调用
        // ... 使用reflect调用函数
        // 处理返回值和错误
    }

    return result, nil
}
```

### 5.3 钩子执行中间件（GoFrame）

放在 `internal/service/middleware`。CRUD 钩子由 Repository 在 create/update/destroy 前后调用更稳妥；HTTP 中间件只做上下文注入。自定义 API 仍走 `/api/custom/*` 通配，不在 `g.Meta` 里为每个脚本建路由。错误通过 `gerror` 返回，交给 `MiddlewareHandlerResponse`，不要手写 `WriteJson` 作为主路径。

```go
package middleware

import (
    "github.com/gogf/gf/v2/net/ghttp"

    "itcodex/server/internal/service/metadata"
)

// YaegiCRUDMiddleware 可选。优先在 Repository 内调钩子；此处仅示意 HTTP 接入。
func YaegiCRUDMiddleware(db *metadata.Database) func(r *ghttp.Request) {
    return func(r *ghttp.Request) {
        collectionName := r.GetRouterString("collection")
        if collectionName == "" {
            r.Middleware.Next()
            return
        }

        // 根据HTTP方法和路径判断钩子点
        hookPoint := determineHookPoint(r)

        // 读取请求body中的数据
        var data map[string]interface{}
        if r.IsPost() || r.IsPut() || r.IsPatch() {
            _ = r.GetJson(&data)
        }

        // 如果是更新/删除，先查询旧数据
        var oldData map[string]interface{}
        if hookPoint == "beforeUpdate" || hookPoint == "beforeDelete" {
            id := r.GetRouterString("id")
            if id != "" {
                // 从数据库查询旧数据
                // oldData = ...
            }
        }

        // 执行 before 钩子
        if strings.HasPrefix(hookPoint, "before") {
            ctx := r.Context()
            var input interface{}
            if hookPoint == "beforeUpdate" {
                input = []map[string]interface{}{oldData, data}
            } else if hookPoint == "beforeDelete" {
                input = oldData
            } else {
                input = data
            }

            result, err := yaegiMgr.ExecuteHook(ctx, collectionName, hookPoint, input)
            if err != nil {
                r.Response.WriteStatus(422)
                r.Response.WriteJson(g.Map{
                    "error": err.Error(),
                })
                return
            }

            // 如果钩子修改了data，替换回去
            if modified, ok := result.(map[string]interface{}); ok && hookPoint != "beforeDelete" {
                // 将修改后的数据重新设置到请求中
                // ...
            }
        }

        // 继续执行后续处理
        r.Middleware.Next()

        // 获取响应结果
        // 执行 after 钩子
        if strings.HasPrefix(hookPoint, "after") {
            // 解析响应数据
            // 执行 after 钩子
        }
    }
}

// CustomAPIRouter 自定义API路由注册
func CustomAPIRouter(yaegiMgr *yaegi.YaegiManager, group *ghttp.RouterGroup) {
    // 表级自定义API: /api/custom/c/:collection/*
    group.ALL("/c/:collection/*", func(r *ghttp.Request) {
        collectionName := r.GetRouterString("collection")
        subPath := r.GetRouterString("*")

        fullPath := "/c/" + collectionName + "/" + subPath
        method := r.Method

        key := method + ":" + fullPath
        script := yaegiMgr.GetCustomAPI(key)
        if script == nil {
            // 尝试全局匹配
            key = method + ":" + "/global/" + subPath
            script = yaegiMgr.GetCustomAPI(key)
        }

        if script == nil {
            r.Response.WriteStatus(404)
            r.Response.WriteJson(g.Map{"error": "自定义API不存在"})
            return
        }

        // 构建YaegiHTTPContext
        yaegiCtx := buildYaegiContext(r)

        // 调用脚本的Handle函数
        fn := script.symbols["Handle"]
        // reflect.Value 调用
        // ...
    })
}
```

## 6. 脚本示例

### 6.1 CRUD 钩子示例

```go
// 文件: scripts/orders_hooks.go
package main

import (
    "context"
    "fmt"

    "itcodex/utils"
)

// BeforeCreate 创建订单前自动生成订单号
func BeforeCreate(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
    // 生成订单号: ORD-yyyyMMdd-xxxxx
    now := utils.Now()
    datePart := utils.FormatTime(now, "20060102")
    seq := utils.NanoID(6)
    orderNo := fmt.Sprintf("ORD-%s-%s", datePart, seq)

    data["orderNo"] = orderNo

    // 如果金额大于10000，设置需要审批
    if amount, ok := data["amount"].(float64); ok && amount > 10000 {
        data["needsApproval"] = true
    }

    return data, nil
}

// AfterCreate 创建订单后发送通知
func AfterCreate(ctx context.Context, result map[string]interface{}) error {
    orderID := result["id"]
    orderNo := result["orderNo"]

    utils.LogInfo("新订单创建成功:", orderID, orderNo)

    // 这里可以调用其他服务发送通知
    // ...

    return nil
}

// BeforeUpdate 更新前检查
func BeforeUpdate(ctx context.Context, oldData, newData map[string]interface{}) (map[string]interface{}, error) {
    // 已完成的订单不能修改金额
    if oldData["status"] == "completed" {
        if _, ok := newData["amount"]; ok {
            return nil, fmt.Errorf("已完成的订单不能修改金额")
        }
    }
    return newData, nil
}
```

### 6.2 自定义 API 示例

```go
// 文件: scripts/users_api.go
package main

import (
    "fmt"

    "itcodex/context"
    "itcodex/utils"
    "itcodex/metadata"
)

// 绑定路径: POST /api/custom/c/users/:id/change-password
func Handle(ctx *context.YaegiHTTPContext) {
    userID := ctx.Params["id"]

    var req struct {
        OldPassword string `json:"oldPassword"`
        NewPassword string `json:"newPassword"`
    }
    if err := ctx.Request.BindJSON(&req); err != nil {
        ctx.Response.JSONError(400, "请求参数错误")
        return
    }

    // 查询用户
    users := metadata.DB.Collection("users")
    user, err := users.FindByID(userID)
    if err != nil {
        ctx.Response.JSONError(404, "用户不存在")
        return
    }

    // 验证旧密码
    if !utils.VerifyPassword(req.OldPassword, user["password"].(string)) {
        ctx.Response.JSONError(400, "旧密码错误")
        return
    }

    // 新密码长度校验
    if len(req.NewPassword) < 6 {
        ctx.Response.JSONError(400, "新密码长度不能少于6位")
        return
    }

    // 更新密码
    hashed, _ := utils.HashPassword(req.NewPassword)
    _, err = users.UpdateByID(userID, map[string]interface{}{
        "password": hashed,
    })
    if err != nil {
        ctx.Response.JSONError(500, "密码修改失败")
        return
    }

    ctx.Response.JSONSuccess(map[string]interface{}{
        "message": "密码修改成功",
    })
}
```

### 6.3 多字段联合校验示例（使用 CEL）

```go
// 文件: scripts/tasks_validation.go
package main

import (
    "context"
    "fmt"

    "itcodex/validation"
)

// BeforeValidate 校验开始日期和结束日期
func BeforeValidate(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
    // 使用CEL表达式校验
    ok, err := validation.ValidateCEL(data, `
        data.startDate == null ||
        data.endDate == null ||
        data.startDate <= data.endDate
    `)
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, validation.NewValidationError("开始日期不能晚于结束日期")
    }

    // 多字段校验：已完成任务必须有完成时间
    ok, err = validation.ValidateCEL(data, `
        data.status != 'completed' || data.completedAt != null
    `)
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, validation.NewFieldValidationError("completedAt", "已完成的任务必须填写完成时间")
    }

    return data, nil
}
```

## 7. 安全考虑

### 7.1 沙箱限制

1. **禁止访问文件系统**：不导出 os、io/ioutil 等包
2. **禁止网络访问**：不导出 net/http 客户端（需要HTTP调用应通过封装API）
3. **禁止系统调用**：不导出 syscall、os/exec 等包
4. **超时控制**：脚本执行设置超时（默认5秒）
5. **资源限制**：限制内存使用，防止死循环

```go
// 黑名单包：不允许脚本import
var forbiddenPackages = map[string]bool{
    "os":        true,
    "os/exec":   true,
    "os/signal": true,
    "syscall":   true,
    "net":       true,
    "net/http":  true, // 脚本中需要HTTP调用通过封装的httpClient
    "io/ioutil": true,
    "plugin":    true,
    "unsafe":    true,
    "runtime":   true,
}
```

### 7.2 脚本验证

在保存脚本前，尝试编译脚本验证语法正确性：

```go
func (m *YaegiManager) ValidateScript(content string) error {
    i := interp.New(interp.Options{})
    i.Use(m.exportedSymbols)
    _, err := i.Eval(content)
    return err
}
```
