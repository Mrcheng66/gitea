# 内部 Gitea 开发环境

## 锁定工具链

- Go 1.26.4
- Node.js 22.23.0
- pnpm 11.9.0
- Go 构建镜像 `docker.io/library/golang:1.26.4-alpine3.24@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648`

版本同时记录在根目录 `.tool-versions`。本地 Go 可通过 `GOTOOLCHAIN` 自动获取 1.26.4；CI 与发布构建必须使用相同版本。

## 基线验证

基线以未加入组织项目业务代码的 Gitea v1.27.1 文件树为准。执行命令：

```bash
pnpm install --frozen-lockfile
make test-frontend
make test-backend
GITEA_TEST_DATABASE=sqlite make test-integration-compile
make frontend
make backend
docker build --build-arg GITEA_VERSION=1.27.1-internal.0 -t gitea-internal:baseline .
```

### 2026-08-05 初始环境记录

- `go version`：`go1.26.4 darwin/arm64`，通过 Go 自动工具链下载成功。
- `node --version`：`v22.23.0`。
- `pnpm --version`：`11.9.0`。
- Docker CLI：`27.5.1`；通过 `docker buildx imagetools inspect` 解析到多架构索引 digest `sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648`。Docker daemon 未运行，因此镜像构建仍是环境阻断，不是源码基线失败。
- `pnpm install --frozen-lockfile`：通过。
- `make test-frontend`：通过，46 个测试文件、137 个测试。
- `make test-backend`：大部分通过，存在两个上游/本机环境基线失败：`services/gitdiff.TestGitDiffTreeRespectsDiffOrderFile/GlobalDiffOrderFile` 在 Git 2.42.0 下调用 `git config` 返回 129；`services/migrations.TestMigrateWhiteBlocklist` 受本机迁移来源白名单配置影响，拒绝 `gitlab.com`。两个失败在业务代码加入前已存在，暂不修改上游测试。
- `GITEA_TEST_DATABASE=sqlite make test-integration-compile`：通过。
- `make frontend`：通过。
- `make backend`：通过并生成 Gitea 二进制；Go 尝试更新用户级模块 stat cache 时有沙箱权限告警，但未阻断构建。

Docker daemon 可用后仍需补跑基线镜像构建。

## 常用验证

遵循根目录 `AGENTS.md`：

```bash
make fmt
make lint-go
make lint-js
make test-backend
make test-frontend
```

单个 Go 测试使用 `go test -run '^TestName$' ./modulepath/`；单个前端测试使用 `pnpm exec vitest <path-filter>`。
