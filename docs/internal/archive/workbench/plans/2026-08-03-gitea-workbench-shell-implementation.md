# Gitea 研发管理壳实施计划

## 目标

按已确认的设计交付一套单域名研发代码平台：根路径和管理视图由 Workbench 提供，仓库、代码、提交详情、合并请求和设置继续由 Gitea 提供。WorkBench 使用 Vue 3 + TypeScript、Go 和独立 SQLite，并复用 Gitea OAuth2、用户、组织、仓库权限与代码数据。

## 实施原则

- 不修改 Gitea 核心源码或数据库。
- 不覆盖用户尚未提交的 `docs/operations/deployment.md` 改动。
- 先建立可测试的后端纵切，再实现界面和部署集成。
- 每个阶段必须有自动化验证，不能只依赖浏览器目测。
- 第一阶段严格限制在已确认设计范围，不加入 Issue、看板、通知和复杂报表。

## Task 1：建立 Workbench 工程骨架

**新增文件：**

- `workbench/go.mod`
- `workbench/cmd/workbench/main.go`
- `workbench/internal/config/config.go`
- `workbench/internal/config/config_test.go`
- `workbench/internal/webapp/*`
- `workbench/web/package.json`
- `workbench/web/tsconfig*.json`
- `workbench/web/vite.config.ts`
- `workbench/Dockerfile`

**步骤：**

1. 建立 Go 模块和最小 HTTP 服务。
2. 建立 Vue 3 + TypeScript + Vite 工程。
3. 生产构建使用 build tag 将 Vue `dist` 嵌入 Go 二进制；测试构建使用内置占位页。
4. 配置解析必须校验 URL、密钥长度、组织名、监听地址和数据库路径。

**验证：**

- `go test ./...`
- `npm run typecheck`
- `npm run build`

## Task 2：实现 SQLite 数据模型与迁移

**新增文件：**

- `workbench/internal/store/store.go`
- `workbench/internal/store/models.go`
- `workbench/internal/store/migrations.go`
- `workbench/internal/store/store_test.go`

**步骤：**

1. 创建项目档案、跟进人、审计、OAuth 会话、OAuth state、用户/仓库/提交缓存、用户仓库权限缓存和同步状态表。
2. SQLite 使用 WAL、外键和 busy timeout。
3. 实现项目档案读取、首次创建、乐观锁更新和审计事务。
4. 实现会话、state、缓存和过期清理。

**验证：**

- 空库自动迁移。
- 创建项目档案后可读取。
- 旧版本更新返回冲突且不覆盖新数据。
- 更新项目档案与审计记录在同一事务提交。

## Task 3：实现 Gitea OAuth2 与 API 客户端

**新增文件：**

- `workbench/internal/gitea/client.go`
- `workbench/internal/gitea/types.go`
- `workbench/internal/gitea/client_test.go`
- `workbench/internal/securebox/securebox.go`
- `workbench/internal/securebox/securebox_test.go`

**步骤：**

1. 使用 Authorization Code Grant、state 和 PKCE 登录。
2. 使用 AES-GCM 加密 OAuth access/refresh token。
3. 支持 token exchange、refresh 和 revoke。
4. 封装当前用户、可访问仓库、当前用户团队、组织成员和最近提交读取。
5. 处理分页、超时、限流和上游错误，禁止记录令牌和上游敏感正文。

**验证：**

- 使用 `httptest.Server` 验证授权 URL、token exchange、refresh、分页和 API 错误映射。
- 密文不可直接包含明文令牌，错误密钥无法解密。

## Task 4：实现会话、中间件与 Workbench API

**新增文件：**

- `workbench/internal/server/server.go`
- `workbench/internal/server/auth.go`
- `workbench/internal/server/api.go`
- `workbench/internal/server/sync.go`
- `workbench/internal/server/server_test.go`

**步骤：**

1. 实现登录、OAuth 回调、退出、服务端会话和 CSRF。
2. 实现 `/healthz`、`/readyz` 和 `/_workbench/api/me`。
3. 实现 dashboard、projects、project detail、project patch、people、activity 和手动同步 API。
4. 对每个写请求校验组织、团队和仓库可见性。
5. 对同一用户合并并发同步；60 秒内使用缓存。
6. Gitea 不可用时只返回该用户最后已知可见缓存，并标记 stale。

**验证：**

- 未登录返回 401。
- 无 CSRF、非编辑团队、无仓库访问权返回 403。
- 合法编辑返回新版本和审计记录。
- Gitea 失败时不泄漏其他用户缓存。

