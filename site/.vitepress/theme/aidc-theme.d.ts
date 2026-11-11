import type { AidcStats } from '../stats.mts'

/**
 * 给 themeConfig 补上自定义字段的类型。
 * 缺了这一段，首页组件里 `theme.value.aidcStats` 会被 TS 判为不存在。
 */
declare module 'vitepress' {
	interface ThemeConfig {
		/** 构建期扫描得出的规模数字，来源见 .vitepress/stats.mts */
		aidcStats: AidcStats
	}
}
