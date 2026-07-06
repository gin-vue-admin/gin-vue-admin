# 测试

分层测试策略，SQLite 临时库隔离，无外部依赖即可跑。

## 覆盖范围

| 层 | 范围 | 工具 |
|----|------|------|
| repository | 真实 SQLite + GORM，覆盖 CRUD/过滤/数据范围 | `testutil.NewTestDB(t)` |
| service | SQLite 或 mock repo，覆盖业务编排 | `testutil.NewTestDB(t)` |
| handler | `httptest` 端到端，打通鉴权→业务→持久化 | `net/http/httptest` |
| pkg | 纯函数包，覆盖核心逻辑 | 标准测试 |

## 当前覆盖率

| 包 | 覆盖率 |
|----|--------|
| `pkg/datascope` | 100% |
| `pkg/csvutil` | 100% |
| `pkg/pagination` | 87.5% |
| `pkg/audit` | 65.7% |
| `repository` | 53.2% |
| `service` | 53.9% |
| `middleware` | 25.4% |
| `handler` | 6.3%（模式已建立，渐进补齐） |

## 跑测试

```bash
make server-test         # 全量
make server-cover        # 生成覆盖率报告 cover.html
```

## 写测试的模式

### Repository / Service

```go
func TestFooRepo_CRUD(t *testing.T) {
    db := testutil.NewTestDB(t)   // SQLite 临时库 + AutoMigrate
    repo := NewFooRepository(db)
    // ... 走真实 GORM，隔离 MySQL
}
```

### Handler 集成测试

```go
func newAuthTestServer(t *testing.T) (*gin.Engine, *service.AuthService) {
    db := testutil.NewTestDB(t)
    audit.Register(db)
    // ... 装配 repo→service→handler→gin engine
}

func TestLogin(t *testing.T) {
    r, _ := newAuthTestServer(t)
    w := doJSON(t, r, POST, "/api/auth/sessions",
        LoginRequest{Username: "admin", Password: "123456"})
    assert.Equal(t, 200, w.Code)
}
```

参考 `handler/auth_integration_test.go` 与 `handler/user_integration_test.go`。

## CI 门禁

GitHub Actions（`.github/workflows/ci.yml`）在每次 push/PR 跑：
golangci-lint → swagger 生成 → go vet → go test -race → build。本地复现：`make server-ci`。
