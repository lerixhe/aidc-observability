<script setup lang="ts">
/**
 * 首页主体内容（学习路径 / 三段说明 / 收尾 CTA）。
 * 与 hero 的版本徽章、指标条一起构成首页，样式全部以 .lp- 前缀限定作用域。
 */

import { withBase } from 'vitepress'

const modules = [
	{
		id: '开篇',
		title: '认识这门课',
		desc: '从「指标全绿、任务却慢」的现场切入：这门课解决什么、要什么前置，再用五分钟把实验环境跑起来。',
		href: '/00-intro/00-what-problem',
	},
	{
		id: '第一篇',
		title: '两套世界观',
		desc: '流量模型、丢包语义、故障代价、遥测粒度，四条默认假设全要换。再往下是 RDMA 的内核旁路、RoCEv2 与 DCB、自适应路由。',
		href: '/01-paradigm/02-why-different',
	},
	{
		id: '第二篇',
		title: '不丢包是买来的',
		desc: 'PFC 用逐跳反压换不丢包，代价是队头阻塞与拥塞上游化；后面的账交给 ECN 和 DCQCN。最后看四种典型故障在现场长什么样。',
		href: '/02-lossless/05-pfc',
	},
	{
		id: '第三篇',
		title: '流量与连线',
		desc: 'AllReduce 把流量矩阵变成已知量；Clos 与多 Rail 决定拓扑必须进数据模型；超节点又把一半流量收回私有网络。',
		href: '/03-traffic-topo/08-nccl',
	},
	{
		id: '第四篇',
		title: '把现场变成数据',
		desc: 'gNMI 流式订阅、INT 逐跳元数据、光模块劣化的早期信号，以及从设备到可查询的那条数据管道。',
		href: '/04-telemetry/11-gnmi',
	},
	{
		id: '第五篇',
		title: '从数据到结论',
		desc: '指标口径怎么定、SLO 怎么算、告警怎么收敛、时序库怎么选，一次故障的 MTTR 卡在哪一段。',
		href: '/05-analysis/15-metrics-slo',
	},
	{
		id: '第六篇',
		title: '路线与收口',
		desc: 'IB 与 RoCE 的选择题，UEC 想改什么，一条端到端排障主线，最后一张能自己打分的能力表。',
		href: '/06-selection/20-ib-vs-roce',
	},
]

const facts = [
	{
		kicker: '讲什么',
		title: '机制后面总跟着一句信号',
		body: '每个机制都要回答「它坏了会在哪个计数器、哪根曲线上留下痕迹」。回答不了这句的，不进正文。',
	},
	{
		kicker: '适合谁',
		title: '正在接手智算集群的人',
		body: '做过传统 IDC 或云网络的人、写可观测与 AIOps 平台的人，以及要判断这套无损体系到底怎么转的基础设施同学。',
	},
	{
		kicker: '怎么读',
		title: '前两篇是地基',
		body: '从第三篇起跳读，后面的判读会变成背结论。标了 lab 的讲可以边跑边看，L1 层只要 Go，不要特权。',
	},
]
</script>

