# workflows/

> 状态：`deploy.yml` 已就绪（v0.1）；`ci.yml` / `sensitive.yml` 待 v0.2。

| 文件 | 触发 | 内容 |
|---|---|---|
| `deploy.yml` | ✅ push `main` 且改动 `site/**`（可手动触发） | `withastro/action@v3` 构建 `site/` → `deploy-pages` |
| `ci.yml` | push / PR | markdownlint + `astro check` + `go test ./labs/...` + `docker compose config` |
| `sensitive.yml` | PR | 跑 `scripts/check-sensitive.sh`（用内置最小词表） |

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
    steps:
      - uses: actions/checkout@v4
      - uses: withastro/action@v3
        with:
          path: ./site
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment: github-pages
    steps:
      - uses: actions/deploy-pages@v4
```
