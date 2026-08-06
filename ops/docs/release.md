# 一条命令发布

日常发布只操作 `origin/main`。代码推送到 `main` 后，在生产服务器执行：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh
```

脚本自动拉取最新提交、备份当前数据、构建镜像、切换容器并检查本机和公网健康状态。成功发布不会删除 `/srv/gitea/data`。

## 修改场景

| 修改内容 | 推送前检查 | 服务器发布 |
| --- | --- | --- |
| 前端页面、样式、TypeScript | `make lint-js` 和相关 Vitest | `sudo /opt/gitea-stack/ops/scripts/deploy.sh` |
| Go 业务逻辑 | `make fmt`、`make lint-go` 和相关 Go 测试 | `sudo /opt/gitea-stack/ops/scripts/deploy.sh` |
| `go.mod` 或 `go.sum` | `make tidy`、`make fmt`、`make lint-go` 和相关测试 | `sudo /opt/gitea-stack/ops/scripts/deploy.sh` |
| 数据库字段或迁移 | Go 检查、迁移相关测试和 `ops/tests/check.sh` | `sudo /opt/gitea-stack/ops/scripts/deploy.sh` |
| 运维脚本或 Compose | `ops/tests/check.sh` | `sudo /opt/gitea-stack/ops/scripts/deploy.sh` |

## 常用命令

发布最新 `main`：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh
```

查看当前版本和健康状态：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh status
```

恢复最近一次发布前的版本和数据：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh rollback
```

查看帮助：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh help
```

## 自动执行内容

一次有代码变化的发布依次执行：

1. 检查生产目录、工具、磁盘空间和发布锁。
2. 获取 `origin/main` 的完整提交。
3. 暂停 Gitea 并备份 `/srv/gitea/data`。
4. 重新启动旧容器，让构建期间服务继续可用。
5. 按提交生成镜像版本，例如 `1.27.1-internal.817a183823ab`。
6. 构建并启动新容器。
7. 检查本机和 `https://git.ghgmall.cn/api/healthz`。
8. 成功后记录版本，并只保留最近五份发布备份。

目标提交已经发布且服务健康时，脚本直接退出，不重复备份或构建。

## 失败处理

- 备份、配置或构建失败：旧容器继续运行，代码和 `.env` 恢复到发布前状态。
- 新容器或健康检查失败：自动恢复发布前的数据、代码、`.env` 和镜像。
- 自动回滚也失败：退出码为 `3`，保留失败版本数据和恢复现场，不继续删除文件。

退出码：

| 退出码 | 含义 |
| --- | --- |
| `0` | 发布成功，或者当前提交已经发布且健康 |
| `1` | 切换前失败，生产旧版本仍在运行 |
| `2` | 新版本失败，但自动回滚成功 |
| `3` | 新版本和自动回滚都失败，需要人工处理 |
| `64` | 命令参数错误 |
| `75` | 已有另一个发布正在运行 |

## 发布数据

发布状态保存在：

```text
/var/lib/gitea-stack/current-release.env
/var/lib/gitea-stack/deployments.log
```

发布备份保存在：

```text
/var/lib/gitea-stack/releases/
```

每份备份包含数据归档、校验值、发布前 `.env`、Compose 配置、发布元数据和日志。脚本只保留按时间排序的最新五份。

## 第一次启用发布脚本

服务器上的旧版本还没有 `deploy.sh` 时，只需要引导一次。该命令只从远程读取脚本，不提前切换生产源码：

```bash
cd /opt/gitea-stack
git fetch origin main
git show origin/main:ops/scripts/deploy.sh \
  | sudo tee /usr/local/sbin/gitea-deploy >/dev/null
sudo chmod 0700 /usr/local/sbin/gitea-deploy
sudo /usr/local/sbin/gitea-deploy
```

第一次成功后，生产代码中已经包含脚本。之后固定使用：

```bash
sudo /opt/gitea-stack/ops/scripts/deploy.sh
```

## 首次使用检查

生产源码、运行镜像元数据和 `.env` 必须一致：

```bash
cd /opt/gitea-stack

test "$(git rev-parse HEAD)" = \
  "$(awk -F= '$1 == "GITEA_INTERNAL_COMMIT" {print $2}' ops/compose/.env)"
```

确认数据目录：

```bash
sudo test -d /srv/gitea/data
sudo test ! -L /srv/gitea/data
```

确认服务健康后再执行第一次一条命令发布：

```bash
curl -fsS http://127.0.0.1:3000/api/healthz
curl -fsS https://git.ghgmall.cn/api/healthz
```

## 禁止操作

正常发布不要执行：

```bash
docker compose down -v
docker volume prune
docker system prune --volumes
rm -rf /srv/gitea/data
```

宝塔 Nginx 仍然代理到 `127.0.0.1:3000`，日常代码发布不需要修改域名、证书或反向代理。
