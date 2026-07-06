# 贡献指南

感谢你愿意为 gin-vue-admin 贡献力量！完整规范见仓库根 [CONTRIBUTING.md](https://github.com/gin-vue-admin/gin-vue-admin/blob/main/CONTRIBUTING.md)。

## 开发流程

1. **拉分支**：从 `main` 切出 `feat/<topic>` 或 `fix/<topic>`
2. **写代码**：遵循分层（handler→service→repository→model），复用 `GenericRepository[T]`
3. **写测试**：新功能必须带测试，参考 [测试](./testing.md)
4. **本地校验**（提交前全绿）：
   ```bash
   make server-lint        # golangci-lint
   make server-test        # 单测
   make server-swag        # 改 handler 注解后重生成 docs 包
   make server-ci          # 一键 CI 复现
   ```
5. **提交**：遵循 [Conventional Commits](https://www.conventionalcommits.org/)：
   - `feat: 新增用户导出`
   - `fix(auth): 修复刷新令牌校验`
   - `refactor(repo): 泛型基类抽取`
6. **提 PR**：描述动机、影响、测试方式，关联 issue

## 代码规范

- **Go**：`gofmt` + `goimports` + `.golangci.yml` 规则集
- **架构**：依赖单向，repository 抽接口以便单测
- **错误**：用 `pkg/apperr`，统一 RFC 7807 ProblemDetail
- **响应**：`response.Success` / `apperr.Write`，不自定义 wrapper
- **原则**：YAGNI / DRY / KISS

## Swagger 注解

任何改动 handler 签名/路由/DTO 的 PR，必须同步更新注解并重新生成：

```bash
make server-swag
```

## 行为准则

友善、尊重、对事不对人。详见 [CODE_OF_CONDUCT](https://github.com/gin-vue-admin/gin-vue-admin)。
