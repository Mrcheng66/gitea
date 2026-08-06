# Design - 研发代码平台

这是研发代码平台的锁定设计系统。管理壳和定制后的 Gitea 页面都必须读取并遵循本文件，不为单个页面重新生成主题。

## Genre

`modern-minimal`

界面服务于研发团队的高频扫描和操作。表达应克制、准确、工具化，不使用营销式大标题、装饰性插画、玻璃效果或大面积品牌色。

## Macrostructure Family

- 应用页面：`Workbench`。项目状态是主视图，使用紧凑表格或列表；最近代码和成员动态作为次级信息流。
- 内容页面：`Long Document`。仅用于未来的帮助和运维内容，保持连续阅读结构。
- Gitea 原生页面：保留上游信息架构，只统一导航、颜色、字体、控件和间距。
- 营销页面：当前范围不包含。

应用页面使用完整贴边应用栏，不使用营销站点的浮动导航或页脚。主导航顺序固定为：工作台、项目、成员、最近代码、探索。

## Theme

主题为浅色、绿色锚点的定制调色板。轴值为 `light / geometric-sans / chromatic-green`。

- `--color-paper`: `oklch(98% 0.006 145)`
- `--color-paper-2`: `oklch(96% 0.008 145)`
- `--color-paper-3`: `oklch(93% 0.010 145)`
- `--color-ink`: `oklch(22% 0.012 145)`
- `--color-ink-2`: `oklch(34% 0.012 145)`
- `--color-rule`: `oklch(86% 0.010 145)`
- `--color-rule-2`: `oklch(78% 0.012 145)`
- `--color-muted`: `oklch(52% 0.010 145)`
- `--color-neutral`: `oklch(42% 0.012 145)`
- `--color-accent`: `oklch(52% 0.130 140)`
- `--color-accent-ink`: `oklch(98% 0.006 145)`
- `--color-focus`: `oklch(58% 0.180 140)`
- `--color-link`: `oklch(50% 0.130 250)`
- `--color-info`: `oklch(60% 0.110 250)`
- `--color-warning`: `oklch(65% 0.130 75)`
- `--color-danger`: `oklch(56% 0.180 25)`

绿色只用于品牌、主操作、当前导航和进度。蓝色只用于代码、仓库和可导航文本。状态色必须同时配合文字，不以颜色单独传达含义。

## Typography

- Display：Geist，600，normal。
- Body：Geist，400 或 500。
- Chinese fallback：`"Noto Sans SC"`, `"PingFang SC"`, `"Microsoft YaHei"`。
- Mono：Geist Mono，400 或 500，仅用于提交哈希、分支、标签和代码。
- 字间距固定为 `0`。
- 应用标题最大使用 `--text-xl`，面板标题使用 `--text-sm` 或 `--text-md`。

字体资源必须随应用构建，不依赖运行时外部 CDN。中文字体使用系统回退，不打包大型字库。

## Spacing

使用 4 点间距系统：

- `--space-3xs`: `0.25rem`
- `--space-2xs`: `0.5rem`
- `--space-xs`: `0.75rem`
- `--space-sm`: `1rem`
- `--space-md`: `1.5rem`
- `--space-lg`: `2rem`
- `--space-xl`: `3rem`
- `--space-2xl`: `4rem`

表格行高、工具栏、图标按钮和头像必须使用稳定尺寸，动态内容不能改变整体布局。

## Motion

- `--ease-out`: `cubic-bezier(0.16, 1, 0.3, 1)`
- `--ease-in`: `cubic-bezier(0.7, 0, 0.84, 0)`
- `--ease-in-out`: `cubic-bezier(0.65, 0, 0.35, 1)`
- `--dur-micro`: `120ms`
- `--dur-short`: `180ms`
- `--dur-long`: `280ms`

只动画 `transform` 和 `opacity`。表单抽屉和弹窗允许一次短过渡；表格、统计和信息流不做入场动画。`prefers-reduced-motion` 下只保留不超过 120ms 的透明度变化。

## Microinteractions Stance

- 保存成功使用表单内静默反馈，不弹庆祝式提示。
- 冲突、权限不足和同步失败必须在相关区域内说明原因和下一步。
- 焦点环立即出现，不参与动画。
- 工具提示仅用于不熟悉的图标；悬停延迟 800ms，键盘聚焦立即显示。
- 删除等不可逆操作使用明确确认；普通编辑直接保存并提供变更记录。

## Component Voice

