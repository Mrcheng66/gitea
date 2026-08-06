# Gitea 原生项目管理与定制化重构实施计划

日期：2026-08-05
状态：待执行确认
设计依据：`docs/superpowers/specs/2026-08-05-gitea-native-project-customization-design.md`
上游基线：Gitea `v1.27.1`，提交 `a62dfffbe7d3a454c2f2d3c4b2788ba432724a5f`

## 1. 目标与完成标准

本计划把当前 Gitea + Workbench 双应用重构为一个基于 Gitea v1.27.1 Fork 的模块化单体。项目、动态字段、配置发布、指标和历史成为 Gitea 原生能力；身份、组织、团队、仓库、Issue、合并请求、Release、Git 协议、会话和权限继续复用 Gitea。

完成标准：

- 仓库根目录是完整 Gitea v1.27.1 源码，官方历史和当前平台历史均可追溯。
- 生产只运行一个定制 Gitea 服务，不再运行 Workbench 或 OAuth 壳。
- 组织成员可读取项目；配置的编辑团队可维护项目；组织 Owners 可管理、发布和回滚配置。
- 项目独立于仓库，并可关联零到多个当前用户可访问的 Gitea 仓库。
- 动态字段、列表、筛选、排序和指标都来自已发布配置，不接受原始 SQL、脚本或公式。
- 旧 `project_profiles` 和 `project_audit_events` 可预检、导入、复核和幂等重跑；没有档案的仓库不自动变成项目。
- SQLite 数据迁移、权限裁剪、配置原子发布、乐观锁、API scope、Web 页面和切换回滚均有自动化测试。
- 登录、Web Git、SSH Git、仓库、Issue、PR、Release 和备份恢复没有回归。

## 2. 源码核对后的必要技术修正

设计文档中的模块名是逻辑边界。Gitea v1.27.1 已经使用以下名字表示 Issue/PR 看板：

- `models/project`
- `services/projects`
- `templates/projects`
- `routers/web/org/projects.go`
- 数据表 `project`
- Web 路由 `/{username}/-/projects` 与 `/{username}/{reponame}/projects`

新领域不能覆盖这些对象。实际实现统一使用 `orgproject` 技术命名空间：

| 逻辑职责 | 实际路径或物理名称 |
| --- | --- |
| 模型 | `models/orgproject` |
| 服务 | `services/orgproject` |
| Web 路由 | `routers/web/orgproject` |
| API 路由 | `routers/api/v1/orgproject` |
| 模板 | `templates/orgproject` |
| 前端 | `web_src/js/features/org-project.ts`、`web_src/js/components/orgproject` |
| 样式 | `web_src/css/features/org-project.css` |
| 数据表前缀 | `org_project_*` |
| 产品名称 | 中文“项目”，英文“Projects” |
| 上游看板名称 | 中文统一显示为“看板”，英文保持上游“Projects” |

首期产品路由固定为：

- 全局入口：`/projects`
- 组织项目：`/org/{org}/projects`
- 项目详情：`/org/{org}/projects/{slug}`
- 项目配置：`/org/{org}/settings/projects`
- 仓库反向关联：`/{username}/{reponame}/-/project-links`
- API：`/api/v1/orgs/{org}/projects`

这样不会与上游看板路由冲突，也不需要改变上游 Issue 看板的数据模型。

## 3. 实施原则

- 新长期分支以 Gitea v1.27.1 为第一父历史；当前 `main` 在切换完成前保持不动。
- 每个阶段形成可审查提交；数据模型、权限、配置发布、迁移工具和生产切换不得压成一个大提交。
- 先做领域模型和服务测试，再接 Web/API，最后接 UI 和运维切换。
- 所有写操作进入服务层事务；路由只负责身份上下文、绑定、错误映射和响应。
- 所有动态查询由受控 AST 编译为参数化 SQLite 查询；列名和别名只来自代码枚举，不拼接配置文本。
- 项目模块明确只支持 SQLite。启用模块而数据库不是 SQLite 时，进程启动和迁移命令必须失败并给出明确错误。
- 不修改 Git、仓库、Issue、PR、Release 的核心写路径。
- 不恢复旧 Workbench 应用壳、OAuth 会话、缓存表或 Nginx 分流。

## 4. 仓库历史与文件树接入方案

### Task 1：建立可回滚的 Gitea 基线分支

**只修改 Git 引用，不修改当前 `main` 文件树。**

1. 记录当前主线并创建永久保留分支：

   ```bash
   git status --short --branch
   git branch legacy/workbench-main main
   ```

2. 增加官方只读上游并获取固定标签：

   ```bash
   git remote add upstream https://github.com/go-gitea/gitea.git
   git fetch upstream tag v1.27.1 --no-tags
   git rev-parse 'v1.27.1^{commit}'
   ```

