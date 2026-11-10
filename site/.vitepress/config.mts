import { readdirSync, readFileSync, existsSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitepress'

// GitHub Pages：https://lerixhe.github.io/aidc-observability/
const SITE = 'https://lerixhe.github.io'
const REPO = 'https://github.com/lerixhe/aidc-observability'

const DOCS_DIR = fileURLToPath(new URL('../docs', import.meta.url))

/**
 * 侧边栏模块表：模块级手写 label，模块内自动生成。
 * 新增课（目录内加文件）无需改本文件；新增模块才需要加一行。
 * 只挂载已有内容的模块，避免侧边栏出现空分组。
 */
const MODULES = [
	{ dir: '00-intro', text: 'M0 · 导读与实验室环境' },
	{ dir: '01-paradigm', text: 'M1 · 范式转换' },
	{ dir: '02-lossless', text: 'M2 · 无损网络机制与失败模式' },
	{ dir: '03-traffic-topo', text: 'M3 · 流量与拓扑' },
	{ dir: '04-telemetry', text: 'M4 · 遥测数据面' },
	{ dir: '05-analysis', text: 'M5 · 分析与自动化' },
	{ dir: '06-selection', text: 'M6 · 选型与收口' },
	{ dir: 'appendix', text: '附录' },
]

/** 从 frontmatter 里取 title，取不到就退回文件名 */
function readTitle(file: string, fallback: string): string {
	const head = readFileSync(file, 'utf8').slice(0, 2000)
	const m = head.match(/^title:\s*(.+)$/m)
	return m ? m[1].trim().replace(/^["']|["']$/g, '') : fallback
}

function moduleItems(dir: string) {
	const abs = join(DOCS_DIR, dir)
	if (!existsSync(abs)) return []
	return readdirSync(abs)
		.filter((f) => f.endsWith('.md') && f !== 'index.md')
		.sort()
		.map((f) => {
			const base = f.replace(/\.md$/, '')
			return {
				text: readTitle(join(abs, f), base),
				link: `/${dir}/${base}`,
			}
		})
}

const sidebar = [
	// 导航栏已去掉站名（siteTitle: false），回首页入口放这里
	{ text: '课程首页', link: '/' },
	...MODULES.map(({ dir, text }) => {
		const items = moduleItems(dir)
		return items.length ? { text, collapsed: false, items } : null
	}).filter((x): x is NonNullable<typeof x> => x !== null),
]

export default defineConfig({
	title: '智算网络可观测指南',
	description: '从有损 IDC 到无损智算：机制、遥测与可运行的实验室',
	lang: 'zh-CN',
	base: '/aidc-observability/',
	srcDir: './docs',
	cleanUrls: true,
	lastUpdated: true,
	sitemap: { hostname: SITE + '/aidc-observability/' },

	head: [
		['link', { rel: 'icon', type: 'image/svg+xml', href: '/aidc-observability/favicon.svg' }],
		// 中文站：显式声明语言，利于搜索引擎与阅读器
		['meta', { property: 'og:locale', content: 'zh_CN' }],
	],

	themeConfig: {
		// 导航栏不显示站名，回首页入口下放到侧边栏首项
		siteTitle: false,
		outline: { level: [2, 3], label: '本页目录' },
		socialLinks: [{ icon: 'github', link: REPO }],
		editLink: {
			pattern: `${REPO}/edit/main/site/docs/:path`,
			text: '在 GitHub 上编辑此页',
		},
		lastUpdated: {
			text: '最后更新于',
			formatOptions: { dateStyle: 'medium' },
		},
		docFooter: { prev: '上一课', next: '下一课' },
		sidebarMenuLabel: '目录',
		returnToTopLabel: '回到顶部',
		darkModeSwitchLabel: '主题',
		sidebar,

		search: {
			provider: 'local',
			options: {
				translations: {
					button: {
						buttonText: '搜索文档',
						buttonAriaLabel: '搜索文档',
					},
					modal: {
						displayDetails: '显示详情',
						resetButtonTitle: '清除查询',
						resetButtonAriaLabel: '清除查询',
						noResultsText: '没有找到结果',
						footer: {
							selectText: '选择',
							navigateText: '切换',
							closeText: '关闭',
						},
					},
				},
			},
		},

	},
})
