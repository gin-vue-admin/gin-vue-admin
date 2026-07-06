# 贡献指南

感谢你愿意为 gin-vue-admin 贡献力量！本文档说明参与协作的方式与规范。

## 开发环境准备

```bash
git clone <repo-url>
cd gin-vue-admin

# 后端（Go 1.25+）
cd server
cp configs/config.example.yaml configs/config.yaml   # 首次：从模板创建本地配置（config.yaml 被 gitignore）
make -C .. server-swag                              # 首次生成 Swagger docs 包
go run ./cmd/api                                    # :8080

# 前端（Node 22+，推荐 nvm use 22）
cd web && pnpm install && pnpm dev                  # :5173
```

> 需要 MySQL。开发环境默认连 Docker 容器 `gva-mysql:13306`（root/root），见 `deploy/docker-compose.yml`。

## 开发工作流

1. **拉分支**：从 `main` 切出 `feat/<topic>` 或 `fix/<topic>`，不要直接提交 `main`。
2. **写代码**：遵循现有分层（`handler → service → repository → model`），复用泛型 `GenericRepository[T]`。新增模块参考 [`docs/new-module-guide.md`](docs/new-module-guide.md) 的 9 步 checklist。
3. **写测试**：repository/service 用 `testutil.NewTestDB(t)`（SQLite 隔离）；handler 用 `httptest`。新功能必须带测试，覆盖率不低于所在包均值。
4. **本地校验**（提交前必须全绿）：
   ```bash
   cd server
   golangci-lint run ./...
   go test ./...
   swag init -g cmd/api/main.go -o docs --parseInternal --parseDependency   # 改了 handler 注解后重生成
   ```
5. **提交**：遵循 [Conventional Commits](https://www.conventionalcommits.org/)：
   - `feat: 新增用户导出功能`
   - `fix(auth): 修复刷新令牌校验失效`
   - `refactor(repo): 泛型基类抽取分页`
   - `docs: 补充 swagger 注解`
6. **提 PR**：描述变更动机、影响范围、测试方式，关联相关 issue。

## 代码规范

- **Go**：`gofmt` + `goimports` + `.golangci.yml` 规则集；注释与现有代码同语言（中文）。
- **架构**：依赖单向（handler→service→repository→model），禁止反向依赖；repository 抽接口以便单测 mock。
- **错误**：用 `internal/pkg/apperr` 的 `NotFound/Conflict/Validation/...`，统一 RFC 7807 ProblemDetail。
- **响应**：成功 `response.Success`，失败 `apperr.Write`，不自定义 wrapper。
- **YAGNI/DRY/KISS**：仅实现所需；重复代码上浮到基类；避免过度设计。

## Swagger 注解

任何改动 handler 签名/路由/DTO 的 PR，必须同步更新注解并重新生成 docs 包：

```bash
make server-swag   # 或带完整 flag 见上
```

## 报告问题 / 提建议

- Bug 请用 Bug 报告模板，附复现步骤、预期/实际、日志（脱敏）。
- 新功能建议先开 issue 讨论，避免重复劳动或方向偏离。

## 行为准则

友善、尊重、对事不对人。维护者保留对违规内容拒绝或移除的权利。

## License

提交即表示你同意以 [MIT License](LICENSE) 发布你的贡献。
