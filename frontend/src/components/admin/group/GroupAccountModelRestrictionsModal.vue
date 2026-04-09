<template>
  <BaseDialog
    :show="show"
    :title="t('admin.groups.accountModelRestrictions.title')"
    width="wide"
    @close="handleClose"
  >
    <div v-if="group" class="space-y-5">
      <div class="flex flex-wrap items-center gap-3 rounded-lg bg-gray-50 px-4 py-3 text-sm dark:bg-dark-700">
        <span class="font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
        <span class="text-gray-400">|</span>
        <span class="text-gray-600 dark:text-gray-400">
          {{ t('admin.groups.platforms.' + group.platform) }}
        </span>
        <template v-if="group.account_count !== undefined">
          <span class="text-gray-400">|</span>
          <span class="text-gray-600 dark:text-gray-400">
            {{ t('admin.groups.accountModelRestrictions.targetCount', { count: group.account_count || 0 }) }}
          </span>
        </template>
      </div>

      <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
        <p class="text-sm text-amber-700 dark:text-amber-400">
          <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
          {{ t('admin.groups.accountModelRestrictions.impactScope') }}
        </p>
      </div>

      <div v-if="group.platform === 'openai'" class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
        <p class="text-sm text-blue-700 dark:text-blue-400">
          <Icon name="infoCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
          {{ t('admin.groups.accountModelRestrictions.openaiPassthroughHint') }}
        </p>
      </div>

      <form id="group-account-model-restrictions-form" class="space-y-5" @submit.prevent="handleSubmit">
        <div>
          <div class="mb-4 flex gap-2">
            <button
              id="group-account-model-restriction-mode-whitelist"
              type="button"
              :class="[
                'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
                modelRestrictionMode === 'whitelist'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
              @click="modelRestrictionMode = 'whitelist'"
            >
              {{ t('admin.accounts.modelWhitelist') }}
            </button>
            <button
              id="group-account-model-restriction-mode-mapping"
              type="button"
              :class="[
                'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
                modelRestrictionMode === 'mapping'
                  ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
              @click="modelRestrictionMode = 'mapping'"
            >
              {{ t('admin.accounts.modelMapping') }}
            </button>
          </div>

          <div v-if="modelRestrictionMode === 'whitelist'" class="space-y-3">
            <div class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
              <p class="text-xs text-blue-700 dark:text-blue-400">
                {{ t('admin.groups.accountModelRestrictions.emptyWhitelistHint') }}
              </p>
            </div>
            <ModelWhitelistSelector v-model="allowedModels" :platforms="[group.platform]" />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.selectedModels', { count: allowedModels.length }) }}
            </p>
          </div>

          <div v-else class="space-y-3">
            <div class="rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
              <p class="text-xs text-purple-700 dark:text-purple-400">
                {{ t('admin.groups.accountModelRestrictions.mappingHint') }}
              </p>
            </div>

            <div v-if="modelMappings.length > 0" class="space-y-2">
              <div
                v-for="(mapping, index) in modelMappings"
                :key="index"
                class="flex items-center gap-2"
              >
                <input
                  v-model="mapping.from"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.accounts.requestModel')"
                />
                <Icon name="arrowRight" size="sm" class="text-gray-400" />
                <input
                  v-model="mapping.to"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.accounts.actualModel')"
                />
                <button
                  type="button"
                  class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                  @click="removeModelMapping(index)"
                >
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>

            <button
              type="button"
              class="w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
              @click="addModelMapping"
            >
              <Icon name="plus" size="sm" class="mr-1 inline" />
              {{ t('admin.accounts.addMapping') }}
            </button>

            <div class="flex flex-wrap gap-2">
              <button
                v-for="preset in presets"
                :key="preset.label"
                type="button"
                :class="['rounded-lg px-3 py-1 text-xs transition-colors', preset.color]"
                @click="addPresetMapping(preset.from, preset.to)"
              >
                + {{ preset.label }}
              </button>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 border-t border-gray-200 pt-4 dark:border-dark-600">
          <button type="button" class="btn btn-secondary" @click="handleClose">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            class="btn btn-primary"
            :disabled="submitting"
          >
            <Icon v-if="submitting" name="refresh" size="sm" class="mr-2 animate-spin" />
            {{ submitting ? t('admin.groups.accountModelRestrictions.submitting') : t('admin.groups.accountModelRestrictions.submit') }}
          </button>
        </div>
      </form>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelWhitelistSelector from '@/components/account/ModelWhitelistSelector.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  buildModelMappingObject as buildModelMappingPayload,
  getPresetMappingsByPlatform
} from '@/composables/useModelWhitelist'

interface Props {
  show: boolean
  group: AdminGroup | null
}

interface ModelMapping {
  from: string
  to: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  updated: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const submitting = ref(false)
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const modelMappings = ref<ModelMapping[]>([])

const presets = computed(() => {
  if (!props.group) {
    return []
  }
  return getPresetMappingsByPlatform(props.group.platform)
})

const resetForm = () => {
  submitting.value = false
  modelRestrictionMode.value = 'whitelist'
  allowedModels.value = []
  modelMappings.value = []
}

watch(() => props.show, () => {
  resetForm()
})
watch(() => props.group?.id, () => {
  resetForm()
})

const addModelMapping = () => {
  modelMappings.value.push({ from: '', to: '' })
}

const removeModelMapping = (index: number) => {
  modelMappings.value.splice(index, 1)
}

const addPresetMapping = (from: string, to: string) => {
  if (modelMappings.value.some((item) => item.from === from)) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const buildModelMapping = (): Record<string, string> => {
  if (modelRestrictionMode.value === 'whitelist') {
    const mapping: Record<string, string> = {}
    for (const model of allowedModels.value) {
      mapping[model] = model
    }
    return mapping
  }

  return buildModelMappingPayload(
    modelRestrictionMode.value,
    allowedModels.value,
    modelMappings.value
  ) ?? {}
}

const handleClose = () => {
  resetForm()
  emit('close')
}

const handleSubmit = async () => {
  if (!props.group) {
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.groups.updateAccountModelRestrictions(props.group.id, {
      credentials: {
        model_mapping: buildModelMapping()
      }
    })
    appStore.showSuccess(
      t('admin.groups.accountModelRestrictions.updated', {
        success: result.success,
        failed: result.failed
      })
    )
    emit('updated')
    handleClose()
  } catch (error: any) {
    appStore.showError(
      error?.response?.data?.detail || t('admin.groups.accountModelRestrictions.failed')
    )
  } finally {
    submitting.value = false
  }
}
</script>