<template>
	<div class="lp-content">
		<section class="lp-section" aria-labelledby="lp-path">
			<header class="lp-head">
				<p class="lp-kicker">学习路径</p>
				<h2 id="lp-path">七个模块，一条因果链</h2>
				<p class="lp-lede">从「网络为什么变慢」问到「哪颗光模块在劣化」，每一层的结论都是下一层的起点。</p>
			</header>

			<ol class="lp-cards">
				<li v-for="m in modules" :key="m.id">
					<a class="lp-card" :href="withBase(m.href)">
						<span class="lp-card__top">
							<span class="lp-card__id">{{ m.id }}</span>
						</span>
						<span class="lp-card__title">{{ m.title }}</span>
						<span class="lp-card__desc">{{ m.desc }}</span>
						<span class="lp-card__link">
							进入模块
							<svg viewBox="0 0 16 16" width="13" height="13" aria-hidden="true">
								<path
									d="M3 8h9M8.5 4.5 12 8l-3.5 3.5"
									fill="none"
									stroke="currentColor"
									stroke-width="1.6"
									stroke-linecap="round"
									stroke-linejoin="round"
								/>
							</svg>
						</span>
					</a>
				</li>
			</ol>
		</section>

		<section class="lp-section lp-facts" aria-label="课程说明">
			<article v-for="f in facts" :key="f.title">
				<p class="lp-kicker">{{ f.kicker }}</p>
				<h3>{{ f.title }}</h3>
				<p class="lp-body">{{ f.body }}</p>
			</article>
		</section>

		<section class="lp-cta" aria-labelledby="lp-start">
			<div class="lp-cta__text">
				<h2 id="lp-start">先跑通环境，再谈判读</h2>
				<p>第一讲讲清问题在哪，第二讲把实验环境跑起来。后面每一讲都默认这两件事已经成立。</p>
			</div>
			<a class="lp-cta__btn" :href="withBase('/00-intro/00-what-problem')">
				进入第一课
				<svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
					<path
						d="M3 8h9M8.5 4.5 12 8l-3.5 3.5"
						fill="none"
						stroke="currentColor"
						stroke-width="1.6"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
				</svg>
			</a>
		</section>

		<p class="lp-note">
			实验是教学级模型，不和硬件行为等价，每个 lab 的 README 都写了边界与真机验证路径。机制类断言尽量给出公开可查的来源，见各讲末尾的资料清单。
		</p>
	</div>
</template>

<style scoped>
.lp-content {
	display: flex;
	flex-direction: column;
	gap: clamp(3rem, 7vw, 4.5rem);
}

.lp-section {
	display: flex;
	flex-direction: column;
	gap: 1.75rem;
}

.lp-head {
	display: flex;
	flex-direction: column;
	gap: 0.6rem;
	max-width: 46rem;
}

.lp-kicker {
	margin: 0;
	font-size: var(--sl-text-xs);
	font-weight: 600;
	letter-spacing: 0.14em;
	text-transform: uppercase;
	color: var(--lp-accent);
}

h2 {
	margin: 0;
	padding: 0;
	border-top: none;
	font-size: clamp(1.35rem, 1.4vw + 0.9rem, 1.75rem);
	font-weight: 650;
	letter-spacing: -0.015em;
	line-height: 1.25;
	color: var(--lp-fg);
}

.lp-lede {
	margin: 0;
	color: var(--lp-muted);
	font-size: var(--sl-text-base);
	line-height: 1.7;
}

/* ---------- 模块卡片 ---------- */

.lp-cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr));
	gap: 1rem;
	margin: 0;
	padding: 0;
	list-style: none;
}

.lp-card {
	position: relative;
	display: flex;
	flex-direction: column;
	gap: 0.7rem;
	height: 100%;
	padding: 1.35rem 1.35rem 1.15rem;
	border: 1px solid var(--lp-border);
	border-radius: var(--lp-radius);
	background: var(--lp-surface);
	text-decoration: none;
	transition:
		transform 0.2s ease,
		border-color 0.2s ease,
		box-shadow 0.2s ease,
		background-color 0.2s ease;
}

/* hover 时顶部的一道极细高光 */
.lp-card::before {
	content: '';
	position: absolute;
	inset-inline: 1.35rem;
	top: -1px;
	height: 1px;
	background: linear-gradient(90deg, transparent, var(--lp-accent), transparent);
	opacity: 0;
	transition: opacity 0.2s ease;
}

.lp-card:hover {
	transform: translateY(-3px);
	border-color: var(--lp-border-strong);
	box-shadow: var(--lp-shadow);
}

.lp-card:hover::before {
	opacity: 0.9;
}

.lp-card__top {
	display: flex;
	align-items: center;
	justify-content: space-between;
}

