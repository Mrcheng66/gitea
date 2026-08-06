# 部署手册

## 1. 拓扑和前置条件

生产只运行当前仓库构建的定制 Gitea。宿主机 Nginx 把全部 Web 请求转发到 `127.0.0.1:3000`，SSH 使用 2222；不存在 Workbench 容器、8081 端口、OAuth 壳或路径分流。

主机要求：

- Ubuntu Server 24.04 LTS，建议至少 2 vCPU、2 GB RAM、40 GB SSD。
- Docker Engine、Docker Compose v2、Git、`curl`、`jq`、`openssl`、`tar`。
- 已解析到服务器的域名和由宿主机 Nginx 管理的有效 HTTPS 证书。
- 私有腾讯云 COS 存储桶，以及仅可访问备份前缀的 CAM 凭据。

安全组开放 TCP 80、443 和 2222。管理 SSH 端口只对白名单 IP 开放。

## 2. 准备源码、数据和发布变量

把已审核的内部发布提交检出到 `/opt/gitea-stack`，创建数据目录：

```bash
sudo install -d -m 0755 /srv/gitea/data /var/lib/gitea-stack
sudo install -d -m 0700 /etc/gitea-stack
sudo chown -R 1000:1000 /srv/gitea/data
cd /opt/gitea-stack
sudo cp ops/compose/.env.example ops/compose/.env
sudo chmod 0644 ops/compose/.env
```

编辑 `ops/compose/.env`：

- `GITEA_DOMAIN`：生产域名。
- `GITEA_VERSION`：固定内部版本，例如 `1.27.1-internal.1`，禁止使用 `latest`。
- `GITEA_UPSTREAM_COMMIT`：固定为 v1.27.1 的 `a62dfffbe7d3a454c2f2d3c4b2788ba432724a5f`。
- `GITEA_INTERNAL_COMMIT`：待部署提交的完整 SHA，可用 `git rev-parse HEAD` 获取。
- `GITEA_BUILD_DATE`：UTC RFC3339 构建时间，例如 `date -u +%Y-%m-%dT%H:%M:%SZ`。
- `GITEA_DATA_DIR`：宿主机持久化目录。

发布前确认源码与元数据一致：

```bash
test "$(git rev-parse HEAD)" = "$(awk -F= '$1 == "GITEA_INTERNAL_COMMIT" {print $2}' ops/compose/.env)"
git merge-base --is-ancestor a62dfffbe7d3a454c2f2d3c4b2788ba432724a5f HEAD
```

组织项目模块只支持 SQLite。现有实例如果使用 MySQL、PostgreSQL 或 MSSQL，不得直接切换到此部署。

## 3. 构建并启动单一 Gitea

```bash
cd /opt/gitea-stack
docker compose \
  --project-directory . \
  --env-file ops/compose/.env \
  -f ops/compose/compose.yaml \
  config
docker compose \
  --project-directory . \
  --env-file ops/compose/.env \
  -f ops/compose/compose.yaml \
  build --pull gitea
docker compose \
  --project-directory . \
  --env-file ops/compose/.env \
  -f ops/compose/compose.yaml \
  up -d gitea
docker compose \
  --project-directory . \
  --env-file ops/compose/.env \
  -f ops/compose/compose.yaml \
  ps
curl -fsS http://127.0.0.1:3000/api/healthz
```

Compose 显式启用 `[org_project] ENABLED=true`，数据库固定为 `/data/gitea/gitea.db`。首次启动会执行 Gitea 原生迁移；已有数据必须先完成备份和预生产恢复演练，且同一数据目录不能同时被两个 Gitea 进程访问。

镜像标签记录内部版本、上游提交、内部提交和构建时间：

```bash
docker image inspect code-lab/gitea:1.27.1-internal.1 \
  --format '{{json .Config.Labels}}' | jq .
```

## 4. 配置 Nginx

把 `ops/deploy/nginx/gitea.conf` 中的 `location /` 合并到现有 HTTPS `server` 块。该规则把 `/`、`/projects`、`/org/*/projects`、`/api/*`、仓库页面和 Git HTTPS 请求统一转发到 Gitea 3000。

```bash
sudo nginx -t
sudo systemctl reload nginx
curl -fsS "https://<GITEA_DOMAIN>/api/healthz"
```

配置中不得出现 8081、`/_workbench` 或面向旧应用的路径分流。

## 5. 一次性导入旧项目数据

保留旧 Workbench SQLite 的只读副本，先运行预检，再导入：

冻结旧系统写入并停止当前 Gitea 后，通过一次性容器运行命令；`/backup/workbench.db` 必须是只读副本：

```bash
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml \
  run --rm --no-deps -v /backup/workbench.db:/backup/workbench.db:ro gitea \
  gitea admin org-project preflight-workbench \
  --database /backup/workbench.db \
  --organization <org> \
  --report /data/workbench-preflight.json

docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml \
  run --rm --no-deps -v /backup/workbench.db:/backup/workbench.db:ro gitea \
  gitea admin org-project import-workbench \
  --database /backup/workbench.db \
  --organization <org> \
  --actor <admin-user> \
  --report /data/workbench-import.json
```

阻断项必须为零。连续执行两次 import，第二次必须零新增。导入完成后，旧数据库只在回滚窗口内离线保留，不挂载到 Gitea 容器，也不纳入日常运行数据。

## 6. 配置 COS 和定时任务

```bash
sudo cp ops/examples/cos.env.example /etc/gitea-stack/cos.env
sudo cp ops/examples/backup.env.example /etc/gitea-stack/backup.env
sudo chmod 0600 /etc/gitea-stack/cos.env /etc/gitea-stack/backup.env
sudo ops/scripts/configure-cos.sh /etc/gitea-stack/cos.env
sudo chmod 0600 /etc/gitea-stack/cos.yaml

sudo install -m 0644 ops/deploy/systemd/gitea-backup.service /etc/systemd/system/
sudo install -m 0644 ops/deploy/systemd/gitea-backup.timer /etc/systemd/system/
sudo install -m 0644 ops/deploy/systemd/gitea-health.service /etc/systemd/system/
sudo install -m 0644 ops/deploy/systemd/gitea-health.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now gitea-backup.timer gitea-health.timer
sudo systemctl start gitea-backup.service gitea-health.service
```

日常备份只停止并归档 Gitea 数据、Compose 配置和发布元数据。健康检查验证：公网根路径、Gitea API、SQLite 数据库类型、组织项目启用状态、SQLite JSON1、已发布/草稿配置指针、TLS、磁盘和最近成功备份。

## 7. 上线检查

按[验收清单](acceptance.md)逐项完成。开放写入前必须确认导入计数、权限矩阵、原生项目页面、Git HTTPS/SSH、备份和恢复路径均正常。
