# site/ — VitePress 文档站

> 状态：**v0.3**（23 课 + 附录 A/B 上线，M0–M6 七个模块齐全）

## 本地运行

```bash
cd site
pnpm install
pnpm dev        # http://localhost:5173/aidc-observability/
pnpm build      # 产出 site/.vitepress/dist/
pnpm preview    # 预览构建产物
```

## 配置要点（`.vitepress/config.mts`）

| 项 | 值 | 说明 |
|---|---|---|
| `base` | `/aidc-observability/` | GitHub Pages 项目站，路径带仓库前缀 |
| `srcDir` | `./docs` | 内容目录（配置与主题留在 `site/.vitepress/`） |
| `cleanUrls` | `true` | 产出 `00-intro/00-what-problem.html`，URL 无 `.html` 后缀 |
| `lang` | `zh-CN` | 单语言站，英文站推迟到 v2 |
| `sidebar` | 模块级手写 `text` + 模块内自动生成 | **模块内新增课不用改配置**；新增模块才需要加一行 |
| `themeConfig.siteTitle` | `false` | 导航栏不显示站名，回首页入口下放到侧边栏首项 |
| `editLink` | `${REPO}/edit/main/site/docs/:path` | 页面底部「编辑此页」 |
| `lastUpdated` | `true` | 依赖仓库 git 历史 |
| `search` | `local` | VitePress 内置本地搜索，无需外部服务 |

侧边栏在 `config.mts` 里用 `readdirSync` + frontmatter `title` 生成，所以：

- 模块内加一节课，侧边栏自动出现，**不用改配置**；
- 新模块（新目录）要在 `MODULES` 数组里加一行 `{ dir, text }`，否则不会挂载。

## 目录规划

```
site/
├── .vitepress/
│   ├── config.mts              # 站点配置：base、侧边栏生成、搜索、编辑链接
│   └── theme/
│       ├── index.ts            # 继承默认主题 + 首页 hero 插槽 + 全局组件
│       ├── style.css           # --lp-* 设计变量（明暗两套）与全站微调
│       └── components/         # 首页 hero 徽章 / 指标条 / 学习路径卡片
└── docs/                       # 内容根（srcDir）
    ├── index.md                # 首页（layout: home，正文只有一个 <HomeSections />）
    ├── public/favicon.svg
    ├── 00-intro/               # 课 00-01（导读与环境）
    ├── 01-paradigm/            # 课 02-04
    ├── 02-lossless/            # 课 05-07
    ├── 03-traffic-topo/        # 课 08-10
    ├── 04-telemetry/           # 课 11-14
    ├── 05-analysis/            # 课 15-19
    ├── 06-selection/           # 课 20-22
    └── appendix/               # 附录 A 术语表 / B 参考清单
```

- 目录数字前缀与文件名数字前缀都只影响侧边栏排序，**会出现在 URL 里**（`/01-paradigm/02-why-different`）。
- 空目录用 `.gitkeep` 占位；某个模块还没有内容时，先不要挂到 `MODULES` 里（否则侧边栏出现空分组）。

## 一课的内容语汇

| 需求 | 写法 |
|---|---|
| 经验映射 / 提示 | `::: info 标题`、`::: tip 标题` |
| 警示 | `::: warning 标题`（严重级用 `::: danger`） |
| 要点卡 | `::: tip 本课要点` |
| 折叠参考答案 | `::: details 参考答案与评分维度` |
| 双栏对比 | Markdown 表格（窄屏自动横向滚动，`style.css` 已放开 `display: table`） |

容器必须独占一行，`:::` 与内容之间留空行；容器内可以正常写 Markdown、表格与代码块。

站内链接一律写 **`.md` 相对路径**（`./03-rdma.md`、`../02-lossless/05-pfc.md`），
VitePress 会把它编译成 `cleanUrls` 的干净 URL；写目录形式 `../02-lossless/05-pfc/` 会变成死链。

## 首页结构

首页由四部分组成，改文案时各归各处：

| 文件 | 负责 |
|---|---|
| `docs/index.md` | frontmatter：`layout: home`、hero 标题/tagline、两个 CTA；正文只挂 `<HomeSections />` |
| `.vitepress/theme/components/HomeHeroBadge.vue` | hero 上方的版本徽章（挂 `home-hero-before` 插槽） |
| `.vitepress/theme/components/HomeHeroMetrics.vue` | hero 下方的指标条（挂 `home-hero-after` 插槽，15 节课 / 6 模块…） |
| `.vitepress/theme/components/HomeSections.vue` | 学习路径 7 张模块卡、三段说明、收尾 CTA、边界声明 |
| `.vitepress/theme/style.css` | 全站 `--lp-*` 设计变量（明暗两套）、hero 居中版式与光晕网格 |

导航栏左上角**不显示站名**（`themeConfig.siteTitle: false`），回首页入口是侧边栏首项「课程首页」，
由 `config.mts` 的 `sidebar` 第一项提供。

注意两点：

- 首页组件里的站内链接要过 `withBase()`（`import { withBase } from 'vitepress'`），
  否则部署到 `/aidc-observability/` 子路径会 404——这一点和 markdown 里的链接不同，后者由框架自动加前缀。
- hero 背景层用 `inset-inline: calc(50% - 50vw)` 出血，靠 `.VPHero { overflow-x: clip }`
  防横向滚动；**不要把 `100vw` 直接写在 `.VPHero` 上**，它会撑宽外层容器。

## 部署

`.github/workflows/deploy.yml`：push `main`（改动 `site/**`）→ `pnpm build` → `upload-pages-artifact` → `deploy-pages`。
首次部署需在仓库 Settings → Pages 里把 Source 设为 **GitHub Actions**。