3. 校验提交必须等于：

   ```text
   a62dfffbe7d3a454c2f2d3c4b2788ba432724a5f
   ```

4. 从标签创建长期定制分支：

   ```bash
   git switch -c codex/gitea-native-v1.27.1 v1.27.1
   ```

5. 用 `ours` 策略连接旧平台历史。合并提交保留两边祖先，但文件树继续保持纯 Gitea：

   ```bash
   git merge --allow-unrelated-histories -s ours legacy/workbench-main -m "chore: connect legacy platform history"
   ```

**验证：**

```bash
git merge-base --is-ancestor v1.27.1 HEAD
git merge-base --is-ancestor legacy/workbench-main HEAD
test "$(git rev-list --first-parent --max-parents=0 HEAD)" = "$(git rev-list --max-parents=0 v1.27.1 | tail -n 1)"
git status --short
```

**回滚：** 删除尚未推送的 `codex/gitea-native-v1.27.1` 分支即可；`main`、`origin/main`、`legacy/workbench-main` 和标签不变。禁止强推覆盖 `main`。

### Task 2：迁移文档和运维资产，不恢复旧应用壳

**从 `legacy/workbench-main` 恢复并重排：**

- `docs/superpowers/specs/2026-08-05-gitea-native-project-customization-design.md` -> `docs/internal/org-project/design.md`
- 本实施计划 -> `docs/internal/org-project/implementation.md`
- 历史设计与计划 -> `docs/internal/archive/workbench/`
- `docs/operations/*` -> `ops/docs/*`
- `compose.yaml`、`.env.example` -> `ops/compose/`
- `deploy/*` -> `ops/deploy/`
- `scripts/*` -> `ops/scripts/`
- `tests/*.sh`、`tests/stubs/*` -> `ops/tests/`
- `examples/*` -> `ops/examples/`
- `design.md` -> `docs/internal/archive/workbench/ui-design.md`

**明确不恢复到新文件树：**

- `workbench/` 全部文件。
- `gitea/custom/templates/custom/extra_links.tmpl`。
- `gitea/custom/public/assets/css/theme-workbench.css`。
- `.hallmark/` 与 `.superpowers/brainstorm/` 运行产物。
- 旧根目录 `README.md`、`.dockerignore`、`.gitignore`，由 Gitea 上游版本作为基础再做最小合并。

旧文件没有丢失：它们永久存在于 `legacy/workbench-main` 和合并历史中。切换验收前，不删除该分支。

**新增文件：**

- `docs/internal/org-project/README.md`
- `docs/internal/upstream-policy.md`
- `ops/README.md`
- `ops/migration/README.md`

`docs/internal/upstream-policy.md` 记录：固定 v1.27.1、仅回移高危安全和必要兼容补丁、每个补丁的来源提交、风险、测试和回滚，以及本地迁移编号冲突处理。

**验证：**

```bash
test -f docs/internal/org-project/design.md
test -f docs/internal/org-project/implementation.md
test -f ops/compose/compose.yaml
test ! -e workbench
test ! -e gitea/custom/templates/custom/extra_links.tmpl
git log --all -- workbench/internal/store/migrations.go
git diff --check
```

## 5. 工具链与上游基线

### Task 3：锁定构建环境并建立零改动基线

源码要求 Go `1.26.4`；当前开发机为 Go `1.25.4`。Node `22.23.0` 和 pnpm `11.9.0` 满足 `package.json` 的 Node `>=22.18.0`、pnpm `>=11.0.0` 要求。

构建镜像把上游 `Dockerfile` 的 Go 基础镜像从浮动的 `golang:1.26-alpine3.24` 固定到 `golang:1.26.4-alpine3.24`，并在首次执行时解析和记录镜像 digest。开发机若安装本地 Go，也必须锁定 `1.26.4`，不得使用旧工具链或浮动 minor tag 绕过版本要求。

**新增文件：**

- `.tool-versions`，固定 `golang 1.26.4`、`nodejs 22.23.0`、`pnpm 11.9.0`
- `docs/internal/development.md`

**修改文件：**

- `Dockerfile`
- `Dockerfile.rootless`

**基线命令：** 以下命令必须在锁定的 Go 1.26.4 环境或等价的 pinned build container 中执行。

```bash
pnpm install --frozen-lockfile
make test-frontend
make test-backend
GITEA_TEST_DATABASE=sqlite make test-integration-compile
make frontend
make backend
docker build --build-arg GITEA_VERSION=1.27.1-internal.0 -t gitea-internal:baseline .
```

首次基线允许记录上游或环境已有失败，但必须在 `docs/internal/development.md` 写明命令、日期、日志摘要和是否阻断。定制代码不能掩盖基线失败。

## 6. 模块开关、命名和数据库迁移

### Task 4：增加 orgproject 设置与 SQLite 强制校验

