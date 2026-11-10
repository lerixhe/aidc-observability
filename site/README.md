# site/ — Astro Starlight 文档站

> 状态：**规划中**，v0.1 起初始化。当前目录为空。

## 计划初始化

```bash
cd site
pnpm create astro@latest . --template starlight
pnpm add -D @astrojs/starlight
pnpm dev          # http://localhost:4321
```

## 目录规划

```
site/
├── astro.config.mjs      # 站点配置，sidebar 用 autogenerate
├── package.json
├── tsconfig.json
└── src/content/docs/
    ├── index.mdx         # Splash 首页
    ├── 00-intro/
    │   ├── 00-about.md
    │   └── 01-setup.md
    ├── 01-paradigm/       # 02-04
    ├── 02-lossless/       # 05-07
    ├── 03-traffic-topo/   # 08-10
    ├── 04-telemetry/      # 11-14
    ├── 05-analysis/       # 15-19
    ├── 06-selection/      # 20-22
    └── appendix/          # 术语表 / 参考资料
```

## Starlight 约定（摘自 docs/PROJECT.md）

- 侧边栏 `autogenerate` + 目录数字前缀，新增课不改配置
- 经验映射用 `<Aside type="note" title="经验映射">`
- 要点卡用 `<Aside type="tip" title="本课要点">`
- 折叠答案保留 MDX 原生 `<details>`
- 不用手写上下课导航（用内置 pagination）

## 部署

GitHub Actions（`.github/workflows/deploy.yml`）用 `withastro/action@v3`
构建 `site/` 并推送到 `gh-pages`。v0.1 落地时配置。
