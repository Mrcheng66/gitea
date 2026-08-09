# Design — Gitea 研发项目管理

本文件锁定组织项目模块的页面设计系统。项目总账、项目详情、代码活动和变更记录必须共享同一套视觉语言。

## Genre

modern-minimal。面向研发部门内部使用，强调快速浏览、横向比较和风险识别。

## Macrostructure family

- 应用页面：Workbench，以工具栏、数据表格、详情面板为主。
- 内容页面：沿用 Gitea 原生容器和组织导航。
- 不使用营销页结构、装饰图片或独立应用壳。

## Theme

颜色全部映射到 Gitea 现有主题变量，兼容亮色和暗色主题：

- 页面：`var(--color-body)`。
- 主表面：`var(--color-box-body)`。
- 次表面：`var(--color-box-header)`。
- 正文：`var(--color-text)`。
- 次要文字：`var(--color-text-light-2)`。
- 边界：`var(--color-secondary)`。
- 强调：`var(--color-primary)`。
- 风险：`var(--color-red)`、`var(--color-yellow)`、`var(--color-green)`。

## Typography

- Display：`var(--fonts-regular)`，600，normal。
- Body：`var(--fonts-regular)`，400。
- Mono：`var(--fonts-monospace)`，400。
- 控件和表格保持 Gitea 的 12–14px 信息密度。
- 标题使用轻微负字距，不使用斜体或装饰字体。

## Spacing

采用 4px 基础比例：4、8、12、16、24、32px。页面优先使用项目令牌和现有 `tw-gap-*` / `tw-p-*` 工具类。

## Motion

- motion-cut；不做页面进入动画。
- 仅对按钮、链接、筛选和表格行提供必要状态反馈。
- 尊重 `prefers-reduced-motion`。

## Microinteractions stance

- 保存成功使用 Gitea 现有提示。
- 悬停只改变表面或文字颜色，不移动布局。
- `:focus-visible` 使用 Gitea 主色焦点环。
- 总账不支持行内编辑；详情页进入编辑状态后修改。

## CTA voice

- 主操作使用 Gitea `ui primary button`。
- 次操作使用 `ui button` 或文本链接。
- 危险操作与普通保存分区，不并列强调。

## Per-page allowances

- 项目总账：风险摘要、搜索筛选、数据表格和分页。
- 项目详情：概览主栏、责任与日期侧栏、仓库和代码活动。
- 代码活动与历史：复用详情页签和相同面板语言。
- 应用页面不使用视觉 enrichment；功能本身构成页面。

## What pages MUST share

- Gitea 全局与组织导航。
- 颜色、字体、4px 间距体系和 4–8px 圆角。
- 风险、阶段、进度和成员的展示语义。
- 表格、面板、页签、按钮和焦点状态。

## What pages MAY differ on

- 总账使用横向表格；详情使用主栏与侧栏。
- 活动和历史可以采用更适合时间信息的列表。

## Exports

生产页面通过根目录 `tokens.css` 映射到 Gitea 主题变量；不得在项目模块 CSS 中新增硬编码品牌颜色或字体。