**新增文件：**

- `modules/setting/orgproject.go`
- `modules/setting/orgproject_test.go`

**修改文件：**

- `modules/setting/setting.go`
- `cmd/helper.go`
- `routers/install/install.go`
- `custom/conf/app.example.ini`

**配置：**

```ini
[org_project]
ENABLED = true
DEFAULT_PAGE_SIZE = 25
MAX_PAGE_SIZE = 100
MAX_FIELDS = 64
MAX_ENUM_OPTIONS = 100
MAX_REPOSITORIES_PER_PROJECT = 20
```

`loadOrgProjectFrom` 在 `LoadSettings` 和 `LoadSettingsForInstall` 已加载数据库设置后执行；`cmd/helper.go:initDB` 也显式加载并验证，确保 `gitea migrate` 和 `gitea admin org-project` 不会绕过限制。安装表单选择非 SQLite 且模块启用时返回明确错误。`ENABLED=true` 且 `DB_TYPE` 不是 `sqlite3` 时，任何建库、迁移、Web 启动或导入命令都必须在同步表之前失败。测试覆盖默认值、边界值、禁用模块、安装路径和非 SQLite 拒绝。

**验证：**

```bash
go test ./modules/setting -run OrgProject
```

### Task 5：创建 orgproject 模型和 Gitea 迁移 343

**新增文件：**

- `models/orgproject/project.go`
- `models/orgproject/repository.go`
- `models/orgproject/field_value.go`
- `models/orgproject/config.go`
- `models/orgproject/editor_team.go`
- `models/orgproject/change_log.go`
- `models/orgproject/errors.go`
- `models/orgproject/main_test.go`
- `models/orgproject/project_test.go`
- `models/orgproject/config_test.go`
- `models/fixtures/org_project.yml`
- `models/fixtures/org_project_repository.yml`
- `models/fixtures/org_project_field_value.yml`
- `models/fixtures/org_project_config_version.yml`
- `models/fixtures/org_project_config_pointer.yml`
- `models/fixtures/org_project_editor_team.yml`
- `models/fixtures/org_project_change_log.yml`
- `models/migrations/v1_27/v343.go`
- `models/migrations/v1_27/v343_test.go`

**修改文件：**

- `models/migrations/migrations.go`
- `models/migrations/migrations_test.go`

**物理数据表：**

1. `org_project`
   - `id`、`owner_id`、`slug`、`name`、`description`、`lifecycle`、`version`
   - `created_by`、`created_unix`、`updated_unix`
   - 唯一键 `(owner_id, slug)`，索引 `(owner_id, lifecycle, updated_unix)`

2. `org_project_repository`
   - `id`、`project_id`、`repository_id`、`role`、`created_by`、`created_unix`
   - 唯一键 `(project_id, repository_id)`
   - 一个项目最多一个 `primary`；SQLite 使用部分唯一索引实现

3. `org_project_field_value`
   - `id`、`project_id`、`field_key`、`value_text`、`value_number`、`value_time`、`value_bool`、`value_user_id`、`value_json`
   - 唯一键 `(project_id, field_key)`
   - 按 `field_key` 与各类型值列建立组合索引
   - 服务层保证恰好一个类型列有值；迁移测试验证 SQLite 约束和索引

4. `org_project_config_version`
   - `id`、`owner_id`、`version`、`state`、`payload`
   - `created_by`、`created_unix`、`published_by`、`published_unix`
   - 唯一键 `(owner_id, version)`

5. `org_project_config_pointer`
   - `owner_id` 主键、`draft_version_id`、`published_version_id`、`version`

6. `org_project_editor_team`
   - `owner_id`、`team_id`、`created_by`、`created_unix`
   - 唯一键 `(owner_id, team_id)`
   - 团队必须属于同一组织；权限查询 JOIN 原生 Team 并校验 `org_id`，已删除团队的悬空关系自动失效；Owner 下次保存设置时清理悬空行，不修改 Gitea 团队删除核心流程

7. `org_project_change_log`
   - `id`、`project_id`、`actor_id`、`request_id`、`changed_fields`、`before_value`、`after_value`、`source`、`created_unix`
   - `changed_fields` 是稳定字段键数组；前后值是按字段键组织的受控 JSON 对象，一次请求只生成一条事件
   - 唯一键 `request_id`，索引 `(project_id, created_unix, id)`

迁移编号 `343` 是本 Fork 的第一个本地迁移。后续安全补丁若从上游带来迁移，必须在回移记录中重新编号到本地序列末尾，不能覆盖本地 343。

