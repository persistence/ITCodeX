# ITCodeX 元数据模块 - CobaltDB 嵌入式集成设计

> 版本: v1.0
> 日期: 2026-09-03
> 数据库: CobaltDB (嵌入式)

## 1. CobaltDB 简介与集成策略

### 1.1 嵌入式使用场景
- CobaltDB 作为嵌入式数据库直接集成到 Go 进程中
- 无需独立部署数据库服务
- 数据存储在本地文件中
- 零运维成本，适合单机/桌面/边缘场景

### 1.2 GoFrame ORM 适配
CobaltDB 需要适配 GoFrame 的 ORM 接口 `gdb.DB`，有两种策略：

**策略A：使用 SQLite 兼容模式**
- 如果 CobaltDB 兼容 SQLite 语法和驱动接口，直接复用 GoFrame 的 SQLite 驱动
- 配置中 `type: "sqlite"`，link 指定 cobaltdb 协议

**策略B：自定义 GoFrame 驱动适配**
- 如果 CobaltDB 提供独立的 Go driver，需要实现 GoFrame 的 `gdb.Driver` 接口
- 注册驱动类型 `cobaltdb`

```go
// 注册自定义CobaltDB驱动到GoFrame
import (
    "github.com/gogf/gf/v2/database/gdb"
    cobaltdrv "your-module/driver/cobaltdb"
)

func init() {
    if err := gdb.Register("cobaltdb", &cobaltdrv.Driver{}); err != nil {
        panic(err)
    }
}
```

配置示例：
```yaml
database:
  default:
    type: "cobaltdb"
    link: "cobaltdb::@file(./data/app.db?cache=shared&mode=rwc)"
```

## 2. 数据类型映射

### 2.1 字段类型到 CobaltDB 类型映射

| 元数据字段类型 | CobaltDB 存储类型 | 说明 |
|---------------|------------------|------|
| string | VARCHAR(n) / TEXT | 短字符串用VARCHAR(255)，长文本用TEXT |
| text | TEXT / LONGTEXT | 多行文本、富文本 |
| integer | INTEGER | 32位整数 |
| bigInt | BIGINT / INTEGER | 64位整数（Snowflake ID等） |
| float | REAL / FLOAT | 单精度浮点 |
| double | DOUBLE / REAL | 双精度浮点 |
| decimal | NUMERIC(p,s) / DECIMAL | 精确小数（金额等） |
| boolean | BOOLEAN / INTEGER(0/1) | 布尔值 |
| date | DATE | 仅日期 |
| time | TIME | 仅时间 |
| datetime | DATETIME / TIMESTAMP | 日期时间 |
| timestamp | TIMESTAMP | Unix时间戳存储也可用BIGINT |
| json | JSON / TEXT | JSON数据（CobaltDB如果原生支持JSON则优先用JSON） |
| jsonb | JSON / TEXT | 二进制JSON（兼容） |
| uuid | CHAR(36) / TEXT(36) | UUID字符串 |
| nanoid | VARCHAR(21) | NanoID字符串 |
| password | VARCHAR(255) | 密码哈希 |
| blob | BLOB / BINARY | 二进制数据 |
| point | JSON / TEXT(GeoJSON) | 几何点 |
| sort | INTEGER | 排序字段 |

## 3. 数据库初始化

### 3.1 系统表创建

启动时自动创建以下系统表（如果不存在）：

