/**
 * 全站主题入口：继承 VitePress 默认主题，只做三件事。
 *
 * 1) 首页 hero 的插槽（版本徽章挂在标题上方 / 指标条挂在 hero 下方）；
 * 2) doc-before 插槽补一个 h1（内容源只写 frontmatter.title）；
 * 3) 全局注册 HomeSections，首页 markdown 直接写 <HomeSections />；
 * 4) 引入 --lp-* 设计变量与全站微调。
 */

import { h } from 'vue'
import type { Theme } from 'vitepress'
import DefaultTheme from 'vitepress/theme'

import HomeHeroBadge from './components/HomeHeroBadge.vue'
import HomeHeroMetrics from './components/HomeHeroMetrics.vue'
import HomeSections from './components/HomeSections.vue'
import DocTitle from './components/DocTitle.vue'

import './style.css'

export default {
	extends: DefaultTheme,
	Layout() {
		return h(DefaultTheme.Layout, null, {
			'home-hero-info-before': () => h(HomeHeroBadge),
			'home-hero-after': () => h(HomeHeroMetrics),
			// 内容源不写 # 标题，课程页的 h1 由 frontmatter.title 补上
			'doc-before': () => h(DocTitle),
		})
	},
	enhanceApp({ app }) {
		app.component('HomeSections', HomeSections)
	},
} satisfies Theme
