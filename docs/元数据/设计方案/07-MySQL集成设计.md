# ITCodeX 元数据模块 - MySQL 集成设计

> 版本: v1.3
> 日期: 2026-09-06
> 数据库: MySQL 8（Docker 本机）
> 系统表走 GoFrame dao/do；业务表走 database/sql。见 [README · GoFrame 使用边界](./README.md#goframe-使用边界)。

## 1. MySQL 部署与连接策略

### 1.1 本机 Docker 部署

使用 Docker 在本机运行 MySQL 8，默认账号密码：

| 项 | 值 |
|----|----|
| 镜像 | `mysql:8.0` |
| 端口 | `3306` |
| Root 用户 | `root` |
| Root 密码 | `123456` |
| 业务库 | `itcodex` |

`docker-compose.yml`（项目根目录）：

```yaml
services:
  mysql:
    image: mysql:8.0
    container_name: itcodex-mysql
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: "123456"
      MYSQL_DATABASE: itcodex
    command:
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
      - --default-authentication-plugin=caching_sha2_password
    volumes:
      - itcodex_mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-p123456"]
      interval: 5s
      timeout: 5s
      retries: 20

volumes:
  itcodex_mysql_data:
```

启动：

```bash
docker compose up -d mysql
```

### 1.2 双通道访问

数据库层唯一选型为 **MySQL**。连接由 GoFrame `database.default` 建立（`g.DB()`），不要再手写 `gdb.New`。

| 对象 | 访问方式 |
|------|----------|
| 系统表 | `dao.Xxx.Ctx(ctx).Data(do.Xxx{...})`，禁止 `g.Map`；`created_at`/`updated_at` 由 ORM 维护 |
| 业务表 | `database/sql`（可 `g.DB().GetCore().DB`）+ 动态 DDL + Repository |

```yaml
database:
  logger:
    level: "all"
    stdout: true
  default:
    type: "mysql"
    link: "mysql:root:123456@tcp(127.0.0.1:3306)/itcodex?loc=Local&parseTime=true&charset=utf8mb4"
    debug: true
    createdAt: "created_at"
    updatedAt: "updated_at"
```

系统表结构稳定后：`gf gen dao`（只生成 `c_collections` / `c_fields` / `c_indexes` / `c_yaegi_scripts`）。生成文件禁止手改。

业务表 DSN 与上面 `link` 去掉 `mysql:` 前缀后相同，仅用于理解驱动参数，实现上不单独 `sql.Open` 第二套账号。

## 2. 数据类型映射

### 2.1 字段类型到 MySQL 类型映射

| 元数据字段类型 | MySQL 存储类型 | 说明 |
|---------------|----------------|------|
| string | VARCHAR(n) | 短字符串，默认 VARCHAR(255) |
| text | TEXT / LONGTEXT | 多行文本、富文本 |
| integer | INT | 32位整数 |
| bigInt | BIGINT | 64位整数（Snowflake ID 等） |
| float | FLOAT | 单精度浮点 |
| double | DOUBLE | 双精度浮点 |
| decimal | DECIMAL(p,s) | 精确小数（金额等） |
| boolean | TINYINT(1) / BOOLEAN | 布尔值 |
| date | DATE | 仅日期 |
| time | TIME | 仅时间 |
| datetime | DATETIME | 日期时间 |
| timestamp | TIMESTAMP / BIGINT | 时间戳；Unix 毫秒可用 BIGINT |
| json | JSON | JSON 数据 |
| jsonb | JSON | 兼容别名 |
| uuid | CHAR(36) | UUID 字符串 |
| nanoid | VARCHAR(21) | NanoID 字符串 |
| password | VARCHAR(255) | 密码哈希 |
| blob | BLOB / LONGBLOB | 二进制数据 |
| point | JSON | GeoJSON 点 |
| sort | INT | 排序字段 |

## 3. 数据库初始化

### 3.1 系统表创建

启动时自动创建以下系统表（如果不存在），或执行 `manifest/sql/metadata_system.sql`。标识符使用 MySQL 反引号；主键推荐 `BIGINT`（Snowflake）或 `BIGINT AUTO_INCREMENT`。这些表对应 `gf gen dao`，不要在业务代码里手写平行的 entity。

```sql
CREATE TABLE IF NOT EXISTS `c_collections` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL UNIQUE,
    `display_name` VARCHAR(255) NOT NULL DEFAULT '',
    `type` VARCHAR(50) NOT NULL DEFAULT 'general',
    `options` JSON NULL,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_fields` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `collection_name` VARCHAR(255) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `type` VARCHAR(50) NOT NULL,
    `display_name` VARCHAR(255) NOT NULL DEFAULT '',
    `is_required` TINYINT(1) NOT NULL DEFAULT 0,
    `is_unique` TINYINT(1) NOT NULL DEFAULT 0,
    `is_indexed` TINYINT(1) NOT NULL DEFAULT 0,
    `validation` JSON NULL,
    `options` JSON NULL,
    `sort` INT NOT NULL DEFAULT 0,
    `created_at` DATETIME NULL,
    UNIQUE KEY `uk_collection_field` (`collection_name`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_indexes` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `collection_name` VARCHAR(255) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `fields` JSON NOT NULL,
    `unique` TINYINT(1) NOT NULL DEFAULT 0,
    `options` JSON NULL,
    `created_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_yaegi_scripts` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `collection_name` VARCHAR(255) NULL,
    `name` VARCHAR(255) NOT NULL,
    `hook_point` VARCHAR(50) NOT NULL,
    `content` LONGTEXT NOT NULL,
    `api_path` VARCHAR(255) NULL,
    `http_method` VARCHAR(20) NULL,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `priority` INT NOT NULL DEFAULT 0,
    `options` JSON NULL,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 4. Schema 同步与 DDL

- 建表：`CREATE TABLE IF NOT EXISTS ... ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
- 增列：`ALTER TABLE ... ADD COLUMN ...`
- 删列：`ALTER TABLE ... DROP COLUMN ...`
- 索引：`CREATE [UNIQUE] INDEX ... ON ... (...)`
- 业务主键：默认 Snowflake 写入 `BIGINT`；也可使用 `AUTO_INCREMENT`

## 5. Filter 到 SQL 翻译

Filter 翻译为参数化 SQL，占位符使用 `?`。MySQL 支持 `LIMIT ? OFFSET ?`。

操作符范围以 [05-API接口设计 · Filter 分期](./05-API接口设计.md#filter-操作符分期) 为准，未列入的 NocoBase 算子不翻译。标识符使用反引号。

## 6. 事务

### 6.1 通道

- 系统表：优先 `dao.Xxx.Transaction` / `g.DB().Transaction`
- 业务表：`database/sql` 的 `BeginTx` / `Commit` / `Rollback`，`*sql.Tx` 放入 `context`
- 同一请求若同时写系统表和业务表，在外层开一个事务，业务 SQL 使用同一 `*sql.Tx`（或 `gdb.TX` 转底层连接），避免双连接提交

### 6.2 业务写入默认工作单元

`Repository.Create` / `Update` / `Destroy` 以及关联 add/set/remove **默认 Begin**，覆盖：

1. 主表语句
2. 关联表 / 中间表语句
3. Yaegi 钩子内经 `Collection(ctx, …)` 发出的业务表 SQL

实现必须从 `context` 取 Tx（见 [04 §4.3](./04-Yaegi二开设计.md#43-crud-工作单元事务)）。钩子不得另开连接写同一库。

嵌套 `Transaction()` 或钩子内再 `Create`：join 已有 Tx，MySQL 不做真正的嵌套事务。

DDL 不放在该工作单元内（MySQL DDL 隐式提交）。

### 6.3 隔离级别

默认使用连接/驱动默认隔离级别（InnoDB REPEATABLE READ）。不在 v1 暴露每请求隔离级别参数。

## 7. 性能建议

1. 使用连接池（如 MaxOpenConns=20, MaxIdleConns=5）
2. 合理索引（唯一约束、高频过滤字段）
3. 大表分页优先游标/主键翻页，避免深 OFFSET
4. JSON 字段按需建虚拟列索引
5. 生产环境勿使用 root；本机开发可用 root/123456
