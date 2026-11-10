<script setup lang="ts">
/**
 * 首页 hero 下方的指标条（挂到 default theme 的 home-hero-after 插槽）。
 * 四个数字只讲规模，不讲能力——用来给「这站有多大」一个即时量级。
 */

const metrics = [
	{ value: '15', label: '节课' },
	{ value: '6', label: '个模块' },
	{ value: '4', label: '类典型故障' },
	{ value: '0', label: '特权依赖' },
]
</script>

<template>
	<div class="lp-hero-metrics">
		<ul class="lp-metrics">
			<li v-for="m in metrics" :key="m.label">
				<span class="lp-metrics__value">{{ m.value }}</span>
				<span class="lp-metrics__label">{{ m.label }}</span>
			</li>
		</ul>
	</div>
</template>

<style scoped>
.lp-hero-metrics {
	display: flex;
	justify-content: center;
	padding-inline: 24px;
	margin-block: clamp(1.5rem, 4vw, 2.5rem) 0;
}

.lp-metrics {
	display: flex;
	flex-wrap: wrap;
	justify-content: center;
	gap: clamp(1.5rem, 6vw, 3.5rem);
	margin: 0;
	padding: clamp(1.25rem, 3vw, 1.75rem) 0 0;
	list-style: none;
	border-top: 1px solid var(--lp-border);
	width: min(100%, 44rem);
}

.lp-metrics li {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 0.15rem;
}

.lp-metrics__value {
	font-size: clamp(1.5rem, 2vw + 0.8rem, 2rem);
	font-weight: 650;
	line-height: 1.1;
	letter-spacing: -0.02em;
	font-variant-numeric: tabular-nums;
	color: var(--lp-fg);
}

.lp-metrics__label {
	font-size: var(--sl-text-xs);
	color: var(--lp-muted);
}
</style>