Gitea 对全新数据库会把 migration version 直接设为最新，再通过 `SyncAllTables` 建表，因此 v343 不会在 fresh install 上逐条执行。`org_project_repository` 的 partial unique index 和其他 SQLite 专用对象必须同时通过 `db.RegisterModel(..., initFunc)` 的幂等初始化函数创建。测试分别覆盖“旧库执行 v343”和“空库 SyncAllTables + initFunc”，并断言两种路径得到相同 schema 与索引。

**验证：**

```bash
go test ./models/orgproject
GITEA_TEST_DATABASE=sqlite make migrations.individual.test#v1_27
GITEA_TEST_DATABASE=sqlite make test-migration
```

## 7. 配置领域与发布事务

### Task 6：定义配置 DTO、默认配置和规范化规则

**新增文件：**

- `services/orgproject/config/types.go`
- `services/orgproject/config/defaults.go`
- `services/orgproject/config/normalize.go`
- `services/orgproject/config/validate.go`
- `services/orgproject/config/validate_test.go`

默认字段固定键：

- `stage`
- `progress`
- `owner`
- `followers`
- `start_date`
- `target_date`
- `risk`
- `summary`

默认枚举值与旧 Workbench 一致，以保证导入无损。配置 payload 使用有版本号的 JSON schema：

```json
{
  "schema_version": 1,
  "fields": [],
  "list_view": {},
  "filters": [],
  "metrics": []
}
```

规范化必须保证稳定排序、去重和一致 JSON 输出，便于 diff、审计和测试。验证覆盖字段键、类型、默认值、必填、归档、枚举、列表引用、筛选器、排序和指标聚合边界。

**验证：**

```bash
go test ./services/orgproject/config
```

### Task 7：实现草稿、发布、历史和回滚服务

**新增文件：**

- `services/orgproject/config/service.go`
- `services/orgproject/config/publish.go`
- `services/orgproject/config/rollback.go`
- `services/orgproject/config/service_test.go`

**事务规则：**

- 首次访问由 Owner 显式初始化默认草稿，不在普通读取请求中隐式写库。
- 保存草稿创建新的 immutable draft snapshot，并使用 pointer `version` 乐观锁切换 `draft_version_id`；冲突返回领域错误 `ErrConfigConflict`。
- 发布先验证草稿，再创建不可变 published 快照并原子切换 pointer。
- 回滚不是修改旧快照，而是复制目标 published payload 形成新 published 版本。
- 成员运行页只加载 `published_version_id`，永不回退读取 draft。
- 已发布字段删除只归档定义；已有字段值保留。

**测试：**

- 并发保存只有一个成功。
- 发布验证失败不切换 pointer。
- 发布事务故障后仍读取旧版本。
- 回滚生成新版本且历史版本不变。
- 无 published 配置时返回可识别的未初始化状态。

## 8. 权限与仓库可见性

### Task 8：实现组织项目权限策略

**新增文件：**

- `services/orgproject/permission.go`
- `services/orgproject/permission_test.go`
- `routers/web/orgproject/context.go`
- `routers/api/v1/orgproject/context.go`

**权限规则：**

- `CanRead`：当前用户是组织成员或站点管理员。
- `CanEdit`：组织 Owner、站点管理员，或属于 `org_project_editor_team` 中的团队。
- `CanConfigure`：仅组织 Owner或站点管理员。
- 所有项目读取先检查组织成员身份；不因项目字段为“公开”而允许组织外读取。
- 项目不存在与项目所属组织不可见统一映射为 404。
- 资源可见但操作角色不足映射为 403。

**仓库规则：**

- 使用 `models/perm/access.GetIndividualUserRepoPermission` 检查当前用户。
- 详情、列表 DTO 和活动汇总只装载具有 `unit.TypeCode` 读取权限的关联仓库。
- 添加关联时，操作者必须能读取目标仓库，且仓库 Owner 必须是项目所属组织。
- API、Web 和审计日志均不包含被裁剪仓库的 ID、名称或计数。

**测试矩阵：** Owner、编辑团队成员、普通成员、非成员、站点管理员；公开仓库、私有可见仓库、私有不可见仓库、跨组织仓库、已删除仓库。

## 9. 项目写入、查询和指标

### Task 9：实现项目命令服务与审计

**新增文件：**

- `services/orgproject/project/create.go`
- `services/orgproject/project/update.go`
- `services/orgproject/project/archive.go`
- `services/orgproject/project/repository.go`
- `services/orgproject/project/values.go`
- `services/orgproject/project/audit.go`
- `services/orgproject/project/service_test.go`

**行为：**

- 创建项目写入固定字段、动态值、仓库关系和 change log，全部在同一事务。
- 更新要求客户端提供项目 `version`；成功后原子递增。
- slug 必须满足 Gitea 命名约束，并保留 `new`、`config`、`settings`、`dashboard`、`history`，避免与静态 Web 路由冲突。
- 动态输入只接受已发布配置中的活动字段。
- 成员字段只接受当前组织成员 ID；多选和成员数组排序去重后再序列化。
- `request_id` 对 Web 和 API 写入都必填，用于幂等审计。
- 归档不删除项目、字段值、仓库关系或历史。
- 删除仓库只保留项目并清理或标记失效关联，不删除项目。