.lp-card__id {
	font-family: var(--sl-font-mono);
	font-size: var(--sl-text-xs);
	font-weight: 600;
	letter-spacing: 0.08em;
	color: var(--lp-accent);
}

.lp-card__title {
	font-size: var(--sl-text-lg);
	font-weight: 650;
	line-height: 1.35;
	letter-spacing: -0.01em;
	color: var(--lp-fg);
}

.lp-card__desc {
	font-size: var(--sl-text-sm);
	line-height: 1.7;
	color: var(--lp-muted);
}

.lp-card__link {
	display: inline-flex;
	align-items: center;
	gap: 0.35rem;
	margin-block-start: auto;
	padding-block-start: 0.35rem;
	font-size: var(--sl-text-xs);
	font-weight: 600;
	color: var(--lp-fg);
	opacity: 0.55;
	transition: opacity 0.2s ease;
}

.lp-card__link svg {
	transition: transform 0.2s ease;
}

.lp-card:hover .lp-card__link {
	opacity: 1;
}

.lp-card:hover .lp-card__link svg {
	transform: translateX(3px);
}

/* ---------- 三段说明 ---------- */

.lp-facts {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: clamp(1.5rem, 4vw, 2.5rem);
	padding-block-start: clamp(1.5rem, 4vw, 2.25rem);
	border-top: 1px solid var(--lp-border);
}

.lp-facts article {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

/* 三项固定三列；窄屏堆成一列，别让第三项单独落到第二行 */
@media (max-width: 48rem) {
	.lp-facts {
		grid-template-columns: 1fr;
	}
}

.lp-facts h3 {
	margin: 0;
	padding: 0;
	border-top: none;
	font-size: var(--sl-text-base);
	font-weight: 650;
	letter-spacing: -0.01em;
	color: var(--lp-fg);
}

.lp-body {
	margin: 0;
	font-size: var(--sl-text-sm);
	line-height: 1.75;
	color: var(--lp-muted);
}

/* ---------- 收尾 CTA ---------- */

.lp-cta {
	position: relative;
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: space-between;
	gap: 1.5rem;
	padding: clamp(1.5rem, 4vw, 2.25rem);
	border: 1px solid var(--lp-border);
	border-radius: calc(var(--lp-radius) + 6px);
	background:
		radial-gradient(28rem 16rem at 8% 0%, var(--lp-glow-a), transparent 70%),
		var(--lp-surface);
	overflow: hidden;
}

.lp-cta__text {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
	max-width: 34rem;
}

.lp-cta__text p {
	margin: 0;
	font-size: var(--sl-text-sm);
	line-height: 1.7;
	color: var(--lp-muted);
}

.lp-cta__btn {
	display: inline-flex;
	align-items: center;
	gap: 0.45rem;
	padding: 0.7rem 1.35rem;
	border-radius: 999px;
	background: var(--lp-fg);
	color: var(--lp-surface-solid);
	font-size: var(--sl-text-sm);
	font-weight: 600;
	text-decoration: none;
	white-space: nowrap;
	transition:
		transform 0.18s ease,
		box-shadow 0.18s ease;
}

.lp-cta__btn:hover {
	transform: translateY(-1px);
	box-shadow: 0 14px 30px -16px hsl(220 60% 4% / 0.7);
}

.lp-cta__btn svg {
	transition: transform 0.18s ease;
}

.lp-cta__btn:hover svg {
	transform: translateX(3px);
}

.lp-note {
	margin: 0;
	padding-block-start: 1.25rem;
	border-top: 1px solid var(--lp-border);
	font-size: var(--sl-text-xs);
	line-height: 1.7;
	color: var(--lp-muted);
	opacity: 0.8;
}

@media (prefers-reduced-motion: reduce) {
	.lp-card,
	.lp-card::before,
	.lp-card__link,
	.lp-card__link svg,
	.lp-cta__btn,
	.lp-cta__btn svg {
		transition: none;
	}
}
</style>
