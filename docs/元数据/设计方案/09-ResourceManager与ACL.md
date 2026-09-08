# ResourceManager 与 ACL MVP

## 范围

服务保留现有 `/api/c/:collection` REST 契约，在内部将请求映射为
`resource + action`。认证采用短期 JWT access token 与一次性轮换的
refresh token；授权默认拒绝。

本期不实现权限片段、多数据源 ACL、显式 deny、SSO 和权限配置界面。

## 请求链路

```text
HTTP -> Optional JWT -> REST Adapter -> ResourceManager -> ACL
     -> GenericRepository Guard -> MySQL
```

标准动作包括：

- `list/get/count/create/createMany/update/updateMany/destroy/destroyMany`
- `association:list/add/set/remove`
- `upload/file:get`
- Yaegi 自定义动作 `custom:<scriptName>`

## 策略

`acl_policies` 持久化 subject、resource、action、行过滤和读写字段。

- subject 支持 `role`、`loggedIn`、`public`
- resource/action 支持 `*` 通配
- 多角色行过滤取 OR，再与调用方 Filter、固定参数依次做 AND
- 多角色字段集合取并集；未授权字段不能参与读取、过滤、排序、关联加载或写入
- 无匹配策略即拒绝

行过滤仅支持现有 Filter JSON，并允许受控变量
`$currentUser.id`、`$currentUser.roles`，不执行任意 SQL 或 CEL。

## 安全边界

ACL 在 ResourceManager 做动作检查，并在 GenericRepository 再次执行。
二次检查覆盖关联递归访问和 Yaegi Repository，避免脚本或服务内调用绕过
HTTP 中间件。关联直接 SQL 在执行前验证源记录行权限，并尽量改用目标
Repository 执行目标表更新。

`X-Actor-Id` 不再作为身份来源。`created_by/updated_by` 从已验证 JWT
身份写入。

## 启动配置

必须通过环境变量提供至少 32 字节的 `ITCODEX_JWT_SECRET`。首次部署可设置：

```text
ITCODEX_BOOTSTRAP_ADMIN_USERNAME=admin
ITCODEX_BOOTSTRAP_ADMIN_PASSWORD=<strong-password>
```

仅当用户表为空时创建管理员；密码不会写入配置或日志。创建后应移除首次
引导密码环境变量。

## API

- `/api/auth/login|refresh|logout|me`
- `/api/meta/security/users`
- `/api/meta/security/roles`
- `/api/meta/security/users/{userId}/roles/{roleId}`
- `/api/meta/security/policies`

安全管理接口自身分别受 `auth.users:*`、`auth.roles:*`、
`acl.policies:*` 权限约束。
