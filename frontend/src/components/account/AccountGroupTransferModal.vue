<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.groupTransfer.title')"
    width="normal"
    @close="emit('close')"
  >
    <form id="account-group-transfer-form" class="space-y-4" @submit.prevent="handleSubmit">
      <div class="rounded-lg bg-blue-50 p-4 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">
        {{ t('admin.accounts.groupTransfer.description') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.groupTransfer.sourceGroup') }}</label>
        <Select
          data-testid="group-transfer-source"
          :model-value="sourceGroupId ?? ''"
          :options="groupOptions"
          @update:model-value="(value) => sourceGroupId = normalizeGroupValue(value)"
        />
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.groupTransfer.targetGroup') }}</label>
        <Select
          data-testid="group-transfer-target"
          :model-value="targetGroupId ?? ''"
          :options="targetGroupOptions"
          @update:model-value="(value) => targetGroupId = normalizeGroupValue(value)"
        />
      </div>

      <div>
        <label class="input-label" for="group-transfer-count">
          {{ t('admin.accounts.groupTransfer.count') }}
        </label>
        <input
          id="group-transfer-count"
          v-model.number="count"
          type="number"
          min="1"
          step="1"
          class="input"
        />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="account-group-transfer-form"
          class="btn btn-primary"
          :disabled="submitting"
        >
          {{ submitting ? t('admin.accounts.groupTransfer.transferring') : t('admin.accounts.groupTransfer.submit') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup } from '@/types'

interface Props {
  show: boolean
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  transferred: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const submitting = ref(false)
const sourceGroupId = ref<number | null>(null)
const targetGroupId = ref<number | null>(null)
const count = ref(1)

const groupOptions = computed(() =>
  props.groups.map(group => ({
    value: group.id,
    label: group.name
  }))
)

const targetGroupOptions = computed(() =>
  props.groups
    .filter(group => group.id !== sourceGroupId.value)
    .map(group => ({
      value: group.id,
      label: group.name
    }))
)

const normalizeGroupValue = (value: string | number | boolean | null) => {
  if (value === '' || value === null || typeof value === 'boolean') return null
  return Number(value)
}

const resetForm = () => {
  sourceGroupId.value = null
  targetGroupId.value = null
  count.value = 1
  submitting.value = false
}

watch(
  () => props.show,
  (show) => {
    if (!show) {
      resetForm()
    }
  }
)

const handleSubmit = async () => {
  if (!sourceGroupId.value || !targetGroupId.value) {
    appStore.showError(t('admin.accounts.groupTransfer.selectGroups'))
    return
  }
  if (sourceGroupId.value === targetGroupId.value) {
    appStore.showError(t('admin.accounts.groupTransfer.groupsMustDiffer'))
    return
  }
  if (!count.value || count.value <= 0) {
    appStore.showError(t('admin.accounts.groupTransfer.invalidCount'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.accounts.transferAccountsByGroup({
      source_group_id: sourceGroupId.value,
      target_group_id: targetGroupId.value,
      count: count.value
    })
    appStore.showSuccess(
      t('admin.accounts.groupTransfer.success', { count: result.moved_count })
    )
    emit('transferred')
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.groupTransfer.failed'))
  } finally {
    submitting.value = false
  }
}
</script>
