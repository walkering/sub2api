<template>
  <BaseDialog
    :show="show"
    :title="t('admin.scheduledTests.groupDialogTitle')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="grid gap-3 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.scheduledTests.group') }}</label>
          <select
            v-model="selectedGroupId"
            class="input"
            :disabled="!!groupId"
          >
            <option :value="null">{{ t('admin.scheduledTests.selectGroup') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">
              {{ group.name }}
            </option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.scheduledTests.cronExpression') }}</label>
          <input
            v-model="createForm.cron_expression"
            type="text"
            class="input"
            placeholder="*/30 * * * *"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.scheduledTests.batchSize') }}</label>
          <input v-model.number="createForm.batch_size" type="number" min="1" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.scheduledTests.offsetSeconds') }}</label>
          <input v-model.number="createForm.offset" type="number" min="0" class="input" />
        </div>
        <div class="md:col-span-2">
          <label class="input-label">{{ t('admin.scheduledTests.modelOptional') }}</label>
          <input v-model="createForm.model_id" type="text" class="input" :placeholder="t('admin.scheduledTests.modelOptionalHint')" />
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-4">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <Toggle v-model="createForm.enabled" />
          {{ t('admin.scheduledTests.enabled') }}
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <Toggle v-model="createForm.auto_recover" />
          {{ t('admin.scheduledTests.autoRecover') }}
        </label>
      </div>

      <div class="flex justify-end">
        <button class="btn btn-primary" :disabled="saving" @click="handleCreate">
          {{ t('admin.scheduledTests.createGroupPlan') }}
        </button>
      </div>

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.scheduledTests.planList') }}
          </div>
          <button class="btn btn-secondary" :disabled="loading || !selectedGroupId" @click="loadPlans">
            {{ t('common.refresh') }}
          </button>
        </div>

        <div v-if="loading" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('common.loading') }}...
        </div>
        <div v-else-if="!selectedGroupId" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduledTests.selectGroupFirst') }}
        </div>
        <div v-else-if="plans.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduledTests.noPlans') }}
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="plan in plans"
            :key="plan.id"
            class="rounded-xl border border-gray-200 p-4 dark:border-dark-600"
          >
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-1">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {{ plan.cron_expression }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.scheduledTests.batchOffsetSummary', { batch: plan.batch_size, offset: plan.offset }) }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.scheduledTests.nextRun') }}: {{ formatDateTime(plan.next_run_at) }}
                </div>
                <div v-if="plan.last_run_at" class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.scheduledTests.lastRun') }}: {{ formatDateTime(plan.last_run_at) }}
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <Toggle :model-value="plan.enabled" @update:model-value="(value: boolean) => toggleEnabled(plan, value)" />
                  {{ t('admin.scheduledTests.enabled') }}
                </label>
                <button class="btn btn-secondary" @click="startEdit(plan)">
                  {{ t('common.edit') }}
                </button>
                <button class="btn btn-danger" :disabled="deletingId === plan.id" @click="removePlan(plan.id)">
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>

            <div v-if="editingPlanId === plan.id" class="mt-4 grid gap-3 border-t border-gray-200 pt-4 dark:border-dark-600 md:grid-cols-2">
              <div>
                <label class="input-label">{{ t('admin.scheduledTests.cronExpression') }}</label>
                <input v-model="editForm.cron_expression" type="text" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.scheduledTests.modelOptional') }}</label>
                <input v-model="editForm.model_id" type="text" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.scheduledTests.batchSize') }}</label>
                <input v-model.number="editForm.batch_size" type="number" min="1" class="input" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.scheduledTests.offsetSeconds') }}</label>
                <input v-model.number="editForm.offset" type="number" min="0" class="input" />
              </div>
              <div class="md:col-span-2 flex flex-wrap items-center gap-4">
                <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <Toggle v-model="editForm.enabled" />
                  {{ t('admin.scheduledTests.enabled') }}
                </label>
                <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <Toggle v-model="editForm.auto_recover" />
                  {{ t('admin.scheduledTests.autoRecover') }}
                </label>
              </div>
              <div class="md:col-span-2 flex justify-end gap-2">
                <button class="btn btn-secondary" @click="cancelEdit">{{ t('common.cancel') }}</button>
                <button class="btn btn-primary" :disabled="saving" @click="saveEdit(plan.id)">
                  {{ t('common.save') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, ScheduledTestPlan } from '@/types'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  groups: AdminGroup[]
  groupId?: number | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const plans = ref<ScheduledTestPlan[]>([])
const selectedGroupId = ref<number | null>(props.groupId ?? null)
const editingPlanId = ref<number | null>(null)

const createForm = reactive({
  model_id: '',
  cron_expression: '*/30 * * * *',
  batch_size: 5,
  offset: 30,
  enabled: true,
  auto_recover: false
})

const editForm = reactive({
  model_id: '',
  cron_expression: '',
  batch_size: 5,
  offset: 30,
  enabled: true,
  auto_recover: false
})

watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    selectedGroupId.value = props.groupId ?? selectedGroupId.value ?? props.groups[0]?.id ?? null
    editingPlanId.value = null
    await loadPlans()
  }
)

