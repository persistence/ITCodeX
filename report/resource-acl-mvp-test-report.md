# ResourceManager 与 ACL MVP 测试报告

日期：2026-09-08

## 已通过

- `go test ./internal/service/auth ./internal/service/acl ./internal/service/resource ./internal/service/security ./internal/service/middleware`
- `server: go test ./... -run ^$`
- `client: go test ./... -run ^$`
- `server: go build ./...`
- `client: go build ./...`
- Cursor 代码诊断：无新增错误

覆盖内容：

- JWT 签发、校验、过期、refresh token 单次轮换与撤销
- ResourceManager 注册、通配匹配、中间件和动态 Repository 动作
- ACL 默认拒绝、角色/登录/公开主体、多角色行过滤、字段并集、固定参数
- Repository 行过滤和字段权限适配
- Yaegi 自定义动作 resource/action 解析
- 客户端登录、Bearer token 和当前用户支持

## 环境限制

本机 MySQL `127.0.0.1:3306` 未启动，Docker Desktop daemon 也不可用，因此：

- 原有 `internal/service/metadata` 数据库集成测试无法执行
- 真实服务 + MySQL 的 HTTP E2E 未执行
- 新增系统表的 `gf gen dao` 未运行；当前安全 Store 使用已定义的固定 SQL 表

已新增可在环境恢复后执行的安全 E2E：

```text
ITCODEX_JWT_SECRET=<至少32字节>
ITCODEX_BOOTSTRAP_ADMIN_PASSWORD=<强密码>
TEST_USERNAME=admin
TEST_PASSWORD=<同上>
```

启动服务后运行 `cd client && go test ./internal/tests -count=1 -v`。