**错误：** `ErrNotFound`、`ErrForbidden`、`ErrConflict`、`ValidationErrors`、`ErrRepositoryNotVisible`，由路由统一映射为 404/403/409/422。

### Task 10：实现 SQLite 类型化查询编译器

**新增文件：**

- `services/orgproject/query/ast.go`
- `services/orgproject/query/compiler.go`
- `services/orgproject/query/list.go`
- `services/orgproject/query/metric.go`
- `services/orgproject/query/compiler_test.go`
- `services/orgproject/query/list_test.go`
- `services/orgproject/query/metric_test.go`

**编译规则：**

- 配置先被解析成内部 AST；任何未知字段、操作符、类型或聚合直接拒绝。
- 动态字段 JOIN 别名由代码按序号生成，例如 `fv_0`，不使用 `field_key` 作为 SQL 标识符。
- `field_key`、筛选值、范围值和分页值全部使用绑定参数。
- 排序列只来自固定列枚举或字段类型到值列的映射。
- 多选和成员数组使用 SQLite `json_each`，并增加启动自检和单元测试确认 JSON1 可用。
- 所有列表增加稳定的 `org_project.id` 次级排序，避免翻页漂移。
- 默认限制页大小 25，最大 100；指标查询限制字段、分组数量和返回桶数量。

**安全测试：** 在字段键、枚举值、筛选值、排序键中输入 SQL 片段，断言 SQL 文本不包含输入且数据库结构未改变。

## 10. API 与 Token Scope

### Task 11：注册 `read:project` 与 `write:project`

**修改文件：**

- `models/auth/access_token_scope.go`
- `models/auth/access_token_scope_test.go`
- `routers/api/v1/api.go`
- `options/locale/locale_en-US.json`
- `options/locale/locale_zh-CN.json`

**变更：**

- 在 scope category 枚举末尾追加 `AccessTokenScopeCategoryProject`，不得插入中间改变既有枚举值。
- 增加 read/write 常量、bitmap、`allAccessTokenScopes`、`allAccessTokenScopeBits` 和 category 映射。
- `write:project` 隐含 `read:project`；`all` 包含项目写权限。
- public-only token 明确拒绝项目 API，因为项目仅对组织成员开放。
- scope 只控制 API 能力，不能绕过组织成员、编辑团队或 Owner 检查。

**验证：**

```bash
go test ./models/auth -run AccessTokenScope
```

### Task 12：实现项目与配置 API

**新增文件：**

- `modules/structs/orgproject.go`
- `routers/api/v1/orgproject/project.go`
- `routers/api/v1/orgproject/config.go`
- `routers/api/v1/orgproject/repository.go`
- `routers/api/v1/orgproject/project_test.go`
- `routers/api/v1/orgproject/config_test.go`
- `tests/integration/api_org_project_test.go`

**修改文件：**

- `routers/api/v1/api.go`
- `templates/swagger/v1_json.tmpl`，由生成命令更新
- `templates/swagger/v1_openapi3_json.tmpl`，由生成命令更新

**端点：**

- `GET/POST /api/v1/orgs/{org}/projects`
- `GET/PATCH /api/v1/orgs/{org}/projects/{slug}`
- `POST/DELETE /api/v1/orgs/{org}/projects/{slug}/repositories/{repo_id}`
- `GET /api/v1/orgs/{org}/projects/{slug}/history`
- `GET/PUT /api/v1/orgs/{org}/project-config/draft`
- `POST /api/v1/orgs/{org}/project-config/validate`
- `POST /api/v1/orgs/{org}/project-config/publish`
- `GET /api/v1/orgs/{org}/project-config/versions`
- `POST /api/v1/orgs/{org}/project-config/rollback/{version}`

列表筛选使用重复 query 参数或明确 JSON body，不接受自由 SQL。写接口通过 Gitea API binder 校验请求大小和类型。

**验证：**

```bash
make generate-swagger
make swagger-validate
GITEA_TEST_DATABASE=sqlite make test-integration#OrgProjectAPI
```

生成文件提交后再运行 `make swagger-check`，确认二次生成没有差异。

## 11. 原生 Web 页面与前端交互

### Task 13：实现服务器端项目页面和路由

**新增文件：**