```sql
-- collections 表：数据表定义
CREATE TABLE IF NOT EXISTS collections (
    id           BIGINT PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    table_name   VARCHAR(255) NOT NULL UNIQUE,
    type         VARCHAR(50)  NOT NULL DEFAULT 'general',
    description  TEXT,
    categories   TEXT,  -- JSON数组
    options      TEXT,  -- JSON对象
    filter_target_key VARCHAR(50) NOT NULL DEFAULT 'id',
    auto_gen_id  BOOLEAN NOT NULL DEFAULT 1,
    sortable     TEXT,  -- JSON对象
    is_system    BOOLEAN NOT NULL DEFAULT 0,
    created_at   DATETIME,
    updated_at   DATETIME,
    created_by   BIGINT,
    updated_by   BIGINT
);

CREATE INDEX idx_collections_type ON collections(type);
CREATE INDEX idx_collections_categories ON collections(categories);

-- fields 表：字段定义
CREATE TABLE IF NOT EXISTS fields (
    id              BIGINT PRIMARY KEY,
    collection_name VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    display_name    VARCHAR(255),
    type            VARCHAR(50)  NOT NULL,
    data_type       VARCHAR(50)  NOT NULL,
    interface_type  VARCHAR(50),
    options         TEXT,  -- JSON
    validation_rules TEXT, -- JSON数组
    "sort"          INTEGER NOT NULL DEFAULT 0,
    is_system       BOOLEAN NOT NULL DEFAULT 0,
    is_required     BOOLEAN NOT NULL DEFAULT 0,
    is_unique       BOOLEAN NOT NULL DEFAULT 0,
    default_value   TEXT,
    description     TEXT,
    created_at      DATETIME,
    updated_at      DATETIME,
    UNIQUE(collection_name, name)
);

CREATE INDEX idx_fields_collection ON fields(collection_name);
CREATE INDEX idx_fields_type ON fields(type);

-- indexes 表：索引定义
CREATE TABLE IF NOT EXISTS indexes (
    id              BIGINT PRIMARY KEY,
    collection_name VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    fields          TEXT NOT NULL, -- JSON数组
    "unique"        BOOLEAN NOT NULL DEFAULT 0,
    options         TEXT,  -- JSON
    UNIQUE(collection_name, name)
);

-- yaegi_scripts 表：Yaegi脚本
CREATE TABLE IF NOT EXISTS yaegi_scripts (
    id              BIGINT PRIMARY KEY,
    collection_name VARCHAR(255),
    name            VARCHAR(255) NOT NULL,
    hook_point      VARCHAR(100) NOT NULL,
    script_content  TEXT NOT NULL,
    api_path        VARCHAR(500),
    http_method     VARCHAR(20),
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    priority        INTEGER NOT NULL DEFAULT 0,
    options         TEXT, -- JSON
    created_at      DATETIME,
    updated_at      DATETIME
);

CREATE INDEX idx_scripts_collection ON yaegi_scripts(collection_name);
CREATE INDEX idx_scripts_hook ON yaegi_scripts(hook_point);
CREATE INDEX idx_scripts_api ON yaegi_scripts(api_path, http_method);
```

### 3.2 业务表动态创建示例

创建 `users` 表的 DDL 示例：

```sql
CREATE TABLE users (
    id          BIGINT PRIMARY KEY,
    username    VARCHAR(50) NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,
    nickname    VARCHAR(100),
    phone       VARCHAR(20),
    avatar      VARCHAR(2048),
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    bio         TEXT,
    settings    TEXT, -- JSON
    created_at  DATETIME,
    updated_at  DATETIME,
    created_by  BIGINT,
    updated_by  BIGINT
);

CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
```

创建带外键的表示例（orders -> users）：

```sql
CREATE TABLE orders (
    id          BIGINT PRIMARY KEY,
    order_no    VARCHAR(50) NOT NULL UNIQUE,
    user_id     BIGINT NOT NULL,
    amount      DECIMAL(12,2) NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    remark      TEXT,
    paid_at     DATETIME,
    created_at  DATETIME,
    updated_at  DATETIME,
    created_by  BIGINT,
    updated_by  BIGINT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at);
```

多对多中间表：

```sql
CREATE TABLE post_tags (
    post_id BIGINT NOT NULL,
    tag_id  BIGINT NOT NULL,
    created_at DATETIME,
    PRIMARY KEY (post_id, tag_id),
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);
```

## 4. 表结构同步（Schema Sync）

### 4.1 同步策略

