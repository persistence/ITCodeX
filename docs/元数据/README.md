# ITCodeX 元数据模块

> 动态元数据引擎 - 类 NocoBase 风格的低代码数据建模能力
>
> 技术栈: **GoFrame** + **MySQL 8** + **CEL-Go** + **Yaegi**
>
> **代码实现路径**: [d:\code\ITCodeX\server](file:///d:/code/ITCodeX/server)
>
> 详细目录结构和模块划分见 [项目结构设计](./设计方案/06-项目结构设计.md)

## 目录结构

```
docs/元数据/
├── README.md                    # 本文档，模块入口
├── 需求描述/                     # 参考需求文档（基于NocoBase整理）
│   ├── 字段/                    # 40+字段类型详细需求
│   └── 数据表/                  # 各类数据表类型需求
├── api/                         # API参考文档（基于NocoBase整理）
└── 设计方案/                     # ✅ 本项目设计方案
    ├── README.md               # 设计方案总览
    ├── 01-总体架构设计.md
    ├── 02-数据模型设计.md
    ├── 03-CEL校验设计.md
    ├── 04-Yaegi二开设计.md
    ├── 05-API接口设计.md
    ├── 06-项目结构设计.md
    ├── 07-MySQL集成设计.md
    └── 08-测试方案.md
```

## 快速导航

### 📋 需求描述（参考）

需求基于 NocoBase 官方文档整理，用于指导功能范围：

| 分类 | 文档 | 说明 |
|------|------|------|
| 基础 | [字段概述](./需求描述/字段/01-字段概述.md) | 字段分类、Interface类型、数据类型映射 |
| 基础 | [字段校验](./需求描述/字段/02-字段校验.md) | 校验规则配置、服务端/客户端校验 |
| 数据表 | [数据表概述](./需求描述/数据表/01-数据表Collection.md) | 表结构类型、创建流程 |
| 数据表 | [普通表](./需求描述/数据表/02-普通表GeneralCollection.md) | 普通表配置、预设字段 |
| 数据表 | [树表](./需求描述/数据表/03-树表TreeCollection.md) | 层级结构数据 |
| 数据表 | [日历表](./需求描述/数据表/04-日历表CalendarCollection.md) | 时间范围事件数据 |
| 数据表 | [评论表](./需求描述/数据表/05-评论表CommentCollection.md) | 评论/讨论数据 |
| 数据表 | [文件表](./需求描述/数据表/06-文件表FileCollection.md) | 文件元数据存储 |
| 字段 | [基础字段(11种)](./需求描述/字段/) | 单行文本/多行文本/手机号/邮箱/URL/数字/整数/百分比/颜色/图标/密码 |
| 字段 | [选择字段(6种)](./需求描述/字段/) | 勾选/下拉单选/单选框组/下拉多选/复选框组/中国行政区 |
| 字段 | [媒体字段(5种)](./需求描述/字段/) | Markdown/富文本/MarkdownVditor/附件(关联)/附件(URL) |
| 字段 | [日期时间(5种)](./需求描述/字段/) | 日期时间无时区/有时区/时间/日期/Unix时间戳 |
| 字段 | [关系字段(5种)](./需求描述/字段/) | belongsTo/hasMany/hasOne/belongsToMany/多对多(数组) |
| 字段 | [几何图形(4种)](./需求描述/字段/) | 点/线/圆/多边形 |
| 字段 | [高级类型(8种)](./需求描述/字段/) | UUID/NanoID/排序/公式/自动序列/JSON/表选择器/加密字段 |
| 字段 | [系统信息(5种)](./需求描述/字段/) | 创建时间/最后更新时间/创建人/最后更新人/表OID |

### 📖 API 参考

基于 NocoBase JS API 的参考设计，用于理解接口能力：

| 文档 | 说明 |
|------|------|
| [Database 数据库](./api/01-index-Database数据库.md) | 数据库连接、Collection管理、事件、迁移、扩展注册 |
| [Collection 数据表](./api/02-collection-数据表.md) | 表定义、字段管理、索引配置、同步 |
| [Field 字段](./api/03-field-字段.md) | 字段基类、内置字段类型、关联字段配置 |
| [Operators 查询运算符](./api/04-operators-查询运算符.md) | 完整的Filter操作符列表（通用/逻辑/数字/字符串/日期/数组/关系） |
| [Repository 数据仓库](./api/05-repository-数据仓库.md) | CRUD接口、查询选项、关联操作、事务 |
| [公共类型](./api/06-shared-公共类型.md) | 共享类型定义 |

### 🏗️ 设计方案（本项目实现）

**从这里开始看起！** 这是为 ITCodeX 定制的实际实现方案：

| 序号 | 文档 | 核心内容 |
|------|------|----------|
| - | [设计方案总览](./设计方案/README.md) | 对标范围、需求/API 取舍表、阶段路线图 |
| 1 | [总体架构设计](./设计方案/01-总体架构设计.md) | 7层架构、4张系统表结构、启动流程、设计原则 |
| 2 | [数据模型设计](./设计方案/02-数据模型设计.md) | Go结构体定义、8大类字段类型实现、Repository接口 |
| 3 | [CEL校验设计](./设计方案/03-CEL校验设计.md) | CEL环境配置、单字段/多字段联合校验、缓存机制 |
| 4 | [Yaegi二开设计](./设计方案/04-Yaegi二开设计.md) | 钩子机制、自定义API、沙箱环境、脚本示例、安全限制 |
| 5 | [API接口设计](./设计方案/05-API接口设计.md) | 实际HTTP API规范、请求/响应格式、Filter操作符 |
| 6 | [项目结构设计](./设计方案/06-项目结构设计.md) | GoFrame目录结构、路由注册、核心代码示例 |
| 7 | [MySQL集成设计](./设计方案/07-MySQL集成设计.md) | Docker 本机 MySQL、类型映射、DDL、Filter到SQL翻译、事务、性能优化 |
| 8 | [测试方案](./设计方案/08-测试方案.md) | 测试策略、测试框架、各模块完整测试用例、基准测试、覆盖率目标 |

## 核心特性

| 特性 | 说明 |
|------|------|
| 🚀 **动态建模** | 运行时创建/修改/删除数据表和字段，无需重启服务 |
| 📦 **MySQL Docker** | 本机 Docker MySQL 8（root/123456，库 itcodex），数据持久化 |
| 🎯 **丰富字段类型** | 目标覆盖 8 大类 Interface，按阶段交付，不是一次做完 |
| ✅ **CEL校验** | 使用 Google CEL 实现单字段/多字段联合校验，编译缓存高性能 |
| 🔌 **Yaegi热插拔二开** | Go 脚本解释器，CRUD钩子+自定义API，热加载无需编译 |
| 🔗 **关系支持** | 完整的 belongsTo/hasOne/hasMany/belongsToMany 关联关系 |
| 📊 **强大查询** | 兼容 NocoBase 风格的 Filter 语法（$eq/$gt/$in/$and/$or/$like等） |

## API Path 规范

| 接口类型 | 路径前缀 | 说明 |
|----------|----------|------|
| 元数据管理 | `/api/meta/*` | Collection/Field/脚本的增删改查管理接口 |
| 标准 CRUD | `/api/c/:collectionName/*` | 动态生成的业务数据增删改查接口 |
| 表级自定义API | `/api/custom/c/:collectionName/*` | Yaegi脚本提供的单表自定义业务接口 |
| 全局自定义API | `/api/custom/global/*` | Yaegi脚本提供的全局自定义接口 |

### 标准 CRUD 接口示例

```
# 查询用户列表
GET    /api/c/users?filter={"age":{"$gt":18}}&sort=-createdAt&page=1&pageSize=20

# 查询单个用户
GET    /api/c/users/123

# 创建用户
POST   /api/c/users
Body:  { "name": "张三", "email": "zhangsan@example.com" }

# 更新用户
PUT    /api/c/users/123
Body:  { "age": 25 }

# 删除用户
DELETE /api/c/users/123
```

## Yaegi 二开示例

### 钩子脚本（创建前自动生成订单号）

```go
// 钩子点: beforeCreate
// 绑定表: orders
package main

import (
    "context"
    "fmt"
    "itcodex/utils"
)

func BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {
    now := utils.Now()
    datePart := utils.FormatTime(now, "20060102")
    seq := utils.NanoID(6)
    data["orderNo"] = fmt.Sprintf("ORD-%s-%s", datePart, seq)
    return data, nil
}
```

### 自定义API脚本（修改密码）

```go
// API路径: POST /api/custom/c/users/:id/change-password
package main

import (
    "itcodex/context"
    "itcodex/metadata"
    "itcodex/utils"
)

func Handle(ctx *context.YaegiHTTPContext) {
    userID := ctx.Params["id"]
    var req struct {
        OldPassword string `json:"oldPassword"`
        NewPassword string `json:"newPassword"`
    }
    if err := ctx.Request.BindJSON(&req); err != nil {
        ctx.Response.JSONError(400, "参数错误")
        return
    }

    users := metadata.DB.Collection("users")
    user, _ := users.FindByID(userID)

    if !utils.VerifyPassword(req.OldPassword, user["password"].(string)) {
        ctx.Response.JSONError(400, "旧密码错误")
        return
    }

    hashed, _ := utils.HashPassword(req.NewPassword)
    users.UpdateByID(userID, map[string]any{"password": hashed})

    ctx.Response.JSONSuccess(map[string]string{"message": "修改成功"})
}
```

## 开发阶段路线图

| 阶段 | 目标 | 核心交付 |
|------|------|----------|
| **第一阶段** | 核心可用 | MySQL、普通表、基础字段、CRUD、Filter 一期子集、必填/长度/范围校验 |
| **第二阶段** | 元数据管理 | 选择/媒体 Interface、CEL 单字段、Collection/Field API |
| **第三阶段** | 高级能力 | CEL 多字段、关系字段、appends、关联写入、索引 |
| **第四阶段** | 二开支持 | Yaegi 钩子、自定义 API、脚本管理、热加载 |
| **第五阶段** | 增强完善 | 公式/加密/几何、树/日历/评论/文件表行为、性能与测试 |

范围裁剪（视图/SQL/继承/外部表、权限、工作流等不做）见 [设计方案总览](./设计方案/README.md#对标-nocobase-的范围)。

## 技术栈依赖

| 组件 | 技术 | 用途 |
|------|------|------|
| Web框架 | GoFrame v2 | HTTP服务、路由、中间件、ORM适配 |
| 数据库 | MySQL 8 | Docker 本机关系型数据库，存储元数据和业务数据 |
| 校验引擎 | cel-go | Google Common Expression Language，表达式校验 |
| 脚本引擎 | Yaegi | Go语言解释器，运行时二次开发 |
| 主键生成 | Snowflake / UUID / NanoID | 多种主键策略 |