- `routers/web/orgproject/landing.go`
- `routers/web/orgproject/project.go`
- `routers/web/orgproject/config.go`
- `routers/web/orgproject/repository.go`
- `routers/web/orgproject/project_test.go`
- `routers/web/orgproject/config_test.go`
- `services/forms/orgproject.go`
- `templates/orgproject/landing.tmpl`
- `templates/orgproject/dashboard.tmpl`
- `templates/orgproject/list.tmpl`
- `templates/orgproject/new.tmpl`
- `templates/orgproject/view.tmpl`
- `templates/orgproject/settings.tmpl`
- `templates/orgproject/history.tmpl`
- `templates/orgproject/shared/*.tmpl`

**修改文件：**

- `routers/web/web.go`
- `templates/base/head_navbar.tmpl`
- `templates/org/menu.tmpl`
- `templates/org/settings/navbar.tmpl`
- `templates/repo/header.tmpl`
- `options/locale/locale_en-US.json`
- `options/locale/locale_zh-CN.json`

**路由行为：**

- `/projects`：一个可访问组织时直接跳转；多个组织时展示选择页；无组织时展示空状态。
- `/org/{org}/projects`：组织成员可读，展示工作台和项目列表子导航。
- `/org/{org}/settings/projects`：复用 `OrgAssignment(RequireOwner: true)`。
- 仓库反向关联页必须先通过原生 repo assignment 和 code read 权限。
- Web POST 使用 Gitea session、CSRF、form binder 和 flash/JSON redirect 习惯。
- 顶部导航按“项目、Issue、合并请求、探索”排列；Logo 继续作为原生工作台入口，不复制第二个 Dashboard shell。

**命名处理：**

- 新增 `org_project.*` locale key。
- 将中文 `repo.projects`、`user.projects` 及相关上游看板文案从泛称“项目”调整为“看板”或“协作看板”；英文不改上游含义。
- 不删除或重写上游看板功能。

### Task 14：实现动态表单、列表、指标和配置编辑器

**新增文件：**

- `web_src/js/features/org-project.ts`
- `web_src/js/components/orgproject/ProjectForm.vue`
- `web_src/js/components/orgproject/ProjectList.vue`
- `web_src/js/components/orgproject/ProjectDashboard.vue`
- `web_src/js/components/orgproject/ProjectHistory.vue`
- `web_src/js/components/orgproject/ConfigEditor.vue`
- `web_src/js/components/orgproject/EditorTeamsEditor.vue`
- `web_src/js/components/orgproject/FieldEditor.vue`
- `web_src/js/components/orgproject/ViewEditor.vue`
- `web_src/js/components/orgproject/MetricEditor.vue`
- `web_src/js/components/orgproject/*.test.ts`
- `web_src/css/features/org-project.css`

**修改文件：**

- `web_src/js/index.ts`
- `web_src/css/index.css`

**界面边界：**

- 使用 Gitea 原生页面框架、菜单、表单、按钮、消息、表格和 Octicon/Lucide 已有图标体系。
- 不创建第二套全屏应用壳，不加独立登录态或前端路由器。
- 服务端模板提供初始数据和 CSRF；Vue 只挂载复杂交互区域。
- 字段类型使用相应原生控件；枚举为下拉/多选，布尔为 checkbox，数值为 input/stepper，日期使用现有日期控件。
- 配置编辑器支持新增、归档、排序、校验、保存草稿、发布、版本对比和回滚确认。
- 编辑团队选择器只列出当前组织团队并保存原生 Team ID；团队改名不丢失权限，已删除团队显示为失效并在保存时清理。
- 发布按钮只对 Owner 显示，服务端仍必须二次鉴权。
- 320、375、414、768、1024、1440 视口无页面级横向滚动；宽表只在表格容器内滚动。
- `prefers-reduced-motion` 下关闭非必要动效。

**验证：**

```bash
make test-frontend
make lint-js
make lint-css
make lint-templates
make frontend
```

## 12. 代码活动与仓库关联

### Task 15：实现权限裁剪后的代码活动汇总

**新增文件：**

- `services/orgproject/activity/service.go`
- `services/orgproject/activity/service_test.go`
- `templates/orgproject/activity.tmpl`
- `web_src/js/components/orgproject/ProjectActivity.vue`

首期活动只读取 Gitea 原生数据，不复制提交缓存：

- 关联仓库最近提交。
- 打开/合并的 Pull Request 数量。
- Release 数量和最近发布时间。

每次查询先对关联仓库做当前用户权限裁剪。活动查询有明确时间窗口、每仓库限制和总量限制，避免项目关联仓库增加后形成无界查询。无权仓库对计数和“隐藏数量”均不产生侧信道。

## 13. Workbench 一次性导入

### Task 16：实现预检和幂等导入命令

**新增文件：**

- `cmd/admin_org_project.go`
- `cmd/admin_org_project_test.go`
- `services/orgproject/migration/source.go`
- `services/orgproject/migration/preflight.go`
- `services/orgproject/migration/import.go`
- `services/orgproject/migration/report.go`
- `services/orgproject/migration/migration_test.go`
- `ops/migration/testdata/workbench-v1.sql`