```go
package metadata

import (
    "context"
    "fmt"
)

// syncToDBCreate 首次创建表
func (c *Collection) syncToDBCreate(ctx context.Context, tx gdb.TX) error {
    // 1. 生成CREATE TABLE语句
    createSQL := c.buildCreateTableSQL()

    // 2. 执行创建
    if _, err := tx.Ctx(ctx).Exec(createSQL); err != nil {
        return fmt.Errorf("创建表 %s 失败: %w", c.TableName, err)
    }

    // 3. 创建索引
    for _, idx := range c.indexes {
        if err := c.createIndex(ctx, tx, idx); err != nil {
            return err
        }
    }

    // 4. 如果是多对多关系，创建中间表
    for _, field := range c.fields {
        if btm, ok := field.(*BelongsToManyField); ok {
            if err := c.createThroughTable(ctx, tx, btm); err != nil {
                return err
            }
        }
    }

    return nil
}

// buildCreateTableSQL 生成建表语句
func (c *Collection) buildCreateTableSQL() string {
    var b strings.Builder
    b.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", c.quoteIdentifier(c.TableName)))

    var columns []string
    for _, field := range c.getOrderedFields() {
        col := c.buildColumnDefinition(field)
        columns = append(columns, "    "+col)
    }

    // 主键
    if c.FilterTargetKey != "" {
        columns = append(columns, fmt.Sprintf("    PRIMARY KEY (%s)", c.quoteIdentifier(c.FilterTargetKey)))
    }

    b.WriteString(strings.Join(columns, ",\n"))
    b.WriteString("\n)")

    return b.String()
}

func (c *Collection) buildColumnDefinition(field Field) string {
    name := c.quoteIdentifier(field.GetName())
    dataType := mapDataTypeToSQL(field.GetDataType(), field.GetOptions())

    var parts []string
    parts = append(parts, name, dataType)

    // 非空约束
    if bf, ok := field.(*BaseField); ok {
        if bf.IsRequired {
            parts = append(parts, "NOT NULL")
        } else {
            parts = append(parts, "NULL")
        }

        // 默认值
        if bf.DefaultValue != nil {
            parts = append(parts, "DEFAULT "+formatDefaultValue(bf.DefaultValue, field.GetDataType()))
        }

        // 唯一约束（单列唯一）
        if bf.IsUnique {
            parts = append(parts, "UNIQUE")
        }
    }

    return strings.Join(parts, " ")
}

func (c *Collection) createIndex(ctx context.Context, tx gdb.TX, idx *Index) error {
    var b strings.Builder
    if idx.Unique {
        b.WriteString("CREATE UNIQUE INDEX ")
    } else {
        b.WriteString("CREATE INDEX ")
    }
    idxName := idx.Name
    if idxName == "" {
        idxName = fmt.Sprintf("idx_%s_%s", c.TableName, strings.Join(idx.Fields, "_"))
    }
    b.WriteString(c.quoteIdentifier(idxName))
    b.WriteString(" ON ")
    b.WriteString(c.quoteIdentifier(c.TableName))
    b.WriteString(" (")
    quotedFields := make([]string, len(idx.Fields))
    for i, f := range idx.Fields {
        quotedFields[i] = c.quoteIdentifier(f)
    }
    b.WriteString(strings.Join(quotedFields, ", "))
    b.WriteString(")")

    _, err := tx.Ctx(ctx).Exec(b.String())
    return err
}
```

### 4.2 变更同步（添加字段、索引）

对于运行时添加字段/索引，使用 ALTER 语句：

```go
// AddFieldToDB 添加字段到数据库
func (c *Collection) AddFieldToDB(ctx context.Context, tx gdb.TX, field Field) error {
    colDef := c.buildColumnDefinition(field)
    sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s",
        c.quoteIdentifier(c.TableName), colDef)

    if _, err := tx.Ctx(ctx).Exec(sql); err != nil {
        return err
    }

    // 如果字段有UNIQUE约束，需要单独创建UNIQUE INDEX
    if bf, ok := field.(*BaseField); ok && bf.IsUnique {
        idxName := fmt.Sprintf("uk_%s_%s", c.TableName, field.GetName())
        idxSQL := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s)",
            c.quoteIdentifier(idxName),
            c.quoteIdentifier(c.TableName),
            c.quoteIdentifier(field.GetName()))
        if _, err := tx.Ctx(ctx).Exec(idxSQL); err != nil {
            return err
        }
    }

    return nil
}
```