## Task 5：实现 Vue 管理界面

**新增文件：**

- `workbench/web/src/main.ts`
- `workbench/web/src/App.vue`
- `workbench/web/src/router.ts`
- `workbench/web/src/api/*`
- `workbench/web/src/components/*`
- `workbench/web/src/views/*`
- `workbench/web/src/styles/*`
- `workbench/web/src/**/*.test.ts`

**步骤：**

1. 实现工作台、项目列表、项目详情、成员列表和最近代码路由。
2. 工作台使用已确认布局：项目状态为主，最近代码为辅，成员动态独立。
3. 项目编辑使用可访问弹窗或抽屉，包含阶段、进度、负责人、跟进人、日期、风险和说明。
4. 实现加载、空、过期、无权限、同步失败、保存失败和 409 冲突状态。
5. 使用 Lucide Vue 图标；不使用 emoji 或自制 SVG 图标。
6. 遵循根目录 `design.md` 的 OKLCH 令牌、字体、响应式和 reduced-motion 规则。

**验证：**

- `npm run test`
- `npm run typecheck`
- `npm run build`
- 320、375、414、768、1024 和 1440 视口无页面级横向滚动；项目表仅自身可滚动。

## Task 6：统一 Gitea 主题与导航

**新增文件：**

- `gitea/custom/public/assets/css/theme-workbench.css`
- `gitea/custom/public/assets/fonts/*`
- 必要时的最小 `gitea/custom/templates/custom/*`

**修改文件：**

- `compose.yaml`
- `.env.example`

**步骤：**

1. 挂载 Gitea custom 目录。
2. 配置 `THEMES` 和 `DEFAULT_THEME`。
3. 只通过官方 custom 目录统一颜色、字体和导航链接，不复制整套上游模板。
4. 开启 Gitea OAuth2 Provider。

**验证：**

- Gitea 页面能加载自定义主题资源。
- 代码浏览、提交详情、登录和设置关键页面结构不被覆盖。

## Task 7：接入单域名路由与容器部署

**修改文件：**

- `deploy/nginx/workbench-routes.conf`
- `compose.yaml`
- `.env.example`
- `tests/check.sh`

**步骤：**

1. 增加固定版本 Workbench 构建参数、容器、数据挂载、健康检查和日志轮转。
2. 宿主机 Nginx 将 `/`、`/projects*`、`/people*`、`/activity*` 和 `/_workbench*` 交给 Workbench，其余交给 Gitea。
3. 明确精确路径与前缀路径，避免 `/project-other` 一类误匹配。
4. 保持 Gitea HTTPS clone 和 SSH 地址不变。

**验证：**

- `docker compose --env-file .env.example config`
- 自动测试 Nginx 路由匹配意图。
- Gitea 3000 和 Workbench 8081 仅发布到宿主机回环地址。

## Task 8：扩展备份、健康检查和运维文档

**修改文件：**

- `scripts/backup.sh`
- `scripts/health-check.sh`
- `tests/test-backup.sh`
- `tests/test-health-check.sh`
- `docs/operations/deployment.md`
- `docs/operations/upgrade.md`
- `docs/operations/recovery.md`
- `docs/operations/acceptance.md`
- `README.md`

**步骤：**

1. 备份同时停止并恢复 Workbench 与 Gitea，归档两个数据目录。
2. 健康检查覆盖根工作台、Gitea `/api/healthz`、Workbench readiness 和最近同步状态。
3. 部署文档增加 OAuth 应用注册、密钥生成、数据目录和首次登录。
4. 恢复和升级文档增加 Workbench 数据、迁移和主题兼容检查。
5. 保留并手工合并用户现有的 `docs/operations/deployment.md` 未提交修改。

**验证：**

- 现有 Shell 测试和新增场景全部通过。
- 备份失败清理仍能恢复两个服务。

## Task 9：端到端验收

**步骤：**

1. 使用模拟 Gitea 服务运行 Workbench 集成测试。
2. 构建 Vue 和 Go 生产产物。
3. 运行所有 Go、前端、Shell 和 Compose 检查。
4. 启动本地预览，用浏览器验证工作台、编辑表单、冲突、过期数据和移动端布局。
5. 对照设计说明的 12 项验收标准记录结果和仍需真实服务器完成的 OAuth/Git 检查。

**最终验证命令：**

```bash
cd workbench && go test ./...
cd workbench/web && npm ci && npm run test && npm run typecheck && npm run build
tests/check.sh
git diff --check
```
