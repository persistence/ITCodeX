# ITCodeX 元数据模块 - 测试 Client 设计方案

> 版本: v1.0
> 日期: 2026-09-03
> 技术栈: Go + net/http + testing

## 1. 概述

本测试 Client 用于对 ITCodeX 元数据服务端进行端到端集成测试，通过 HTTP 接口调用验证元数据管理和数据 CRUD 的完整功能。

### 1.1 设计目标
- 完整覆盖服务端所有公开 API 接口
- 支持自动化测试用例编写和执行
- 提供清晰的测试报告和错误定位
- 支持测试数据的初始化和清理
- 可扩展支持自定义 API 和 Yaegi 脚本测试

### 1.2 服务端信息
- 默认地址: `http://localhost:8000`
- API 前缀: `/api`
- 数据格式: JSON

## 2. 项目结构

```
e:\itcodex\client
├── cmd/
│   └── test-client/
│       └── main.go           # 测试客户端入口（可选）
├── internal/
│   ├── client/               # HTTP 客户端封装
│   │   ├── client.go         # 基础客户端
│   │   ├── meta.go           # 元数据管理 API
│   │   ├── crud.go           # 数据 CRUD API
│   │   └── types.go          # 请求/响应类型定义
│   └── tests/                # 测试用例
│       ├── meta_test.go      # 元数据管理测试
│       ├── crud_test.go      # CRUD 操作测试
│       ├── filter_test.go    # 查询过滤测试
│       ├── validation_test.go# 校验功能测试
│       └── suite_test.go     # 测试套件和公共工具
├── docs/
│   └── 测试Client设计方案.md # 本文档
├── go.mod
└── go.sum
```

## 3. 核心模块设计

### 3.1 基础 HTTP 客户端 (client.go)

**职责:**
- 封装 HTTP 请求发送
- 统一处理请求/响应格式
- 管理基础配置（地址、超时等）
- 统一错误处理

**核心结构:**
```go
type Client struct {
    BaseURL    string
    HTTPClient *http.Client
}

type Response struct {
    Code    int                    `json:"code"`
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data"`
}
```

**主要方法:**
- `NewClient(baseURL string) *Client` - 创建客户端
- `Get(ctx, path string, params map[string]string) (*Response, error)` - GET 请求
- `Post(ctx, path string, body interface{}) (*Response, error)` - POST 请求
- `Put(ctx, path string, body interface{}) (*Response, error)` - PUT 请求
- `Delete(ctx, path string, params map[string]string) (*Response, error)` - DELETE 请求

### 3.2 元数据管理 API (meta.go)

**覆盖接口:**
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/meta/collections` | GET | 获取 Collection 列表 |
| `/api/meta/collections` | POST | 创建 Collection |
| `/api/meta/collections/:name` | GET | 获取 Collection 详情 |
| `/api/meta/collections/:name` | DELETE | 删除 Collection |
| `/api/meta/collections/:name/fields` | GET | 获取字段列表 |
| `/api/meta/collections/:name/fields` | POST | 添加字段 |
| `/api/meta/collections/:name/fields/:field` | DELETE | 删除字段 |
| `/api/meta/scripts` | GET | 获取脚本列表 |
| `/api/meta/scripts` | POST | 创建/更新脚本 |
| `/api/meta/scripts/:id/disable` | POST | 禁用脚本 |

**核心方法:**
```go
func (c *Client) ListCollections(ctx context.Context) ([]Collection, error)
func (c *Client) CreateCollection(ctx context.Context, input CreateCollectionInput) (*Collection, error)
func (c *Client) GetCollection(ctx context.Context, name string) (*Collection, error)
func (c *Client) DropCollection(ctx context.Context, name string) error
func (c *Client) ListFields(ctx context.Context, collection string) ([]Field, error)
func (c *Client) AddField(ctx context.Context, collection string, input CreateFieldInput) ([]Field, error)
func (c *Client) RemoveField(ctx context.Context, collection, field string) ([]Field, error)
```

### 3.3 数据 CRUD API (crud.go)

**覆盖接口:**
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/c/:collection` | GET | 查询列表（分页、过滤、排序） |
| `/api/c/:collection/:id` | GET | 查询单条 |
| `/api/c/:collection` | POST | 创建记录 |
| `/api/c/:collection/:id` | PUT | 更新记录 |
| `/api/c/:collection/:id` | DELETE | 删除单条 |
| `/api/c/:collection` | DELETE | 批量删除 |

**核心方法:**
```go
func (c *Client) List(ctx context.Context, collection string, opts *FindOptions) (*ListResult, error)
func (c *Client) Get(ctx context.Context, collection, id string, opts *FindOneOptions) (map[string]interface{}, error)
func (c *Client) Create(ctx context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error)
func (c *Client) Update(ctx context.Context, collection, id string, data map[string]interface{}) (map[string]interface{}, error)
func (c *Client) Delete(ctx context.Context, collection, id string) (int64, error)
func (c *Client) BulkDelete(ctx context.Context, collection string, filter Filter) (int64, error)
func (c *Client) Count(ctx context.Context, collection string, filter Filter) (int64, error)
```

### 3.4 类型定义 (types.go)

```go
// Collection 类型
type Collection struct {
    Name        string                 `json:"name"`
    DisplayName string                 `json:"displayName"`
    Type        string                 `json:"type"`
    Options     map[string]interface{} `json:"options"`
}