- 圆角：卡片和面板 `5px`，输入 `5px`，状态标签 `4px`，不用大圆角胶囊承载普通文字。
- 边框：1px 中性规则线；不使用多层阴影。弹窗可以使用一层克制阴影。
- 页面分区不做浮动卡片。边框容器只用于表格、列表、表单和弹窗等真实工具表面。
- 不在卡片内嵌套卡片。
- 主要操作使用绿色实心按钮；次级命令使用中性描边或图标按钮。
- 项目阶段使用标签，进展使用进度条加数值，负责人和跟进人使用头像与姓名。

## Page Allowances

- 工作台：项目状态表为首屏主体；最近代码占下方约三分之二，成员动态独立占三分之一。
- 项目列表：允许更完整的筛选、排序和分页，不重复工作台的信息流。
- 项目详情：项目档案、进展历史和关联仓库按无嵌套的纵向分区组织。
- 成员列表：以成员为行，展示参与项目和最近活动，不使用头像卡片墙。
- 最近代码：以时间顺序列表展示，可按项目、成员和分支筛选。
- Gitea 页面：不改变代码浏览、提交详情、合并请求和设置的核心结构。

## What Pages Must Share

- 产品名称、标志、应用导航和当前状态表达。
- 本文件定义的颜色、字体、间距、圆角和交互反馈。
- 项目、仓库、成员和提交的链接样式。
- 桌面与移动端的焦点、加载、空状态、失败和无权限状态。

## What Pages May Differ On

- 信息密度和表格列数可以按页面任务调整。
- 工作台使用组合视图，列表页使用单一主列表，详情页使用纵向分区。
- Gitea 原生页面保留上游组件布局，但不得偏离主题令牌。

## Accessibility

- 正文和控件满足 WCAG AA；焦点环与相邻颜色至少 3:1。
- 所有图标按钮有可访问名称，所有表单控件有显式标签。
- 状态不只依赖颜色；进度条同时提供数值和可访问描述。
- 320、375、414、768 以及桌面宽度不得出现页面级横向滚动。仅项目表格可以在自身容器内水平滚动。
- 可点击文字保持单行；长仓库名允许在展示区域内省略并通过详情读取全名。

## Exports

### tokens.css

```css
:root {
  --color-paper: oklch(98% 0.006 145);
  --color-paper-2: oklch(96% 0.008 145);
  --color-paper-3: oklch(93% 0.010 145);
  --color-ink: oklch(22% 0.012 145);
  --color-ink-2: oklch(34% 0.012 145);
  --color-rule: oklch(86% 0.010 145);
  --color-rule-2: oklch(78% 0.012 145);
  --color-muted: oklch(52% 0.010 145);
  --color-neutral: oklch(42% 0.012 145);
  --color-accent: oklch(52% 0.130 140);
  --color-accent-ink: oklch(98% 0.006 145);
  --color-focus: oklch(58% 0.180 140);
  --color-link: oklch(50% 0.130 250);
  --color-info: oklch(60% 0.110 250);
  --color-warning: oklch(65% 0.130 75);
  --color-danger: oklch(56% 0.180 25);

  --font-display: "Geist", "Noto Sans SC", "PingFang SC", ui-sans-serif, system-ui, sans-serif;
  --font-body: "Geist", "Noto Sans SC", "PingFang SC", ui-sans-serif, system-ui, sans-serif;
  --font-outlier: "Geist Mono", "SFMono-Regular", Consolas, ui-monospace, monospace;

  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-md: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.375rem;
  --text-2xl: 1.75rem;

  --space-3xs: 0.25rem;
  --space-2xs: 0.5rem;
  --space-xs: 0.75rem;
  --space-sm: 1rem;
  --space-md: 1.5rem;
  --space-lg: 2rem;
  --space-xl: 3rem;
  --space-2xl: 4rem;

  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-in: cubic-bezier(0.7, 0, 0.84, 0);
  --ease-in-out: cubic-bezier(0.65, 0, 0.35, 1);
  --dur-micro: 120ms;
  --dur-short: 180ms;
  --dur-long: 280ms;

  --rule-hair: 1px;
  --rule-fine: 2px;
  --radius-card: 5px;
  --radius-pill: 999px;
  --radius-input: 5px;
}
```
### Tailwind v4 `@theme`

