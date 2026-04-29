<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.applyGroups.title')"
    width="normal"
    @close="handleClose"
  >
    <div v-if="proxy" class="space-y-5">
      <div class="rounded-lg bg-gray-50 px-4 py-3 text-sm dark:bg-dark-700">
        <div class="font-medium text-gray-900 dark:text-white">{{ proxy.name }}</div>
        <div class="mt-1 text-gray-600 dark:text-gray-400">
          {{ t('admin.proxies.applyGroups.proxyLabel') }}: {{ proxy.protocol }}://{{ proxy.host }}:{{ proxy.port }}
        </div>
      </div>

      <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
        <p class="text-sm text-amber-700 dark:text-amber-400">
          <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
          {{ t('admin.proxies.applyGroups.impactScope') }}
        </p>
      </div>

      <div v-if="groups.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400">
        {{ t('admin.proxies.applyGroups.noGroups') }}
      </div>

      <form v-else id="proxy-apply-groups-form" class="space-y-4" @submit.prevent="handleSubmit">
        <div class="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
          <span>{{ t('admin.proxies.applyGroups.selectedCount', { count: selectedGroupIds.length }) }}</span>
          <span>{{ t('admin.groups.accountsCount', { count: totalAccountCount }) }}</span>
        </div>

        <div class="max-h-80 space-y-2 overflow-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600">
          <label
            v-for="group in groups"
            :key="group.id"
            class="flex cursor-pointer items-start gap-3 rounded-lg px-3 py-2 transition-colors hover:bg-gray-50 dark:hover:bg-dark-700"
          >
            <input
              v-model="selectedGroupIds"
              type="checkbox"
              class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :value="group.id"
            />
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
                <span class="badge badge-gray">{{ t(`admin.groups.platforms.${group.platform}`) }}</span>
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.proxies.applyGroups.accountCount', { count: group.account_count || 0 }) }}
              </div>
            </div>
          </label>
        </div>
      </form>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="proxy-apply-groups-form"
          class="btn btn-primary"
          :disabled="submitting || !proxy || selectedGroupIds.length === 0 || groups.length === 0"
        >
          <Icon v-if="submitting" name="refresh" size="sm" class="mr-2 animate-spin" />
          {{ submitting ? t('admin.proxies.applyGroups.submitting') : t('admin.proxies.applyGroups.submit') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, Proxy } from '@/types'

interface Props {
  show: boolean
  proxy: Proxy | null
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  updated: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const submitting = ref(false)
const selectedGroupIds = ref<number[]>([])

const totalAccountCount = computed(() => {
  return props.groups
    .filter((group) => selectedGroupIds.value.includes(group.id))
    .reduce((sum, group) => sum + (group.account_count || 0), 0)
})

const resetForm = () => {
  submitting.value = false
  selectedGroupIds.value = []
}

watch(() => props.show, resetForm)
watch(() => props.proxy?.id, resetForm)

const handleClose = () => {
  resetForm()
  emit('close')
}

const handleSubmit = async () => {
  if (!props.proxy) {
    return
  }
  if (selectedGroupIds.value.length === 0) {
    appStore.showError(t('admin.proxies.applyGroups.noSelection'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.proxies.applyToGroups(props.proxy.id, {
      group_ids: selectedGroupIds.value
    })
    appStore.showSuccess(
      t('admin.proxies.applyGroups.updated', {
        success: result.success,
        failed: result.failed,
        count: result.target_count
      })
    )
    emit('updated')
    handleClose()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.proxies.applyGroups.failed'))
  } finally {
    submitting.value = false
  }
}
</script>
