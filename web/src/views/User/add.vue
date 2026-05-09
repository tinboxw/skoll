<script setup lang="ts">
import { ref } from "vue";

import { useI18n } from "../../i18n";

const username = ref("");
const role = ref("viewer");
const { t } = useI18n();

function submit(): void {
	// temporary milestone scaffold.
	alert(`${t("user.createDone")}: ${username.value} (${role.value})`);
}
</script>

<template>
	<section>
		<h2>{{ t("page.userAdd") }}</h2>
		<form @submit.prevent="submit">
			<label>
				<span>{{ t("user.username") }}</span>
				<input v-model="username" type="text" required />
			</label>
			<label>
				<span>{{ t("role.field") }}</span>
				<select v-model="role">
					<option value="viewer">{{ t("role.viewer") }}</option>
					<option value="editor">{{ t("role.editor") }}</option>
					<option value="super_admin">{{ t("role.superAdmin") }}</option>
				</select>
			</label>
			<button type="submit">{{ t("user.create") }}</button>
		</form>
	</section>
</template>

<style scoped>
form {
	display: grid;
	gap: 10px;
	max-width: 320px;
}

label {
	display: grid;
	gap: 6px;
}

input,
select,
button {
	padding: 8px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
}

button {
	background: var(--color-primary);
	color: var(--color-on-primary);
	border: none;
	cursor: pointer;
}

button:hover {
	background: var(--color-primary-strong);
}
</style>

