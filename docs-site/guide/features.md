# 差异化能力

对标 RuoYi/go-admin/django-admin，本基座的差异化能力。

## 审计字段

嵌入 `model.Model` 的实体自动获得 `CreatedBy/UpdatedBy/DeletedBy`：

- **Create/Update**：`middleware.AuthRequired` 写入 userID 到 request context，GORM 回调（`pkg/audit`）自动注入，业务代码零感知
- **Delete（软删）**：GORM 软删 UPDATE 不经回调，故 DeletedBy 在 repository 层 `Updates` 双写 `deleted_at + deleted_by`

```go
type Foo struct {
    model.Model  // 内含 ID/CreatedAt/UpdatedAt/DeletedAt/CreatedBy/UpdatedBy/DeletedBy
    Name string
}
```

## 数据范围（DataScope）

`pkg/datascope` 按角色控制可见数据。`UserService.List` 受其约束。

| 角色 DataScope | 可见范围 |
|----------------|----------|
| `all` | 全部 |
| `dept` | 本人部门 |
| `dept_and_sub` | 本人部门 + 全部子孙（递归） |
| `self` | 仅本人 |

**规则**：多角色取最宽并集；任一角色权限含 `*`（超管）短路为 All；dept 角色但用户无部门 → 退化为 Self（杜绝越权）。

```go
// Resolver 由 userID 推导 Scope
scope, _ := resolver.Resolve(ctx, userID)
// Scope.Apply 翻译为 SQL WHERE
db = scope.Apply(db, "users.dept_id", "users.id")
```

## 登录防枚举

登录失败响应文案统一为"用户名或密码错误"，**不泄露用户是否存在/是否禁用**；真实失败原因（用户不存在/禁用/密码错/令牌签发失败）写入 `login_logs` 表，仅运维可见。

`AuthService.Login` 用 `defer + 命名返回 + 闭包` 在每个 return 出口记录日志，响应与日志解耦。

## 操作日志

`middleware.OperationLog` 拦截所有写操作（GET 跳过），异步记录到 `operation_logs`：
- 用户/IP/方法/路径/状态码/耗时/请求体/响应体摘要
- 经 `async.Runner`（GoroutineRunner）异步，不阻塞主请求

## 系统配置

`sys_config` + 内存缓存（RWMutex+map），运营改配置不发版：

```go
// 编程 API（其他模块零打库读配置）
val, _ := sysConfigSvc.GetValue(ctx, "login_captcha_enabled")
enabled := sysConfigSvc.GetBool("login_captcha_enabled", false)
```

后台 CRUD：`/api/system/config`，权限码 `config:system`。

## 代码生成器

```bash
go run ./cmd/scaffold -name Post -fields title:string,views:int,content:text
```

生成 `model/post.go` + `repository/post.go` + `service/post.go` + `handler/post.go` 四层代码，复用 `GenericRepository`。router/seed 按提示手动装配（YAGNI，不自动改路由）。
