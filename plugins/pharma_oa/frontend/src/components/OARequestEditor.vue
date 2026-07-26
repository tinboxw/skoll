<script setup lang="ts">
import { computed } from "vue";
import { t } from "../i18n";
import type { OAEditorForm } from "../oa-form";
import type { OARequestType } from "../types";

const form = defineModel<OAEditorForm>({ required: true });

const typeOptions = computed<Array<{ value: OARequestType; label: string }>>(() => [
  { value: "leave", label: t("leaveRequest") },
  { value: "expense", label: t("expenseRequest") },
  { value: "procurement", label: t("procurementRequest") },
  { value: "contract", label: t("contractRequest") },
  { value: "custom", label: t("customRequest") }
]);
</script>

<template>
  <el-form label-position="top" class="oa-editor" @submit.prevent>
    <section class="oa-form-section">
      <h3>{{ t("basicInformation") }}</h3>
      <div class="editor-grid">
        <el-form-item :label="t('requestType')" required>
          <el-select v-model="form.requestType">
            <el-option v-for="option in typeOptions" :key="option.value" :label="option.label" :value="option.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('requestTitle')" required>
          <el-input v-model="form.title" maxlength="200" show-word-limit clearable />
        </el-form-item>
        <el-form-item :label="t('approverId')" required>
          <el-input v-model="form.approverId" maxlength="128" clearable />
        </el-form-item>
        <el-form-item :label="t('approverName')" required>
          <el-input v-model="form.approverName" maxlength="200" clearable />
        </el-form-item>
        <el-form-item :label="t('requestDescription')" class="wide">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="5000" show-word-limit />
        </el-form-item>
      </div>
    </section>

    <section class="oa-form-section">
      <h3>{{ t("requestContent") }}</h3>
      <div v-if="form.requestType === 'leave'" class="editor-grid">
        <el-form-item :label="t('leaveType')" required>
          <el-select v-model="form.leaveType">
            <el-option :label="t('annualLeave')" value="annual" />
            <el-option :label="t('sickLeave')" value="sick" />
            <el-option :label="t('personalLeave')" value="personal" />
          </el-select>
        </el-form-item>
        <span aria-hidden="true" />
        <el-form-item :label="t('startDate')" required><el-date-picker v-model="form.startDate" type="date" value-format="YYYY-MM-DD" /></el-form-item>
        <el-form-item :label="t('endDate')" required><el-date-picker v-model="form.endDate" type="date" value-format="YYYY-MM-DD" /></el-form-item>
      </div>
      <div v-else-if="form.requestType === 'expense'" class="editor-grid">
        <el-form-item :label="t('amount')" required><el-input-number v-model="form.amount" :min="0" :precision="2" controls-position="right" /></el-form-item>
        <el-form-item :label="t('expenseCategory')" required>
          <el-select v-model="form.category">
            <el-option :label="t('travelExpense')" value="travel" />
            <el-option :label="t('officeExpense')" value="office" />
            <el-option :label="t('hospitalityExpense')" value="hospitality" />
          </el-select>
        </el-form-item>
      </div>
      <div v-else-if="form.requestType === 'procurement'" class="editor-grid">
        <el-form-item :label="t('amount')" required><el-input-number v-model="form.amount" :min="0" :precision="2" controls-position="right" /></el-form-item>
        <el-form-item :label="t('procurementPurpose')" required><el-input v-model="form.purpose" maxlength="500" clearable /></el-form-item>
      </div>
      <div v-else-if="form.requestType === 'contract'" class="editor-grid">
        <el-form-item :label="t('amount')" required><el-input-number v-model="form.amount" :min="0" :precision="2" controls-position="right" /></el-form-item>
        <el-form-item :label="t('counterparty')" required><el-input v-model="form.counterparty" maxlength="200" clearable /></el-form-item>
        <el-form-item :label="t('effectiveDate')" required><el-date-picker v-model="form.effectiveDate" type="date" value-format="YYYY-MM-DD" /></el-form-item>
      </div>
      <div v-else class="editor-grid">
        <el-form-item :label="t('customFormKey')" required><el-input v-model="form.formKey" maxlength="128" clearable /></el-form-item>
        <el-form-item :label="t('customContent')" class="wide">
          <el-input v-model="form.customContent" type="textarea" :rows="5" maxlength="5000" show-word-limit />
        </el-form-item>
      </div>
    </section>
  </el-form>
</template>