type CreateCollectionInput struct {
    Name        string                 `json:"name"`
    DisplayName string                 `json:"displayName"`
    Type        string                 `json:"type"`
    Description string                 `json:"description,omitempty"`
    Options     map[string]interface{} `json:"options,omitempty"`
    Fields      []CreateFieldInput     `json:"fields,omitempty"`
}

// Field 类型
type Field struct {
    Name        string                 `json:"name"`
    DisplayName string                 `json:"displayName"`
    Type        string                 `json:"type"`
    Required    bool                   `json:"required"`
    Unique      bool                   `json:"unique"`
    IsSystem    bool                   `json:"isSystem"`
    Options     map[string]interface{} `json:"options"`
}

type CreateFieldInput struct {
    Name         string                 `json:"name"`
    DisplayName  string                 `json:"displayName"`
    Type         string                 `json:"type"`
    IsRequired   bool                   `json:"isRequired,omitempty"`
    IsUnique     bool                   `json:"isUnique,omitempty"`
    DefaultValue interface{}            `json:"defaultValue,omitempty"`
    Options      map[string]interface{} `json:"options,omitempty"`
}

// 查询选项
type FindOptions struct {
    Filter   Filter
    Sort     []string
    Fields   []string
    Except   []string
    Page     int
    PageSize int
}

type FindOneOptions struct {
    Fields []string
    Except []string
}

type ListResult struct {
    List       []map[string]interface{} `json:"list"`
    Total      int64                    `json:"total"`
    Page       int                      `json:"page"`
    PageSize   int                      `json:"pageSize"`
    TotalPages int                      `json:"totalPages"`
}

// Filter 类型
type Filter map[string]interface{}
```

## 4. 测试用例设计

### 4.1 测试套件 (suite_test.go)

**公共功能:**
- 测试客户端初始化
- 测试数据自动清理
- 辅助断言方法
- 测试 Collection 创建/删除工具

```go
type TestSuite struct {
    client *Client
    ctx    context.Context
}

func setupTest(t *testing.T) *TestSuite {
    // 初始化客户端，确保服务端运行
    // 返回测试套件实例
}

func (s *TestSuite) createTestCollection(t *testing.T, name string) {
    // 创建测试用 Collection，自动清理
}