**修改文件：**

- `cmd/admin.go`

**命令：**

```bash
gitea admin org-project preflight-workbench \
  --database /backup/workbench.db \
  --organization engineering \
  --actor migration-admin \
  --editor-teams Owners,Developers \
  --report /backup/org-project-preflight.json

gitea admin org-project import-workbench \
  --database /backup/workbench.db \
  --organization engineering \
  --actor migration-admin \
  --editor-teams Owners,Developers \
  --report /backup/org-project-import.json
```

**预检输出：**

- 档案、followers、audit 数量。
- repo ID 是否存在、是否属于目标组织。
- owner/follower/actor 用户是否存在且属于目标组织。
- slug 冲突、重复 request ID、非法日期、非法枚举和损坏 JSON。
- 将导入、跳过、阻断的逐项原因。

**导入规则：**

- 首先为目标组织创建并发布默认配置。
- `--actor` 必须解析为目标组织 Owner，用于默认配置发布和导入操作审计，禁止静默选择任意管理员。
- 每条 `project_profiles` 创建一个 `org_project`，旧 `repo_id` 成为 primary 关联。
- 项目名称和描述从 Gitea 仓库读取；slug 从仓库名规范化并处理冲突。
- 旧字段映射到默认动态字段。
- `project_followers` 写入 `followers`。
- 审计事件拆分为 `legacy-import` change log；保留原 request ID 和时间。
- 使用确定性 request ID 作为导入标记：项目档案为 `legacy-import:profile:{repo_id}`，历史事件为 `legacy-import:audit:{legacy_id}`。重跑先按 request ID 解析已导入 project，再只补齐缺失历史；不能重复项目、关联或审计。
- 缺失仓库、跨组织仓库、缺失关键用户为阻断；非关键历史 actor 缺失可映射为系统导入 actor，但必须在报告中列出。
- 不读取或导入 `oauth_sessions`、`oauth_states`、任何 Gitea cache 或 `sync_state`。

**验证：**

```bash
go test ./services/orgproject/migration ./cmd -run OrgProject
```

对测试数据库连续执行两次 import，第二次必须是零新增并返回成功报告。

## 14. 运维、镜像和生产切换

### Task 17：把部署改为单一 Fork 镜像

**修改文件：**

- `Dockerfile`，仅在需要增加内部版本标签或构建元数据时做最小修改
- `ops/compose/compose.yaml`
- `ops/compose/.env.example`
- `ops/deploy/nginx/gitea.conf`
- `ops/scripts/backup.sh`
- `ops/scripts/health-check.sh`
- `ops/tests/check.sh`
- `ops/tests/test-backup.sh`
- `ops/tests/test-health-check.sh`
- `ops/tests/test-nginx-routing.sh`
- `ops/docs/deployment.md`
- `ops/docs/upgrade.md`
- `ops/docs/recovery.md`
- `ops/docs/acceptance.md`
- 根 `README.md`

**部署变更：**

- Compose 只保留 `gitea` 服务，镜像由当前源码构建并使用固定内部版本，例如 `1.27.1-internal.1`。
- 删除 Workbench 8081 端口、数据卷、env file、依赖和健康检查。
- Nginx 所有 Web 路径回到 Gitea 3000，不再存在 `/projects`、`/_workbench` 等分流。
- 数据库保持 `/data/gitea/gitea.db`，显式设置 `GITEA__org_project__ENABLED=true`。
- 备份不再停止/归档 Workbench 运行数据，但在回滚窗口内单独保留旧 Workbench 数据库只读备份。
- 健康检查增加项目模块 readiness：数据库类型、published config 指针完整性和 JSON1 自检。
- 镜像元数据记录 Gitea tag、上游 commit、内部 commit 和构建时间。

**验证：**

```bash
docker compose --env-file ops/compose/.env.example -f ops/compose/compose.yaml config
ops/tests/check.sh
docker build --build-arg GITEA_VERSION=1.27.1-internal.1 -t gitea-internal:1.27.1-internal.1 .
```

### Task 18：端到端验收与维护窗口切换

**新增文件：**

- `tests/e2e/org-project.test.ts`
- `tests/integration/org_project_web_test.go`
- `ops/migration/cutover-checklist.md`
- `ops/migration/rollback-checklist.md`

**预生产演练：**

1. 从生产备份恢复 Gitea 数据、仓库、附件、配置和 Workbench SQLite 到隔离环境。
2. 构建定制镜像并运行 Gitea migration 343。
3. 执行 Workbench preflight，阻断项必须为零。
4. 执行 import，两次运行验证幂等。
5. 核对项目、仓库关联、followers、字段值、审计和 editor team 数量。
6. 使用 Owner、编辑团队成员、普通成员和非成员账号运行权限矩阵。
7. 验证不可见仓库不出现在页面、API、指标、活动和错误正文。
8. 验证配置草稿不可见、发布原子、并发冲突和回滚。
9. 验证 Git HTTPS clone/push、SSH clone/push、仓库浏览、Issue、PR 和 Release。
10. 验证备份、恢复和旧系统回滚路径。

