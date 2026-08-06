# 内部 Gitea 运维资产

本目录用于构建和运行基于 Gitea v1.27.1 的内部 Fork。生产拓扑只有一个定制 Gitea 服务，组织项目、配置、指标、活动和历史均由 Gitea 原生模块提供。

- `compose/`：单服务 Compose 与固定版本环境变量示例。
- `deploy/`：Nginx 和 systemd 配置。
- `scripts/`：一条命令发布、COS 备份、健康检查和辅助脚本。
- `tests/`：Compose、脚本和 Nginx 路由检查。
- `examples/`：COS 与备份配置示例。
- `migration/`：旧 Workbench 数据的一次性导入说明和测试数据。
- `docs/`：部署、升级、恢复和验收手册。

## 日常发布

代码推送到 `origin/main` 后，在生产服务器运行：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh
```

查看状态或恢复最近一次发布前版本：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh status
sudo /opt/gitea-stack/ops/scripts/deploy.sh rollback
```

简短流程和不同修改场景的本地检查见 [`docs/release.md`](docs/release.md)。

## 运维资产检查

在仓库根目录运行：

```bash
ops/tests/check.sh
```

首次部署以 [`docs/deployment.md`](docs/deployment.md) 为准。旧 Workbench 数据库只在约定的回滚窗口内作为只读文件单独保留，不属于当前运行时或日常 Gitea 备份。
