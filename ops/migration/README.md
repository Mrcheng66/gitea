# Workbench 项目迁移

一次性迁移命令将在 `gitea admin org-project` 下提供：

1. `preflight`：只读检查旧 Workbench SQLite，输出阻断项与统计报告。
2. `import`：在预检通过后导入项目、仓库关联、关注人、字段值和审计记录。
3. `verify`：复核导入数量、关联关系和权限裁剪结果。

导入必须支持幂等重跑；没有 `project_profiles` 档案的仓库不会自动创建项目。具体命令、报告格式和维护窗口步骤在 Task 16 实现后补充。
