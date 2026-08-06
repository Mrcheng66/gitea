# 恢复手册

本流程用于在新 Ubuntu Server 24.04 LTS 主机恢复 COS 中的当前 Gitea 日备份。目标 RPO 为 24 小时、RTO 为 2 小时。

## 1. 准备主机和源码

1. 安装 Docker Engine、Docker Compose v2、Git、`curl`、`jq`、`openssl`、`tar` 和固定版本 COSCLI。
2. 创建 `/srv/gitea`、`/etc/gitea-stack` 和 `/var/lib/gitea-stack`，但不要提前创建 `/srv/gitea/data`。
3. 重新创建 `/etc/gitea-stack/cos.yaml`。
4. 恢复宿主机 Nginx、域名和证书；这些内容不在备份中。

## 2. 下载并检查备份

```bash
sudo install -d -m 0700 /var/lib/gitea-stack/restore
sudo coscli --config-path /etc/gitea-stack/cos.yaml --disable-log cp \
  cos://<存储桶>/gitea/current/gitea-backup.tar.gz \
  /var/lib/gitea-stack/restore/gitea-backup.tar.gz \
  --process-log=false --fail-output=false
sudo tar -tzf /var/lib/gitea-stack/restore/gitea-backup.tar.gz >/dev/null
sudo tar -xOzf /var/lib/gitea-stack/restore/gitea-backup.tar.gz backup-metadata.txt
```

校验失败时下载 COS 的上一个对象版本。根据 `backup-metadata.txt` 检出准确的 `gitea_internal_commit`，不要使用分支最新提交代替备份对应版本。

## 3. 恢复数据和配置

确认没有 Gitea 容器读取目标目录：

```bash
cd /opt/gitea-stack
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml down 2>/dev/null || true
sudo test ! -e /srv/gitea/data
sudo tar -xzf /var/lib/gitea-stack/restore/gitea-backup.tar.gz -C /srv/gitea data
sudo tar -xzf /var/lib/gitea-stack/restore/gitea-backup.tar.gz -C /opt/gitea-stack \
  ops/compose/compose.yaml ops/compose/.env backup-metadata.txt
sudo chown -R 1000:1000 /srv/gitea/data
sudo chmod 0644 /opt/gitea-stack/ops/compose/.env
```

目标数据目录已存在时必须停止恢复，先确认和隔离现有内容，禁止直接覆盖。

## 4. 构建、启动和验证

```bash
cd /opt/gitea-stack
test "$(git rev-parse HEAD)" = "$(awk -F= '$1 == "GITEA_INTERNAL_COMMIT" {print $2}' ops/compose/.env)"
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml config
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml build gitea
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml up -d gitea
curl -fsS http://127.0.0.1:3000/api/healthz
sudo systemctl start gitea-health.service
```

确认 Nginx 仅代理到 3000，再检查登录、组织项目、配置发布指针、仓库访问、HTTPS/SSH Git、Issue、PR 和 Release。重新安装 systemd 单元，并手动完成一次备份。

旧 Workbench 数据库不属于当前日备份。若仍处于切换回滚窗口，应从单独保存的只读副本恢复旧双服务环境，而不是挂载到当前 Gitea。

## 5. 完成恢复

服务稳定后删除下载归档：

```bash
sudo rm -f /var/lib/gitea-stack/restore/gitea-backup.tar.gz
```

保留 `backup-metadata.txt`，记录本次恢复使用的镜像版本、上游提交、内部提交和构建时间。
