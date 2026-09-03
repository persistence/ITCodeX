# ITCodeX 元数据模块 - 设计方案

> 版本: v1.0
> 日期: 2026-09-03
> 技术栈: GoFrame + CobaltDB(嵌入式) + CEL-Go + Yaegi
> **代码根路径**: [d:\code\ITCodeX\server](file:///d:/code/ITCodeX/server)

## 文档目录

| 序号 | 文档 | 说明 |
|------|------|------|
| 01 | [总体架构设计](./01-总体架构设计.md) | 整体架构分层、模块职责、启动流程、系统表结构 |
| 02 | [数据模型设计](./02-数据模型设计.md) | Database/Collection/Field/Repository 等核心结构体定义、内置字段类型 |
| 03 | [CEL校验设计](./03-CEL校验设计.md) | CEL-Go 环境配置、单字段/多字段联合校验、校验规则缓存、执行流程 |
| 04 | [Yaegi二开设计](./04-Yaegi二开设计.md) | Yaegi 脚本引擎、CRUD钩子、自定义API、沙箱环境、Path规范、脚本示例 |
| 05 | [API接口设计](./05-API接口设计.md) | 元数据管理接口、标准CRUD接口、自定义API接口规范、Filter操作符 |
| 06 | [项目结构设计](./06-项目结构设计.md) | 目录结构、GoFrame路由注册、Controller/Logic分层、配置示例 |
| 07 | [CobaltDB集成设计](./07-CobaltDB集成设计.md) | 嵌入式数据库适配、类型映射、建表SQL、Schema同步、Filter到SQL翻译 |
| 08 | [测试方案](./08-测试方案.md) | 测试策略、测试框架、各模块测试用例、基准测试、覆盖率目标 |

## 技术选型

| 组件 | 技术 | 版本 | 说明 |
|------|------|------|------|
| Web框架 | GoFrame | v2 | 高性能Go Web框架 |
| 数据库 | CobaltDB | - | 嵌入式数据库，零部署 |
| 校验引擎 | cel-go | latest | Google CEL，支持多字段联合校验 |
| 脚本引擎 | Yaegi | latest | Go解释器，运行时二开 |
| 主键策略 | Snowflake/UUID/NanoID | - | 支持多种主键生成 |

## 核心特性

1. **动态数据表**：运行时创建/修改/删除数据表和字段，无需重启
2. **丰富字段类型**：支持基础、选择、媒体、日期、关系、几何、高级、系统共 8 大类 40+ 字段类型
3. **CEL校验**：单字段/多字段联合校验，支持自定义表达式，编译缓存高性能
4. **Yaegi二开**：
   - CRUD 生命周期钩子（before/after create/update/delete/validate）
   - 每个元数据表支持自定义API接口
   - Go 脚本语法，热加载无需重新编译
   - 安全沙箱环境
5. **类NocoBase API**：兼容 NocoBase 风格的 Filter 语法（$eq/$gt/$in/$and/$or 等）
6. **嵌入式部署**：CobaltDB 嵌入式使用，单文件存储，零依赖

## API Path 规范

| 接口类型 | Path 前缀 | 说明 |
|----------|-----------|------|
| 元数据管理 | `/api/meta/*` | Collection/Field/脚本管理 |
| 标准 CRUD | `/api/c/:collection/*` | 动态生成的数据增删改查 |
| 表级自定义API | `/api/custom/c/:collection/*` | Yaegi提供的单表自定义接口 |
| 全局自定义API | `/api/custom/global/*` | Yaegi提供的全局自定义接口 |

## 快速开发指引

### 1. 第一阶段（核心可用）
- Database 管理器 + CobaltDB 连接
- 基础字段类型：string/text/integer/bigint/float/double/boolean/datetime
- 系统表自动创建和同步
- Repository 基础 CRUD（find/findOne/create/update/destroy）
- 基础 Filter 解析（$eq/$gt/$lt/$in/$and/$or/$like）
- 简单单字段校验（required/min/max/pattern）

### 2. 第二阶段（元数据管理）
- 完整字段类型实现
- CEL 单字段校验
- Collection 和 Field 元数据管理 API
- 字段校验规则配置

### 3. 第三阶段（高级查询和校验）
- CEL 多字段联合校验
- 关系字段（belongsTo/hasOne/hasMany/belongsToMany）
- 关联查询和关联操作
- 索引管理

### 4. 第四阶段（Yaegi二开）
- Yaegi 解释器初始化和沙箱
- CRUD before/after 钩子执行
- 自定义 API 路由注册和执行
- 脚本管理 API（增删改查、热加载）
- 脚本示例和文档

### 5. 第五阶段（增强）
- 高级字段类型（公式/加密/几何/排序/序列）
- 树表/日历表/评论表/文件表特殊类型
- 性能优化（查询缓存、批量操作）
- 错误处理完善
- 完整测试覆盖
