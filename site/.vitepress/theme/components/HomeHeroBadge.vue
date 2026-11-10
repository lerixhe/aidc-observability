<script setup lang="ts">
/**
 * 首页英雄区里、主标题上方的版本徽章（挂到 default theme 的 home-hero-info-before 插槽）。
 * 挂 hero 内部而不是 hero 之前，是为了继承 hero 的居中排版、也不被 hero 的负 margin 覆盖。
 */
</script>

<template>
	<p class="lp-eyebrow">
		<span class="lp-dot" aria-hidden="true" />
		v0.1 · 内容已上线
	</p>
</template>

<style scoped>
.lp-eyebrow {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	width: fit-content;
	margin: 0 auto 1.25rem;
	padding: 0.3rem 0.75rem 0.3rem 0.6rem;
	border: 1px solid var(--lp-border);
	border-radius: 999px;
	background: var(--lp-surface);
	backdrop-filter: blur(8px);
	color: var(--lp-muted);
	font-size: var(--sl-text-xs);
	letter-spacing: 0.02em;
}

.lp-dot {
	width: 6px;
	height: 6px;
	border-radius: 50%;
	background: var(--lp-accent);
	box-shadow: 0 0 0 3px color-mix(in srgb, var(--lp-accent) 22%, transparent);
}
</style>
