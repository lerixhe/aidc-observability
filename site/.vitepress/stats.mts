import { readdirSync, existsSync } from 'node:fs'
import { join } from 'node:path'

/**
 * 站点对外宣称的规模数字，唯一来源。
 *
 * 课程数与实验数一律从文件系统实数出来，不写死；只有「典型故障类型数」是内容概念，
 * 只能手写，出处见 `docs/02-lossless/07-failures.md`。
 *
 * 调用方：`config.mts` 在构建期算一次，塞进 `themeConfig.aidcStats`，
 * 首页指标条通过 `useData().theme` 读取——不要在 Vue 组件里再写死数字。
 */
export interface AidcStats {
	/** 课程数（各模块目录下的 .md 篇数，不含 appendix） */
	lessons: number
	/** 模块数（docs/ 下带两位数字前缀且有内容的目录） */
	modules: number
	/** 实验数（labs/ 下的子目录数） */
	labs: number
	/** 无损网络典型故障类型数 */
	failureModes: number
}

/** 四大典型故障：PFC 风暴 / PFC 死锁 / incast / HoL 阻塞 */
const FAILURE_MODES = 4

/** 带两位数字前缀的目录才是一个模块；`appendix`、`public` 这类不算 */
const MODULE_DIR = /^\d{2}-/

export function computeStats(docsDir: string, labsDir: string): AidcStats {
	const modules = existsSync(docsDir)
		? readdirSync(docsDir, { withFileTypes: true })
				.filter((e) => e.isDirectory() && MODULE_DIR.test(e.name))
				.map((e) => countLessons(join(docsDir, e.name)))
				.filter((n) => n > 0)
		: []

	return {
		lessons: modules.reduce((sum, n) => sum + n, 0),
		modules: modules.length,
		labs: countLabDirs(labsDir),
		failureModes: FAILURE_MODES,
	}
}

/** 单个模块目录里的课程篇数，跳过占位用的 index.md */
function countLessons(moduleDir: string): number {
	return readdirSync(moduleDir).filter((f) => f.endsWith('.md') && f !== 'index.md').length
}

/** labs/ 下每个实验一个子目录，README.md 不算 */
function countLabDirs(labsDir: string): number {
	if (!existsSync(labsDir)) return 0
	return readdirSync(labsDir, { withFileTypes: true }).filter((e) => e.isDirectory()).length
}