watch(
  () => props.groupId,
  (value) => {
    if (value) {
      selectedGroupId.value = value
    }
  }
)

watch(selectedGroupId, async () => {
  if (props.show) {
    editingPlanId.value = null
    await loadPlans()
  }
})

const validatePlanForm = (form: { cron_expression: string; batch_size: number; offset: number }) => {
  if (!selectedGroupId.value) return t('admin.scheduledTests.selectGroupFirst')
  if (!form.cron_expression.trim()) return t('admin.scheduledTests.cronRequired')
  if (!Number.isFinite(form.batch_size) || form.batch_size <= 0) return t('admin.scheduledTests.batchSizeInvalid')
  if (!Number.isFinite(form.offset) || form.offset < 0) return t('admin.scheduledTests.offsetInvalid')
  return ''
}

const loadPlans = async () => {
  if (!selectedGroupId.value) {
    plans.value = []
    return
  }
  loading.value = true
  try {
    plans.value = await adminAPI.scheduledTests.listByGroup(selectedGroupId.value)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.loadFailed'))
  } finally {
    loading.value = false
  }
}

const resetCreateForm = () => {
  createForm.model_id = ''
  createForm.cron_expression = '*/30 * * * *'
  createForm.batch_size = 5
  createForm.offset = 30
  createForm.enabled = true
  createForm.auto_recover = false
}

const handleCreate = async () => {
  const validationError = validatePlanForm(createForm)
  if (validationError) {
    appStore.showError(validationError)
    return
  }

  saving.value = true
  try {
    await adminAPI.scheduledTests.create({
      group_id: selectedGroupId.value!,
      model_id: createForm.model_id.trim() || undefined,
      cron_expression: createForm.cron_expression.trim(),
      batch_size: Number(createForm.batch_size),
      offset: Number(createForm.offset),
      enabled: createForm.enabled,
      auto_recover: createForm.auto_recover
    })
    appStore.showSuccess(t('admin.scheduledTests.planCreated'))
    resetCreateForm()
    await loadPlans()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.saveFailed'))
  } finally {
    saving.value = false
  }
}

const startEdit = (plan: ScheduledTestPlan) => {
  editingPlanId.value = plan.id
  editForm.model_id = plan.model_id || ''
  editForm.cron_expression = plan.cron_expression
  editForm.batch_size = plan.batch_size
  editForm.offset = plan.offset
  editForm.enabled = plan.enabled
  editForm.auto_recover = plan.auto_recover
}

const cancelEdit = () => {
  editingPlanId.value = null
}

const saveEdit = async (planId: number) => {
  const validationError = validatePlanForm(editForm)
  if (validationError) {
    appStore.showError(validationError)
    return
  }

  saving.value = true
  try {
    await adminAPI.scheduledTests.update(planId, {
      model_id: editForm.model_id,
      cron_expression: editForm.cron_expression.trim(),
      batch_size: Number(editForm.batch_size),
      offset: Number(editForm.offset),
      enabled: editForm.enabled,
      auto_recover: editForm.auto_recover
    })
    appStore.showSuccess(t('admin.scheduledTests.planUpdated'))
    editingPlanId.value = null
    await loadPlans()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.saveFailed'))
  } finally {
    saving.value = false
  }
}

const toggleEnabled = async (plan: ScheduledTestPlan, enabled: boolean) => {
  try {
    await adminAPI.scheduledTests.update(plan.id, { enabled })
    plan.enabled = enabled
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.saveFailed'))
  }
}

const removePlan = async (planId: number) => {
  deletingId.value = planId
  try {
    await adminAPI.scheduledTests.delete(planId)
    appStore.showSuccess(t('admin.scheduledTests.planDeleted'))
    await loadPlans()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.deleteFailed'))
  } finally {
    deletingId.value = null
  }
}

const formatDateTime = (value: string | null) => {
  if (!value) return '—'
  return new Date(value).toLocaleString()
}
</script>
