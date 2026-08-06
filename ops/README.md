# 内部 Gitea 运维资产

本目录用于构建和运行基于 Gitea v1.27.1 的内部 Fork。生产拓扑只有一个定制 Gitea 服务，组织项目、配置、指标、活动和历史均由 Gitea 原生模块提供。

- `compose/`：单服务 Compose 与固定版本环境变量示例。
- `deploy/`：Nginx 和 systemd 配置。
- `scripts/`：COS 备份、健康检查和辅助脚本。
- `tests/`：Compose、脚本和 Nginx 路由检查。
- `examples/`：COS 与备份配置示例。
- `migration/`：旧 Workbench 数据的一次性导入说明和测试数据。
- `docs/`：部署、升级、恢复和验收手册。

在仓库根目录运行：

```bash
ops/tests/check.sh
```

部署和切换步骤以 [`docs/deployment.md`](docs/deployment.md) 为准。旧 Workbench 数据库只在约定的回滚窗口内作为只读文件单独保留，不属于当前运行时或日常 Gitea 备份。
