# Nginx 边缘代理部署设计

## 背景与目标

生产域名 `git.ghgmall.cn` 已由宿主机 Nginx 提供 HTTPS、证书续期和公网反向代理。项目不再运行第二个边缘代理，避免重复管理证书、争抢 80/443 端口以及形成两套上线流程。

本次改造要彻底移除项目中的旧边缘代理运行配置、环境变量、备份内容、恢复步骤、测试和文档描述，将部署模型统一为“宿主机 Nginx + Docker Compose 中的 Gitea 与 Workbench”。

## 架构

浏览器和 Git HTTPS 请求先到达宿主机 Nginx。Nginx 终止 TLS，并按精确路径选择本机回环地址上的后端：

- `/`、`/projects`、`/projects/*`、`/people`、`/people/*`、`/activity`、`/activity/*`、`/_workbench` 和 `/_workbench/*` 转发到 `127.0.0.1:8081` 的 Workbench。
- 其他路径继续交给 `127.0.0.1:3000` 的 Gitea，包括登录、仓库、提交、合并请求、设置、Gitea API 和 Git HTTPS。
- Git SSH 继续由宿主机 `2222` 转发到 Gitea 容器的 `22`。
- Workbench 在 Compose 内部网络中通过 `http://gitea:3000` 调用 Gitea API，不绕行公网 Nginx。

Gitea 的 3000 和 Workbench 的 8081 只绑定 `127.0.0.1`，不得绑定 `0.0.0.0`，公网入口仍只有现有 Nginx。

## 配置与文件变更

- 从 `compose.yaml` 删除边缘代理服务，并为 Gitea 和 Workbench 增加回环端口绑定。
- 从 `.env.example` 删除证书邮箱、边缘代理镜像及其数据目录变量。
- 删除旧边缘代理配置文件。
- 新增可复制到现有 Nginx HTTPS `server` 块中的 Workbench 路由示例。示例只负责路径分流，不声明证书路径或接管完整虚拟主机。
- 保留现有 Gitea custom 主题、OAuth、Workbench 数据卷、COS 备份和 systemd 健康检查。

## 运维与数据边界

项目备份继续覆盖 Gitea 数据、Workbench 数据、Compose 环境文件、Workbench 私密配置和镜像版本，但不备份宿主机 Nginx 的证书或全局配置。Nginx 配置及证书由服务器现有运维机制独立管理；项目仓库中的路由示例可用于重建路径规则。

恢复时先恢复 Gitea 与 Workbench，再在现有 Nginx HTTPS 站点中安装路由规则。升级流程只涉及 Gitea 与 Workbench，不再拉取或升级边缘代理镜像。

## 错误处理与回退

- Nginx 配置必须先通过 `nginx -t`，成功后才能 reload。
- Workbench 未就绪时，不切换根路径和 Workbench 路径；Gitea 的其他路径保持可用。
- 如 Workbench 上线失败，移除新增的 Workbench `location` 规则并 reload Nginx，根路径即可恢复到原 Gitea 处理方式。
- 回环端口被占用时，Compose 应启动失败，不得改为公网绑定规避冲突。

## 测试与验收

- Compose 渲染测试确认不存在已移除的服务和环境变量。
- 端口测试确认 3000、8081 只绑定 `127.0.0.1`，2222 保持 SSH 映射。
- Nginx 路由测试确认所有 Workbench 精确路径进入 8081，`/projects-other`、`/api/*` 和仓库路径仍进入 Gitea。
- 备份测试确认归档不再要求旧边缘代理文件，并仍包含两个数据目录和 Workbench 私密配置。
- 继续运行 Shell/Compose、Go、Vue 单元测试、类型检查和生产构建。
- 真实服务器验收 HTTPS、OAuth 回调、Git HTTPS/SSH、容器重启持久性、备份恢复和 Nginx 回退路径。

## 完成标准

项目默认启动仅包含 Gitea 与 Workbench；仓库配置、脚本、自动测试和现行文档统一描述宿主机 Nginx 架构，不再残留旧边缘代理的运行依赖或操作步骤。