func (s *TestSuite) cleanupTestCollection(t *testing.T, name string) {
    // 清理测试数据
}
```

### 4.2 元数据管理测试 (meta_test.go)

| 测试用例 | 优先级 | 说明 |
|---------|--------|------|
| TestListCollections_Empty | P0 | 初始状态下列表为空或只有系统表 |
| TestCreateCollection_Success | P0 | 创建普通 Collection 成功 |
| TestCreateCollection_WithFields | P0 | 带字段创建 Collection |
| TestCreateCollection_DuplicateName | P1 | 重名创建返回冲突错误 |
| TestGetCollection_Exists | P0 | 获取存在的 Collection 详情 |
| TestGetCollection_NotFound | P1 | 获取不存在的 Collection 返回 404 |
| TestDropCollection_Success | P0 | 删除 Collection 成功 |
| TestDropCollection_NotFound | P1 | 删除不存在的 Collection 返回错误 |
| TestAddField_Success | P0 | 添加字段成功 |
| TestAddField_DuplicateName | P1 | 添加重名字段返回错误 |
| TestListFields_Success | P0 | 获取字段列表成功 |
| TestRemoveField_Success | P0 | 删除字段成功 |

### 4.3 CRUD 操作测试 (crud_test.go)

| 测试用例 | 优先级 | 说明 |
|---------|--------|------|
| TestCreate_SingleRecord | P0 | 创建单条记录成功 |
| TestCreate_RequiredFieldMissing | P0 | 必填字段缺失返回校验错误 |
| TestGet_Exists | P0 | 获取存在的记录成功 |
| TestGet_NotFound | P1 | 获取不存在的记录返回 404 |
| TestList_DefaultPagination | P0 | 默认分页查询 |
| TestList_CustomPagination | P1 | 自定义分页参数 |
| TestUpdate_Success | P0 | 更新记录成功 |
| TestUpdate_NotFound | P1 | 更新不存在的记录 |
| TestDelete_Success | P0 | 删除记录成功 |
| TestDelete_NotFound | P1 | 删除不存在的记录 |
| TestBulkDelete_WithFilter | P1 | 按条件批量删除 |
| TestCount_Success | P1 | 统计数量正确 |

### 4.4 查询过滤测试 (filter_test.go)

| 测试用例 | 优先级 | 说明 |
|---------|--------|------|
| TestFilter_Eq | P0 | 等于查询 ($eq) |
| TestFilter_Ne | P1 | 不等于查询 ($ne) |
| TestFilter_Gt/Gte | P0 | 大于/大于等于查询 |
| TestFilter_Lt/Lte | P0 | 小于/小于等于查询 |
| TestFilter_In | P0 | IN 查询 |
| TestFilter_Like | P1 | 模糊查询 ($like) |
| TestFilter_StartsWith | P2 | 前缀匹配 |
| TestFilter_EndsWith | P2 | 后缀匹配 |
| TestFilter_And | P0 | AND 组合条件 |
| TestFilter_Or | P0 | OR 组合条件 |
| TestSort_Ascending | P0 | 升序排序 |
| TestSort_Descending | P0 | 降序排序 |
| TestSort_MultipleFields | P1 | 多字段排序 |
| TestFields_SelectFields | P1 | 指定返回字段 |
| TestFields_ExceptFields | P1 | 排除字段 |

### 4.5 校验功能测试 (validation_test.go)

| 测试用例 | 优先级 | 说明 |
|---------|--------|------|
| TestValidation_Required | P0 | 必填校验 |
| TestValidation_Unique | P0 | 唯一约束校验 |
| TestValidation_StringLength | P1 | 字符串长度校验 |
| TestValidation_NumberRange | P1 | 数字范围校验 |
| TestValidation_Pattern | P1 | 正则表达式校验 |
| TestValidation_MultipleErrors | P1 | 多个字段同时校验失败 |

### 4.6 Yaegi 脚本测试 (script_test.go) - 可选

| 测试用例 | 优先级 | 说明 |
|---------|--------|------|
| TestCreateScript_Success | P2 | 创建钩子脚本成功 |
| TestScript_BeforeCreateHook | P2 | beforeCreate 钩子执行 |
| TestDisableScript_Success | P2 | 禁用脚本成功 |

## 5. 测试执行流程

### 5.1 前置条件
1. 元数据服务端已启动并监听 8000 端口
2. 服务端数据库已初始化
3. 测试客户端网络可访问服务端

### 5.2 测试执行顺序
1. 元数据管理相关测试（创建/删除 Collection 和 Field）
2. 基础 CRUD 测试
3. 查询过滤和排序测试
4. 数据校验测试
5. Yaegi 脚本测试（可选）

### 5.3 测试数据清理
- 每个测试用例创建的 Collection 在测试结束后自动删除
- 使用 `t.Cleanup()` 注册清理函数
- 支持测试失败时保留数据用于排查

## 6. 错误处理规范

### 6.1 HTTP 错误码处理
| 状态码 | 说明 | 处理方式 |
|--------|------|----------|
| 200 | 成功 | 解析 Data 字段 |
| 201 | 创建成功 | 解析返回数据 |
| 400 | 请求参数错误 | 返回错误信息 |
| 404 | 资源不存在 | 返回 NotFoundError |
| 409 | 资源冲突（重名） | 返回 AlreadyExistsError |
| 422 | 校验失败 | 返回 ValidationError，包含详细字段错误 |
| 500 | 服务器内部错误 | 返回错误信息 |

### 6.2 自定义错误类型
```go
type APIError struct {
    Code    int
    Message string
    Data    interface{}
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

type ValidationError struct {
    FieldErrors map[string][]string `json:"fieldErrors"`
    TableErrors []string            `json:"tableErrors"`
}
```

## 7. 使用示例

### 7.1 客户端初始化
```go
func TestExample(t *testing.T) {
    ctx := context.Background()
    client := NewClient("http://localhost:8000")
    
    // 创建测试表
    coll, err := client.CreateCollection(ctx, CreateCollectionInput{
        Name:        "test_posts",
        DisplayName: "测试文章",
        Type:        "general",
        Fields: []CreateFieldInput{
            {Name: "title", DisplayName: "标题", Type: "string", IsRequired: true},
            {Name: "content", DisplayName: "内容", Type: "text"},
            {Name: "views", DisplayName: "浏览量", Type: "number"},
        },
    })
    assert.NoError(t, err)
    assert.NotNil(t, coll)
    
    // 清理
    t.Cleanup(func() {
        client.DropCollection(ctx, "test_posts")
    })
    
    // 创建数据
    post, err := client.Create(ctx, "test_posts", map[string]interface{}{
        "title":   "测试文章标题",
        "content": "这是文章内容",
        "views":   0,
    })
    assert.NoError(t, err)
    assert.NotZero(t, post["id"])
}
```

## 8. 后续扩展计划

1. **并发测试**: 添加并发请求测试，验证线程安全性
2. **性能测试**: 添加基准测试（Benchmark）
3. **关联查询测试**: 当关系字段实现后，测试关联数据操作
4. **事务测试**: 当批量操作 API 实现后，测试事务特性
5. **自定义 API 测试**: 支持测试 Yaegi 注册的自定义 API
6. **测试报告**: 集成测试报告生成（HTML/JSON）

## 9. 注意事项

1. 测试客户端仅用于测试，不要在生产环境使用
2. 测试会操作服务端数据，请确保连接到测试环境
3. 默认服务端地址为 `http://localhost:8000`，可通过环境变量 `TEST_SERVER_URL` 修改
4. 每个测试用例应独立，不依赖其他测试的执行结果
5. 测试用例命名遵循 `TestXxx_Yyy` 格式，清晰描述测试场景