```css
@theme {
  --color-paper: oklch(98% 0.006 145);
  --color-paper-2: oklch(96% 0.008 145);
  --color-paper-3: oklch(93% 0.010 145);
  --color-ink: oklch(22% 0.012 145);
  --color-ink-2: oklch(34% 0.012 145);
  --color-rule: oklch(86% 0.010 145);
  --color-muted: oklch(52% 0.010 145);
  --color-accent: oklch(52% 0.130 140);
  --color-focus: oklch(58% 0.180 140);
  --color-link: oklch(50% 0.130 250);

  --font-display: "Geist", "Noto Sans SC", ui-sans-serif, system-ui, sans-serif;
  --font-body: "Geist", "Noto Sans SC", ui-sans-serif, system-ui, sans-serif;
  --font-outlier: "Geist Mono", ui-monospace, monospace;

  --spacing-3xs: 0.25rem;
  --spacing-2xs: 0.5rem;
  --spacing-xs: 0.75rem;
  --spacing-sm: 1rem;
  --spacing-md: 1.5rem;
  --spacing-lg: 2rem;
  --spacing-xl: 3rem;
  --spacing-2xl: 4rem;

  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-md: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.375rem;
  --text-2xl: 1.75rem;

  --radius-card: 5px;
  --radius-input: 5px;
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-in: cubic-bezier(0.7, 0, 0.84, 0);
  --ease-in-out: cubic-bezier(0.65, 0, 0.35, 1);
}
```

### DTCG `tokens.json`

```json
{
  "$schema": "https://design-tokens.github.io/community-group/format/",
  "color": {
    "paper": { "$value": "oklch(98% 0.006 145)", "$type": "color" },
    "paper-2": { "$value": "oklch(96% 0.008 145)", "$type": "color" },
    "paper-3": { "$value": "oklch(93% 0.010 145)", "$type": "color" },
    "ink": { "$value": "oklch(22% 0.012 145)", "$type": "color" },
    "ink-2": { "$value": "oklch(34% 0.012 145)", "$type": "color" },
    "rule": { "$value": "oklch(86% 0.010 145)", "$type": "color" },
    "muted": { "$value": "oklch(52% 0.010 145)", "$type": "color" },
    "accent": { "$value": "oklch(52% 0.130 140)", "$type": "color" },
    "focus": { "$value": "oklch(58% 0.180 140)", "$type": "color" },
    "link": { "$value": "oklch(50% 0.130 250)", "$type": "color" }
  },
  "font": {
    "display": { "$value": "Geist, Noto Sans SC, PingFang SC, ui-sans-serif, system-ui, sans-serif", "$type": "fontFamily" },
    "body": { "$value": "Geist, Noto Sans SC, PingFang SC, ui-sans-serif, system-ui, sans-serif", "$type": "fontFamily" },
    "outlier": { "$value": "Geist Mono, SFMono-Regular, Consolas, ui-monospace, monospace", "$type": "fontFamily" }
  },
  "space": {
    "3xs": { "$value": "0.25rem", "$type": "dimension" },
    "2xs": { "$value": "0.5rem", "$type": "dimension" },
    "xs": { "$value": "0.75rem", "$type": "dimension" },
    "sm": { "$value": "1rem", "$type": "dimension" },
    "md": { "$value": "1.5rem", "$type": "dimension" },
    "lg": { "$value": "2rem", "$type": "dimension" },
    "xl": { "$value": "3rem", "$type": "dimension" },
    "2xl": { "$value": "4rem", "$type": "dimension" }
  },
  "duration": {
    "micro": { "$value": "120ms", "$type": "duration" },
    "short": { "$value": "180ms", "$type": "duration" },
    "long": { "$value": "280ms", "$type": "duration" }
  }
}
```

### shadcn/ui CSS variables

```css
:root {
  --background: 98% 0.006 145;
  --foreground: 22% 0.012 145;
  --card: 96% 0.008 145;
  --card-foreground: 22% 0.012 145;
  --popover: 96% 0.008 145;
  --popover-foreground: 22% 0.012 145;
  --primary: 52% 0.130 140;
  --primary-foreground: 98% 0.006 145;
  --secondary: 93% 0.010 145;
  --secondary-foreground: 34% 0.012 145;
  --muted: 86% 0.010 145;
  --muted-foreground: 52% 0.010 145;
  --accent: 52% 0.130 140;
  --accent-foreground: 98% 0.006 145;
  --destructive: 56% 0.180 25;
  --destructive-foreground: 98% 0.006 145;
  --border: 86% 0.010 145;
  --input: 86% 0.010 145;
  --ring: 58% 0.180 140;
  --radius: 5px;
}
```
