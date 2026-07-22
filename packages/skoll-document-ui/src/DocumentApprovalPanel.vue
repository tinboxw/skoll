<script setup lang="ts">
import { Check, RotateCcw, UserRoundCog, X } from "lucide-vue-next";
import { computed, ref, watch } from "vue";

import { documentKitMessages } from "./messages";
import type { DocumentActionRequest, DocumentKitMessages, DocumentLocale, WorkflowActor, WorkflowInstance, WorkflowTask } from "./types";

export type DocumentApprovalAction = "approve" | "reject" | "withdraw" | "delegate" | "cancel";

const props = withDefaults(defineProps<{
	workflow: WorkflowInstance;
	availableActions?: DocumentApprovalAction[];
	delegateOptions?: WorkflowActor[];
	busy?: boolean;
	disabled?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
}>(), {
	availableActions: () => [],
	delegateOptions: () => [],
	busy: false,
	disabled: false,
	locale: "zh-CN",
	messages: () => ({})
});

const emit = defineEmits<{
	(e: "action", request: DocumentActionRequest): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
const selectedAction = ref<DocumentApprovalAction | "">("");
const comment = ref("");
const targetID = ref("");
const formError = ref("");
const pendingTasks = computed(() => props.workflow.tasks.filter((task) => task.status === "pending"));
const currentTask = computed<WorkflowTask | undefined>(() => pendingTasks.value[0]);
const actionRequiresComment = computed(() => selectedAction.value === "reject");

watch(selectedAction, () => {
	formError.value = "";
	if (selectedAction.value !== "delegate") targetID.value = "";
});

function choose(action: DocumentApprovalAction): void {
	selectedAction.value = action;
}

function submit(): void {
	formError.value = "";
	if (!selectedAction.value) return;
	if (actionRequiresComment.value && !comment.value.trim()) {
		formError.value = text.value.commentRequired;
		return;
	}
	const target = props.delegateOptions.find((actor) => actor.id === targetID.value);
	if (selectedAction.value === "delegate" && !target) {
		formError.value = text.value.delegateTargetRequired;
		return;
	}
	emit("action", {
		action: selectedAction.value,
		comment: comment.value.trim(),
		taskId: ["approve", "reject", "delegate"].includes(selectedAction.value) ? currentTask.value?.id : undefined,
		target
	});
}

function iconFor(action: DocumentApprovalAction) {
	if (action === "approve") return Check;
	if (action === "delegate") return UserRoundCog;
	if (action === "withdraw") return RotateCcw;
	return X;
}

function labelFor(action: DocumentApprovalAction): string {
	return text.value[action];
}
</script>

<template>
	<section class="document-approval" :aria-label="text.workflow">
		<header class="document-approval__header">
			<div>
				<h3>{{ text.workflow }}</h3>
				<p>{{ workflow.currentNode || workflow.status }}</p>
			</div>
			<el-tag :type="workflow.status === 'approved' ? 'success' : workflow.status === 'rejected' ? 'danger' : 'warning'">
				{{ workflow.status }}
			</el-tag>
		</header>

		<div v-if="pendingTasks.length" class="document-approval__tasks">
			<div v-for="task in pendingTasks" :key="task.id" class="document-approval__task">
				<span>{{ task.assignee.name || task.assignee.id }}</span>
				<small>{{ task.status }}</small>
			</div>
		</div>

		<div v-if="availableActions.length" class="document-approval__actions" role="group" :aria-label="text.actions">
			<el-button
				v-for="action in availableActions"
				:key="action"
				:type="action === 'approve' ? 'success' : action === 'reject' || action === 'cancel' ? 'danger' : 'default'"
				:plain="selectedAction !== action"
				:disabled="disabled || busy"
				@click="choose(action)"
			>
				<component :is="iconFor(action)" :size="16" />{{ labelFor(action) }}
			</el-button>
		</div>

		<div v-if="selectedAction" class="document-approval__decision">
			<el-form-item v-if="selectedAction === 'delegate'" :label="text.delegateTarget" :error="selectedAction === 'delegate' ? formError : ''">
				<el-select v-model="targetID" filterable :disabled="busy">
					<el-option v-for="actor in delegateOptions" :key="actor.id" :value="actor.id" :label="actor.name || actor.id" />
				</el-select>
			</el-form-item>
			<el-form-item :label="text.comment" :required="actionRequiresComment" :error="selectedAction !== 'delegate' ? formError : ''">
				<el-input v-model="comment" type="textarea" :rows="3" maxlength="4096" show-word-limit :disabled="busy" />
			</el-form-item>
			<div class="document-approval__submit">
				<el-button @click="selectedAction = ''">{{ text.cancel }}</el-button>
				<el-button type="primary" :loading="busy" @click="submit">{{ labelFor(selectedAction) }}</el-button>
			</div>
		</div>
	</section>
</template>

<style scoped>
.document-approval {
	display: grid;
	gap: 14px;
}

.document-approval__header,
.document-approval__task,
.document-approval__submit {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 10px;
}

.document-approval__header h3,
.document-approval__header p {
	margin: 0;
}

.document-approval__header h3 {
	font-size: 1rem;
}

.document-approval__header p,
.document-approval__task small {
	color: var(--color-text-muted);
}

.document-approval__tasks {
	display: grid;
	gap: 6px;
	padding: 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	background: var(--color-surface-soft);
}

.document-approval__actions {
	display: flex;
	gap: 8px;
	flex-wrap: wrap;
}

.document-approval__decision {
	padding: 12px;
	border-left: 3px solid var(--color-primary);
	background: var(--color-surface-soft);
}

.document-approval__decision :deep(.el-select) {
	width: 100%;
}

.document-approval__submit {
	justify-content: flex-end;
}
</style>
