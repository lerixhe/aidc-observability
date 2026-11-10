<script setup lang="ts">
/**
 * 课程页的 h1（挂到 default theme 的 doc-before 插槽）。
 *
 * 内容源只写 frontmatter title、正文从 ## 起，跟 Starlight 时期的写法一致；
 * 而 VitePress 只渲染 Markdown 里真实存在的 # 标题，所以这里按 frontmatter 补一个。
 * 首页（layout: home）不经过 VPDoc，不会插入这个标题。
 */

import { useData } from 'vitepress'

const { frontmatter } = useData()
</script>

<template>
	<h1 v-if="frontmatter.title">{{ frontmatter.title }}</h1>
</template>

<style scoped>
h1 {
	margin: 0 0 1.5rem;
	font-size: 1.75rem;
	font-weight: 700;
	line-height: 1.35;
	letter-spacing: -0.015em;
	color: var(--vp-c-text-1);
}

@media (min-width: 768px) {
	h1 {
		font-size: 2rem;
	}
}
</style>