**维护窗口：**

1. 宣布冻结写入，停止旧 Gitea 与 Workbench。
2. 完整备份 Gitea 数据目录、配置、镜像版本和 Workbench 数据库。
3. 在备份副本上再次执行 preflight 并保存报告。
4. 部署定制 Gitea，执行原生迁移与一次性 import。
5. 启动单一 Gitea，更新 Nginx 并完成快速验收。
6. 验收通过后开放写入，记录切换时间和 commit。
7. 在约定回滚窗口内保留旧镜像、旧 Compose、Workbench 数据库和 Nginx 配置。

**回滚触发条件：** 登录失败、Git 协议失败、迁移计数不一致、权限泄露、配置无法发布、项目写入事务不一致、备份恢复失败。

**回滚步骤：** 停止定制 Gitea，恢复切换前 Gitea 数据和配置，恢复旧 Gitea + Workbench Compose 与 Nginx 分流；不把新系统写入反向同步到旧系统。

## 15. 提交顺序

每个任务至少一个独立提交，推荐顺序：

1. `chore: establish gitea v1.27.1 fork baseline`
2. `chore: migrate internal docs and operations assets`
3. `chore: lock gitea development toolchain`
4. `feat: add org project settings and sqlite guard`
5. `feat: add org project schema and models`
6. `feat: add versioned org project configuration`
7. `feat: add org project permissions`
8. `feat: add org project command services`
9. `feat: add typed org project queries`
10. `feat: add project token scopes and api`
11. `feat: add native org project web experience`
12. `feat: add org project activity summaries`
13. `feat: add workbench project importer`
14. `ops: deploy single customized gitea service`
15. `test: add org project end-to-end acceptance`

不要把生成的 Swagger/OpenAPI、前端产物、迁移、业务代码和运维切换混入同一个不可审查提交。

## 16. 每阶段统一质量门槛

窄范围开发循环：

```bash
go test ./models/orgproject/...
go test ./services/orgproject/...
go test ./routers/web/orgproject ./routers/api/v1/orgproject
pnpm exec vitest run web_src/js/components/orgproject
git diff --check
```

合并前完整门槛：

```bash
make test-backend
make test-frontend
GITEA_TEST_DATABASE=sqlite make test-integration
GITEA_TEST_DATABASE=sqlite make test-migration
make lint-backend
make lint-frontend
make lint-templates
make generate-swagger
make swagger-check
make swagger-validate
make frontend
make backend
ops/tests/check.sh
git diff --check
```

端到端门槛：

```bash
GITEA_TEST_DATABASE=sqlite make test-e2e GITEA_TEST_E2E_FLAGS='tests/e2e/org-project.test.ts'
docker build --build-arg GITEA_VERSION=1.27.1-internal.1 -t gitea-internal:acceptance .
```

## 17. 计划自审结论

- **命名风险已隔离：** 新模块、路由和数据表不覆盖 Gitea 上游 Issue 看板。
- **历史风险已隔离：** Gitea 是第一父历史；旧平台通过合并祖先和保留分支可恢复；当前 `main` 不原地替换。
- **迁移风险已隔离：** 先预检、后导入、支持幂等、保留报告；切换失败直接恢复完整备份。
- **权限风险已隔离：** 项目角色与 Token Scope 双重检查；仓库数据在服务层按当前用户裁剪。
- **配置风险已隔离：** draft/published 分离、immutable snapshot、pointer 原子切换、乐观锁和失败不切换。
- **查询风险已隔离：** SQLite-only、受控 AST、参数绑定、代码生成别名、页大小和指标桶限制。
- **上游维护风险已记录：** 本地 migration 343 之后的安全回移必须显式重编号；不合并后续功能版本。
- **环境风险已发现：** 当前本机 Go 版本不足，执行前先使用上游 Docker 工具链或安装 Go 1.26.4。
- **范围保持：** 首期没有通用低代码、工作流、公式、自定义 SQL、脚本、SaaS、多租户、计费或白标。

## 18. 开始实施前的确认点

执行源码树转换将使新分支工作区从当前小型部署仓库变为约 300 MB 的 Gitea 源码树，并按 Task 2 不恢复 `workbench/` 和旧 custom 壳。旧内容仍在 `legacy/workbench-main` 与 Git 历史中。

实施从 Task 1 开始。在用户确认本计划后，先创建并验证 `codex/gitea-native-v1.27.1`，完成基线和文档/运维资产迁移，再进入模型开发。
