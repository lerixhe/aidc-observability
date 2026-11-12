# workflows/

> 状态：三个 workflow 全部就绪（v1.0）。

| 文件 | 触发 | 内容 |
|---|---|---|
| `deploy.yml` | push `main` 且改动 `site/**`（可手动触发） | `pnpm build`（VitePress）构建 `site/` → `upload-pages-artifact` → `deploy-pages` |
| `labs.yml` | `labs/**` 变更 | 跑 `scripts/check-labs.sh`：vet + test + `expected/` 自校验 |
| `docs.yml` | PR 与 `site/**`、`labs/**/README.md`、`scripts/**` 变更 | 链接体检 + 脱敏扫描 + 规模数字体检 + Mermaid 体检 + 站点构建 |

三个 workflow 按路径分开触发，互不阻塞：改 lab 不会跑站点构建，改课程不会跑 Go 测试。

`docs.yml` 分层说明：

- **链接体检**只查站内。外链会因反爬（ACM、DOI 常返回 403）与网络抖动误报，
  脚本把这类降级为警告，只有 404 / 410 / 域名解析失败才算断链
- **脱敏扫描**只查前雇主与内部系统名，词表不入库：本地跑用 `scripts/sensitive-words.txt`，
  CI 里得用 secret 把这个文件写出来，否则脚本会打印跳过并 `exit 0`（仓库不内置红线词）

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
