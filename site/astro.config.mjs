// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// GitHub Pages：https://lerixhe.github.io/aidc-observability/
const SITE = 'https://lerixhe.github.io';
const BASE = '/aidc-observability';
const REPO = 'https://github.com/lerixhe/aidc-observability';

// https://astro.build/config
export default defineConfig({
	site: SITE,
	base: BASE,
	trailingSlash: 'always',
	integrations: [
		starlight({
			title: '智算网络可观测指南',
			description: '从有损 IDC 到无损智算：机制、遥测与可运行的实验室',
			locales: {
				root: { label: '简体中文', lang: 'zh-CN' },
			},
			defaultLocale: 'root',
			social: [{ icon: 'github', label: 'GitHub', href: REPO }],
			editLink: {
				baseUrl: `${REPO}/edit/main/site/`,
			},
			lastUpdated: true,
			// 侧边栏：模块级手写 label，模块内 autogenerate。
			// 新增课（目录内加文件）无需改本文件；新增模块才需要加一行。
			// 只挂载已有内容的模块，避免侧边栏出现空分组：
			// M0（00-intro）与附录（appendix）留到 v0.3 / v1.0 有内容时再加。
			// 模块内新增课时无需改本文件。
			sidebar: [
				{ label: 'M1 · 范式转换', items: [{ autogenerate: { directory: '01-paradigm' } }] },
				{ label: 'M2 · 无损网络机制与失败模式', items: [{ autogenerate: { directory: '02-lossless' } }] },
				{ label: 'M3 · 流量与拓扑', items: [{ autogenerate: { directory: '03-traffic-topo' } }] },
				{ label: 'M4 · 遥测数据面', items: [{ autogenerate: { directory: '04-telemetry' } }] },
				{ label: 'M5 · 分析与自动化', items: [{ autogenerate: { directory: '05-analysis' } }] },
				{ label: 'M6 · 选型与收口', items: [{ autogenerate: { directory: '06-selection' } }] },
			],
			head: [
				// 中文站：显式声明语言，利于搜索引擎与阅读器
				{ tag: 'meta', attrs: { property: 'og:locale', content: 'zh_CN' } },
			],
		}),
	],
});
