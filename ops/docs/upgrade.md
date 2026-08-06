# 升级手册

平台不自动升级。每次升级都发布一个新的固定内部版本，并保留上游基线、内部提交和构建时间。

## 升级步骤

1. 阅读上游 Gitea 发布说明和内部组织项目变更，确认数据库迁移与回滚约束。
2. 在隔离环境恢复最近备份，使用目标提交完成迁移、项目权限和 Git 协议回归。
3. 手动执行并验证生产备份：

```bash
sudo systemctl start gitea-backup.service
sudo journalctl -u gitea-backup.service -n 50 --no-pager
```

4. 检出待发布提交，更新 `ops/compose/.env`：

```bash
GITEA_VERSION=1.27.1-internal.2
GITEA_INTERNAL_COMMIT=<完整发布提交 SHA>
GITEA_BUILD_DATE=<UTC RFC3339 时间>
```

不得使用浮动标签，不得让元数据 SHA 与工作树 HEAD 不一致。

5. 构建并切换单一 Gitea：

```bash
cd /opt/gitea-stack
test "$(git rev-parse HEAD)" = "$(awk -F= '$1 == "GITEA_INTERNAL_COMMIT" {print $2}' ops/compose/.env)"
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml config
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml build --pull gitea
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml up -d gitea
docker compose --project-directory . --env-file ops/compose/.env -f ops/compose/compose.yaml ps
sudo systemctl start gitea-health.service
```

6. 验证镜像标签、`/api/healthz`、组织项目列表/详情/配置/活动、Owner/编辑团队/普通成员权限，以及 HTTPS clone/push、SSH clone/push、仓库、Issue、PR 和 Release。

## 回退

如果目标版本未执行不可逆迁移，把源码和 `ops/compose/.env` 切回上一固定发布并重新构建、启动。

如果数据库迁移不可逆或出现权限泄露、计数不一致、配置指针损坏，停止 Gitea 并按[恢复手册](recovery.md)恢复切换前数据。不要用旧镜像直接读取已由新版本迁移的数据目录，也不要把新系统写入反向同步到旧 Workbench。
