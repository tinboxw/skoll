<template>
	<main class="page">
		<header class="hero">
			<p class="eyebrow">Separated Architecture Demo</p>
			<h1>Aurora Forge Child Application</h1>
			<p>
				This demo represents a realistic child-app homepage: company narrative, backend-powered stats,
				and recommendation workflows served from separated plugin frontend artifacts.
			</p>
			<div class="actions">
				<button type="button" @click="loadOverview">Load Overview</button>
				<button type="button" class="ghost" @click="loadRecommendations">Load Recommendations</button>
			</div>
		</header>

		<section class="stats-grid">
			<article v-for="item in overviewStats" :key="item.label" class="stat-card">
				<span>{{ item.label }}</span>
				<strong>{{ item.value }}</strong>
				<p>{{ item.change }}</p>
			</article>
		</section>

		<section class="content-grid">
			<article class="panel">
				<h2>Focus Areas</h2>
				<ul class="focus-list">
					<li v-for="item in focusAreas" :key="item.title">
						<strong>{{ item.title }}</strong>
						<p>{{ item.summary }}</p>
						<small>{{ item.region }} · {{ item.stage }}</small>
					</li>
				</ul>
			</article>
			<article class="panel">
				<h2>Output</h2>
				<pre>{{ output }}</pre>
			</article>
		</section>
	</main>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

type MetricCard = {
	label: string;
	value: string;
	change: string;
};

type FocusArea = {
	title: string;
	summary: string;
	region: string;
	stage: string;
};

type OverviewPayload = {
	stats?: MetricCard[];
	focusAreas?: FocusArea[];
};

const output = ref("Click one action to fetch plugin runtime data.");
const latestOverview = ref<OverviewPayload | null>(null);

const fallbackStats: MetricCard[] = [
	{ label: "Pipeline Value", value: "$12.4M", change: "+18% quarter-on-quarter" },
	{ label: "Live Regions", value: "06", change: "APAC launch unlocked" },
	{ label: "Studio Utilization", value: "92%", change: "Two new lab sprints added" }
];

const fallbackFocus: FocusArea[] = [
	{
		title: "Retail Atelier",
		summary: "Convert flagship stores into low-friction pickup lounges with guided discovery walls.",
		region: "Shanghai",
		stage: "prototype"
	},
	{
		title: "Field Service Kit",
		summary: "Bundle diagnostics, repair scripts, and concierge messaging into one tablet workflow.",
		region: "Shenzhen",
		stage: "pilot"
	},
	{
		title: "Executive Briefing Deck",
		summary: "Turn weekly platform signals into a board-ready narrative with market and risk framing.",
		region: "Global",
		stage: "active"
	}
];

const overviewStats = computed(() => {
	return latestOverview.value?.stats && latestOverview.value.stats.length > 0
		? latestOverview.value.stats
		: fallbackStats;
});

const focusAreas = computed(() => {
	return latestOverview.value?.focusAreas && latestOverview.value.focusAreas.length > 0
		? latestOverview.value.focusAreas
		: fallbackFocus;
});

async function requestJSON(url: string): Promise<unknown> {
	const resp = await fetch(url);
	if (!resp.ok) {
		throw new Error(`request failed: ${resp.status}`);
	}
	return await resp.json();
}

async function loadOverview(): Promise<void> {
	output.value = "Loading /demo-separated/overview ...";
	try {
		const payload = (await requestJSON("/demo-separated/overview?plugin_id=demo")) as OverviewPayload;
		latestOverview.value = payload;
		output.value = JSON.stringify(payload, null, 2);
	} catch (error) {
		output.value = String(error);
	}
}

async function loadRecommendations(): Promise<void> {
	output.value = "Loading /demo-separated/recommendations ...";
	try {
		const payload = await requestJSON("/demo-separated/recommendations?active_users=84&error_count=1&rpm=260");
		output.value = JSON.stringify(payload, null, 2);
	} catch (error) {
		output.value = String(error);
	}
}
</script>

<style scoped>
:global(body) {
	margin: 0;
	font-family: "Avenir Next", "Segoe UI", sans-serif;
	background: radial-gradient(circle at 18% 10%, #d9f7ef 0, transparent 34%), #f4f5f1;
	color: #1a242b;
}

.page {
	max-width: 1060px;
	margin: 0 auto;
	padding: 24px 16px 34px;
}

.hero {
	border: 1px solid #d3dbc9;
	border-radius: 16px;
	padding: 18px;
	background: linear-gradient(130deg, #fcfdf7, #eef7f2);
}

.eyebrow {
	margin: 0;
	font-size: 12px;
	letter-spacing: 0.08em;
	text-transform: uppercase;
	color: #55636b;
}

h1 {
	margin: 8px 0;
	font-size: clamp(1.8rem, 3.6vw, 2.8rem);
}

.hero p {
	margin: 0;
	color: #56646b;
}

.actions {
	margin-top: 13px;
	display: flex;
	gap: 8px;
	flex-wrap: wrap;
}

button {
	border: 0;
	border-radius: 999px;
	padding: 10px 15px;
	background: #0f766e;
	color: #fff;
	font-weight: 600;
	cursor: pointer;
}

button:hover {
	background: #0d5e59;
}

button.ghost {
	background: #e2efeb;
	color: #162429;
}

.stats-grid {
	margin-top: 14px;
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
	gap: 10px;
}

.stat-card {
	border: 1px solid #d4dccf;
	border-radius: 12px;
	background: #fff;
	padding: 11px;
}

.stat-card span {
	font-size: 13px;
	color: #5b6970;
}

.stat-card strong {
	display: block;
	margin-top: 5px;
	font-size: 1.35rem;
}

.stat-card p {
	margin: 6px 0 0;
	font-size: 12px;
	color: #5c6a71;
}

.content-grid {
	margin-top: 13px;
	display: grid;
	grid-template-columns: 1.1fr 0.9fr;
	gap: 10px;
}

.panel {
	border: 1px solid #d4dccf;
	border-radius: 12px;
	padding: 11px;
	background: #fff;
}

.panel h2 {
	margin: 0 0 10px;
	font-size: 1rem;
}

.focus-list {
	margin: 0;
	padding: 0;
	list-style: none;
	display: grid;
	gap: 8px;
}

.focus-list li {
	border: 1px dashed #cbd4c4;
	border-radius: 9px;
	padding: 8px;
}

.focus-list p {
	margin: 4px 0;
	font-size: 13px;
	color: #5a686f;
}

.focus-list small {
	color: #64747a;
}

pre {
	margin: 0;
	min-height: 240px;
	border-radius: 9px;
	background: #13212a;
	color: #dde9f2;
	padding: 10px;
	overflow: auto;
	white-space: pre-wrap;
}

@media (max-width: 860px) {
	.content-grid {
		grid-template-columns: 1fr;
	}
}
</style>
