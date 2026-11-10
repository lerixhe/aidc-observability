# workflows/

> 状态：规划中，v0.1 起启用。

| 文件 | 触发 | 内容 |
|---|---|---|
| `deploy.yml` | push `main` 且改动 `site/**` | `withastro/action@v3` 构建，推 `gh-pages` |
| `ci.yml` | push / PR | markdownlint + `astro check` + `go test ./labs/...` + `docker compose config` |
| `sensitive.yml` | PR | 跑 `scripts/check-sensitive.sh`（用内置最小词表） |

`deploy.yml` 骨架（v0.1 落地时补全）：

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
