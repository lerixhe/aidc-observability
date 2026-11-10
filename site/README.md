# site/ — Astro Starlight 文档站

> 状态：**v0.1 试点中**（站点骨架已就绪，已迁移 1 课试点，其余 16 课待铺开）

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

### ⚠️ 两个容易踩的坑

1. **`autogenerate` 不能带 `label`**（Starlight ≥ 0.39 已移除该写法）。
   正确写法是 `{ label: 'M1 · 范式转换', items: [{ autogenerate: { directory: '01-paradigm' } }] }`。
2. **frontmatter 里的绝对路径链接不会自动加 `base`**（hero `actions.link` 也一样）。
   站内链接请用**相对路径**：课程之间 `./03-rdma/`、`../02-lossless/05-pfc/`，首页用 `01-paradigm/02-why-different/`。

## 目录规划

```
site/src/content/docs/
├── index.mdx                    # Splash 首页
├── 00-intro/                    # 课 00-01（v0.3 新写）
├── 01-paradigm/                 # 课 02-04  ← 试点已落地 02
├── 02-lossless/                 # 课 05-07
├── 03-traffic-topo/             # 课 08-10
├── 04-telemetry/                # 课 11-14
├── 05-analysis/                 # 课 15-19
├── 06-selection/                # 课 20-22
└── appendix/                    # 术语表 / 参考资料
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

## 部署

`.github/workflows/deploy.yml`：push `main`（改动 `site/**`）→ `withastro/action@v3` 构建 → 推送 `gh-pages`。
首次部署需在仓库 Settings → Pages 里把 Source 设为 **GitHub Actions**。
