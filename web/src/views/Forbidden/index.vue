<script setup lang="ts">
import { useRouter } from "vue-router";

import { PageShell, StateBlock } from "../../components/Common";
import { useI18n } from "../../i18n";

const router = useRouter();
const { t } = useI18n();

async function backToDashboard(): Promise<void> {
	await router.replace("/skoll/dashboard");
}
</script>

<template>
	<PageShell :title="t('state.noPermission')" :description="t('state.noPagePermission')">
		<StateBlock type="forbidden" :title="t('state.noPermission')" :description="t('state.noPagePermission')">
			<template #actions>
				<el-button type="primary" @click="backToDashboard">{{ t("page.dashboard") }}</el-button>
			</template>
		</StateBlock>
	</PageShell>
</template>
