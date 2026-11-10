# site/ — Astro Starlight 文档站

> 状态：**v0.3**（23 课 + 附录 A 上线，M0–M6 七个模块齐全；附录 B 待 v1.0）

## 本地运行

```bash
cd site
pnpm install
pnpm dev        # http://localhost:4321/aidc-observability/
pnpm build      # 产出 site/dist/
pnpm preview    # 预览构建产物
```

## 配置要点（`astro.config.mjs`）

| 项 | 值 | 说明 |
|---|---|---|
| `site` / `base` | `https://lerixhe.github.io` + `/aidc-observability` | GitHub Pages 项目站，路径带仓库前缀 |
| `locales.root` | `zh-CN` | 单语言站，英文站推迟到 v2 |
| `sidebar` | 模块级手写 `label` + 模块内 `autogenerate` | **模块内新增课不用改配置**；新增模块才需要加一行 |
| `editLink` | 指向 `main` 分支 `site/` 目录 | 页面底部「编辑此页」 |
| `lastUpdated` | `true` | 依赖仓库 git 历史 |
| `components.Hero` | `./src/components/Hero.astro` | 覆盖内置 Hero：首页用出血背景 + 指标条版式（详见下） |
| `components.SiteTitle` | `./src/components/SiteTitle.astro` | 覆盖为空：导航栏不显示站名 |

### ⚠️ 两个容易踩的坑

1. **`autogenerate` 不能带 `label`**（Starlight ≥ 0.39 已移除该写法）。
   正确写法是 `{ label: 'M1 · 范式转换', items: [{ autogenerate: { directory: '01-paradigm' } }] }`。
2. **frontmatter 里的绝对路径链接不会自动加 `base`**（hero `actions.link` 也一样）。
   站内链接请用**相对路径**：课程之间 `./03-rdma/`、`../02-lossless/05-pfc/`，首页用 `00-intro/00-what-problem/`。

## 目录规划

```
site/src/content/docs/
├── index.mdx                    # 首页（splash 模板，正文只有一个 <HomeSections />）
├── 00-intro/                    # 课 00-01（导读与环境）
├── 01-paradigm/                 # 课 02-04
├── 02-lossless/                 # 课 05-07
├── 03-traffic-topo/             # 课 08-10
├── 04-telemetry/                # 课 11-14
├── 05-analysis/                 # 课 15-19
├── 06-selection/                # 课 20-22
└── appendix/                    # 附录 A 术语表（附录 B 待补）
```

- 目录数字前缀与文件名数字前缀都只影响侧边栏排序，**会出现在 URL 里**（`/01-paradigm/02-why-different/`）。
- 空目录用 `.gitkeep` 占位；某个模块还没有内容时，先不要挂到 `sidebar` 里（否则侧边栏出现空分组）。

## 一课的内容语汇（已验证够用）

| 需求 | 写法 |
|---|---|
| 经验映射 / 提示 / 警示 | `<Aside type="note\|tip\|caution" title="...">` |
| 要点卡 | `<Aside type="tip" title="本课要点">` |
| 折叠参考答案 | `<details><summary>参考答案与评分维度</summary>`（内部可正常写 Markdown） |
| 双栏对比 | Markdown 表格（移动端 Starlight 自动横向滚动） |
| 首页卡片 | `<CardGrid>` / `<Card>` / `<LinkCard>` |

MDX 文件必须在 frontmatter 之后 `import { Aside } from '@astrojs/starlight/components';`。

## 首页结构

首页由三部分组成，改文案时各归各处：

| 文件 | 负责 |
|---|---|
| `src/content/docs/index.mdx` | frontmatter：标题、tagline、两个 CTA；正文只挂 `<HomeSections />` |
| `src/components/Hero.astro` | 覆盖 Starlight 内置 Hero：光晕/网格背景、指标条（15 节课 / 6 模块…） |
| `src/components/landing/HomeSections.astro` | 学习路径 6 张模块卡、三段说明、收尾 CTA、边界声明 |
| `src/styles/theme.css` | 全站 `--lp-*` 设计变量（明暗两套）、导航栏布局微调；经 `customCss` 注入 |

导航栏左上角**不显示站名**（`src/components/SiteTitle.astro` 覆盖为空实现），回首页入口是侧边栏首项「课程首页」，由 `astro.config.mjs` 的 `sidebar` 第一项提供。搜索框靠 `theme.css` 里的一条 `grid-template-columns: auto 1fr auto` 贴到导航栏左侧。

注意三点：

- **只有带 `hero` frontmatter 的页面才会渲染 Hero**，所以覆盖组件不会影响课程页；`Hero.astro` 用 `id === ''` 判断首页（集合根 `index.mdx` 的 id 是空串）。
- Hero 背景层用 `inset-inline: calc(50% - 50vw)` 出血，靠 `html[data-has-hero] { overflow-x: clip }` 防横向滚动；**不要把 `100vw` 直接写在 `.lp-hero` 上**，它会撑宽 `TwoColumnContent` 的 grid 列，导致窄屏内容被裁。

## 部署

`.github/workflows/deploy.yml`：push `main`（改动 `site/**`）→ `withastro/action@v3` 构建 → 推送 `gh-pages`。
首次部署需在仓库 Settings → Pages 里把 Source 设为 **GitHub Actions**。
