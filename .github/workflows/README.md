# workflows/

> 状态：三个 workflow 全部就绪（v1.0）。

| 文件 | 触发 | 内容 |
|---|---|---|
| `deploy.yml` | push `main` 且改动 `site/**`（可手动触发） | `pnpm build`（VitePress）构建 `site/` → `upload-pages-artifact` → `deploy-pages` |
| `labs.yml` | `labs/**` 变更 | 跑 `scripts/check-labs.sh`：vet + test + `expected/` 自校验 |
| `docs.yml` | PR 与 `site/**`、`docs/**`、`labs/**/README.md`、`scripts/**` 变更 | 链接体检 + 文风体检 + 脱敏扫描 + 站点构建 |

三个 workflow 按路径分开触发，互不阻塞：改 lab 不会跑站点构建，改课程不会跑 Go 测试。

`docs.yml` 分层说明：

- **链接体检**只查站内。外链会因反爬（ACM、DOI 常返回 403）与网络抖动误报，
  脚本把这类降级为警告，只有 404 / 410 / 域名解析失败才算断链
- **文风体检**按 COURSE-STYLE §2.1 分两档：硬项（分号、感叹号、超长句、句长区间、标点配额底线）
  阻塞合并；观察项（≤10 字分句占比、整句 P90）只打印
- **脱敏扫描**在 CI 用内置最小词表，本地跑才用完整词表（词表不入库）

`deploy.yml` 骨架（已落地，实际文件含 pnpm 版本固定与 `workflow_dispatch`）：

首次部署需在仓库 Settings → Pages 把 Source 设为 **GitHub Actions**。

```yaml
name: Deploy site
on:
  push:
    branches: [main]
    paths: ['site/**', '.github/workflows/deploy.yml']
permissions:
  contents: read
  pages: write
  id-token: write
jobs:
  build:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: site
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v4
        with:
          version: latest
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
          cache-dependency-path: site/pnpm-lock.yaml
      - run: pnpm install --frozen-lockfile
      - run: pnpm build
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: site/.vitepress/dist
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment: github-pages
    steps:
      - uses: actions/deploy-pages@v4
```