> 注意：删除字段、修改字段类型属于高风险操作，生产环境建议谨慎处理。可以：
> - 提供 ALTER TABLE DROP COLUMN / ALTER COLUMN 功能
> - 或者仅支持新增字段，不支持删除/修改（最安全）
> - 修改字段类型时需要数据迁移逻辑

## 5. 查询构造（Filter 翻译到 SQL）

### 5.1 Filter 解析器

将 NocoBase 风格的 filter JSON 翻译为 SQL WHERE 子句：

```go
package metadata

import (
    "fmt"
    "strings"
)

// BuildWhereFromFilter 从Filter构建WHERE子句和参数
func (repo *GenericRepository) BuildWhereFromFilter(filter Filter, params *[]interface{}) (string, error) {
    if filter == nil || len(filter) == 0 {
        return "1=1", nil
    }
    return repo.parseFilterCondition(filter, params, false)
}

func (repo *GenericRepository) parseFilterCondition(cond map[string]interface{}, params *[]interface{}, isOr bool) (string, error) {
    var conditions []string

    for key, value := range cond {
        switch key {
        case "$and":
            subConds, err := repo.parseLogicGroup(value.([]interface{}), params, false)
            if err != nil {
                return "", err
            }
            conditions = append(conditions, "("+subConds+")")

        case "$or":
            subConds, err := repo.parseLogicGroup(value.([]interface{}), params, true)
            if err != nil {
                return "", err
            }
            conditions = append(conditions, "("+subConds+")")

        default:
            // 字段名条件
            if opVal, isOp := value.(map[string]interface{}); isOp {
                // 操作符形式: { field: { $gt: 18 } }
                colSQL, err := repo.parseFieldOperators(key, opVal, params)
                if err != nil {
                    return "", err
                }
                conditions = append(conditions, colSQL)
            } else {
                // 简单等于: { field: value }
                paramIdx := len(*params) + 1
                *params = append(*params, value)
                quotedCol := repo.quoteFieldName(key)
                conditions = append(conditions, fmt.Sprintf("%s = ?", quotedCol))
            }
        }
    }

    joinOp := " AND "
    if isOr {
        joinOp = " OR "
    }
    return strings.Join(conditions, joinOp), nil
}

func (repo *GenericRepository) parseLogicGroup(items []interface{}, params *[]interface{}, isOr bool) (string, error) {
    var subConds []string
    for _, item := range items {
        cond, ok := item.(map[string]interface{})
        if !ok {
            return "", fmt.Errorf("无效的条件格式")
        }
        s, err := repo.parseFilterCondition(cond, params, false)
        if err != nil {
            return "", err
        }
        subConds = append(subConds, s)
    }
    joinOp := " AND "
    if isOr {
        joinOp = " OR "
    }
    return strings.Join(subConds, joinOp), nil
}

func (repo *GenericRepository) parseFieldOperators(fieldName string, ops map[string]interface{}, params *[]interface{}) (string, error) {
    quotedCol := repo.quoteFieldName(fieldName)
    var conds []string

    for op, val := range ops {
        paramIdx := len(*params) + 1

        switch op {
        case "$eq":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s = ?", quotedCol))
        case "$ne":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s != ?", quotedCol))
        case "$gt":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s > ?", quotedCol))
        case "$gte":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s >= ?", quotedCol))
        case "$lt":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s < ?", quotedCol))
        case "$lte":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s <= ?", quotedCol))
        case "$in":
            placeholders := repo.buildInPlaceholders(val.([]interface{}), params)
            conds = append(conds, fmt.Sprintf("%s IN (%s)", quotedCol, placeholders))
        case "$notIn":
            placeholders := repo.buildInPlaceholders(val.([]interface{}), params)
            conds = append(conds, fmt.Sprintf("%s NOT IN (%s)", quotedCol, placeholders))
        case "$like":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s LIKE ?", quotedCol))
        case "$notLike":
            *params = append(*params, val)
            conds = append(conds, fmt.Sprintf("%s NOT LIKE ?", quotedCol))
        case "$startsWith":
            *params = append(*params, val.(string)+"%")
            conds = append(conds, fmt.Sprintf("%s LIKE ?", quotedCol))
        case "$endsWith":
            *params = append(*params, "%"+val.(string))
            conds = append(conds, fmt.Sprintf("%s LIKE ?", quotedCol))
        case "$includes":
            *params = append(*params, "%"+val.(string)+"%")
            conds = append(conds, fmt.Sprintf("%s LIKE ?", quotedCol))
        case "$is":
            if val == nil {
                conds = append(conds, fmt.Sprintf("%s IS NULL", quotedCol))
            } else {
                *params = append(*params, val)
                conds = append(conds, fmt.Sprintf("%s IS ?", quotedCol))
            }
        case "$not":
            if val == nil {
                conds = append(conds, fmt.Sprintf("%s IS NOT NULL", quotedCol))
            } else {
                *params = append(*params, val)
                conds = append(conds, fmt.Sprintf("%s IS NOT ?", quotedCol))
            }
        case "$between":
            arr := val.([]interface{})
            *params = append(*params, arr[0], arr[1])
            conds = append(conds, fmt.Sprintf("%s BETWEEN ? AND ?", quotedCol))
        case "$notBetween":
            arr := val.([]interface{})
            *params = append(*params, arr[0], arr[1])
            conds = append(conds, fmt.Sprintf("%s NOT BETWEEN ? AND ?", quotedCol))
        case "$empty":
            if val.(bool) {
                conds = append(conds, fmt.Sprintf("(%s IS NULL OR %s = '' OR %s = '[]')", quotedCol, quotedCol, quotedCol))
            } else {
                conds = append(conds, fmt.Sprintf("(%s IS NOT NULL AND %s != '' AND %s != '[]')", quotedCol, quotedCol, quotedCol))
            }
        default:
            // 检查自定义操作符
            if customOp, ok := repo.db.operators[op]; ok {
                sql := customOp(quotedCol, val, params)
                conds = append(conds, sql)
            } else {
                return "", fmt.Errorf("不支持的操作符: %s", op)
            }
        }
    }

    return strings.Join(conds, " AND "), nil
}

func (repo *GenericRepository) buildInPlaceholders(values []interface{}, params *[]interface{}) string {
    placeholders := make([]string, len(values))
    for i, v := range values {
        *params = append(*params, v)
        placeholders[i] = "?"
    }
    return strings.Join(placeholders, ", ")
}

// quoteFieldName 处理字段名（支持关联字段 'posts.title'）
func (repo *GenericRepository) quoteFieldName(fieldName string) string {
    if strings.Contains(fieldName, ".") {
        // 关联字段，需要JOIN，此处简化处理
        parts := strings.SplitN(fieldName, ".", 2)
        // TODO: 自动添加JOIN
        return repo.quoteIdentifier(parts[0]) + "." + repo.quoteIdentifier(parts[1])
    }
    return repo.quoteIdentifier(fieldName)
}

func (repo *GenericRepository) quoteIdentifier(ident string) string {
    return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
```

## 6. 事务支持

CobaltDB 支持事务，通过 GoFrame ORM 的事务 API 统一处理：

```go
// Transaction 在事务中执行
func (repo *GenericRepository) Transaction(ctx context.Context, fn func(tx gdb.TX) error) error {
    return repo.db.gdb.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
        return fn(tx)
    })
}
```

## 7. 性能优化建议

1. **索引策略**
   - 为常用查询字段（外键、状态、创建时间）创建索引
   - 为 Filter 常用的条件字段建索引
   - 复合索引注意顺序（等值查询字段在前，范围查询在后）

2. **WAL模式**
   - 开启 Write-Ahead Logging 提升并发读写性能
   - 配置合适的 cache_size

3. **简单分页模式**
   - 大表查询时跳过 COUNT(*) 使用 simplePagination
   - 使用基于 ID 的游标分页（WHERE id < lastId LIMIT n）提升深分页性能

4. **批量操作**
   - 批量创建/更新使用事务
   - 使用 INSERT 批量语法代替循环单条插入

5. **连接配置**
   - 嵌入式数据库通常不需要连接池
   - 设置合适的 busy_timeout 处理并发写入
