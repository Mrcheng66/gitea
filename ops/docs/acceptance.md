# 验收清单

## 自动检查

在仓库根目录运行：

```bash
ops/tests/check.sh

go test ./models/orgproject/...
go test ./services/orgproject/...
go test ./routers/web/orgproject ./routers/api/v1/orgproject
pnpm exec vitest run web_src/js/components/orgproject
```

`ops/tests/check.sh` 验证单服务 Compose、固定内部镜像版本、构建元数据、回环端口绑定、Nginx 单一上游、systemd、备份故障恢复，以及 SQLite/JSON1/配置指针健康检查。

## 发布与拓扑

- [ ] Compose 渲染后只有 `gitea` 服务，未出现 Workbench、8081、旧数据卷或旧 env file。
- [ ] 镜像由当前提交构建，标签中的内部版本、上游 SHA、内部 SHA 和构建时间与发布记录一致。
- [ ] 只监听 `127.0.0.1:3000` 和公开 SSH 2222；3000 无法从公网直接访问。
- [ ] Nginx 配置不含 `/_workbench` 或路径分流，所有 Web/Git HTTP 路径均进入 Gitea 3000。
- [ ] `GITEA__org_project__ENABLED=true`，数据库类型为 SQLite，数据文件为 `/data/gitea/gitea.db`。

## 功能和权限

- [ ] 未登录用户无法查看私有组织、仓库或项目。
- [ ] 组织成员可读取项目；配置的编辑团队可创建和维护项目；组织 Owners 可编辑、发布和回滚配置。
- [ ] 普通成员、非成员和不可见仓库不会通过页面、API、指标、活动或错误正文泄露数据。
- [ ] 项目可关联零到多个可访问仓库；仓库反向关联页与项目详情一致。
- [ ] 配置草稿不影响线上查询；发布原子生效；过期乐观锁写入被拒绝；回滚生成新的发布版本。
- [ ] Workbench preflight 无阻断项；首次 import 计数正确；第二次 import 零新增。
- [ ] HTTPS 和 SSH clone/pull/push、仓库浏览、Issue、PR 和 Release 无回归。

## 运维

- [ ] `/api/healthz` 成功；定时健康检查通过根路径、SQLite 类型、模块启用、JSON1、配置指针、TLS、磁盘和备份状态检查。
- [ ] 人为破坏任一健康条件时，服务返回失败并在 journal 中记录具体原因。
- [ ] 手动备份成功，归档包含 `data/`、`ops/compose/compose.yaml`、`ops/compose/.env` 和 `backup-metadata.txt`，不包含 Workbench 运行数据。
- [ ] 备份失败时 Gitea 会重新启动，COS 当前对象保持不变，锁目录被清理。
- [ ] 从 COS 在隔离主机恢复后，数据、发布元数据、组织项目和 Git 协议检查通过。
- [ ] 旧 Workbench SQLite 只读副本在约定回滚窗口内单独保留，窗口结束后按批准流程销毁。
