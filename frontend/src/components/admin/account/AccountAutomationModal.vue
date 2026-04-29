<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.automation.title')"
    width="extra-wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800/40 dark:bg-blue-900/20 dark:text-blue-200">
        {{ t('admin.accounts.automation.description') }}
      </div>

      <div class="flex items-center justify-between gap-3">
        <div class="text-sm text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.automation.selected', { count: selectedIds.length }) }}
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="loading || running" @click="loadCandidates">
          {{ t('admin.accounts.automation.reload') }}
        </button>
      </div>

      <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
        <div class="grid gap-3 md:grid-cols-4">
          <input
            v-model="searchQuery"
            type="text"
            class="input md:col-span-2"
            :placeholder="t('admin.accounts.searchAccounts')"
            :disabled="running"
          />
          <Select
            v-model="selectedGroup"
            class="w-full"
            :options="groupOptions"
            :disabled="running"
          />
          <Select
            v-model="selectedStatus"
            class="w-full"
            :options="statusOptions"
            :disabled="running"
          />
        </div>
        <div class="mt-3 flex justify-end">
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="loading || running"
            @click="loadCandidates"
          >
            {{ t('common.search') }}
          </button>
        </div>
      </div>

      <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
        <label class="flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-gray-200">
          <input
            v-model="proxyEnabled"
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span>{{ t('admin.accounts.automation.proxyEnabled') }}</span>
        </label>
        <div class="mt-3">
          <label class="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.automation.proxyEndpoint') }}
          </label>
          <input
            v-model="proxyEndpoint"
            type="text"
            class="input"
            :disabled="running || !proxyEnabled"
            :placeholder="t('admin.accounts.automation.proxyPlaceholder')"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.automation.proxyHint') }}
          </p>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700">
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.automation.summary.total') }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ jobStatus?.total ?? candidates.length }}</div>
        </div>
        <div class="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900/40 dark:bg-green-900/10">
          <div class="text-xs text-green-700 dark:text-green-300">{{ t('admin.accounts.automation.summary.success') }}</div>
          <div class="mt-1 text-lg font-semibold text-green-900 dark:text-green-100">{{ jobStatus?.success ?? 0 }}</div>
        </div>
        <div class="rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900/40 dark:bg-red-900/10">
          <div class="text-xs text-red-700 dark:text-red-300">{{ t('admin.accounts.automation.summary.failed') }}</div>
          <div class="mt-1 text-lg font-semibold text-red-900 dark:text-red-100">{{ jobStatus?.failed ?? 0 }}</div>
        </div>
        <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/40 dark:bg-amber-900/10">
          <div class="text-xs text-amber-700 dark:text-amber-300">{{ t('admin.accounts.automation.summary.skipped') }}</div>
          <div class="mt-1 text-lg font-semibold text-amber-900 dark:text-amber-100">{{ jobStatus?.skipped ?? 0 }}</div>
        </div>
      </div>

      <div v-if="resultSummary" class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-200">
        {{ resultSummary }}
      </div>

      <div v-if="loading" class="flex items-center justify-center py-12 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.refreshing') }}
      </div>

      <div v-else-if="candidates.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
        {{ t('admin.accounts.automation.empty') }}
      </div>

      <div v-else class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
        <div class="max-h-[320px] overflow-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left">
                  <input
                    type="checkbox"
                    class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :checked="allSelected"
                    @change="toggleAll($event)"
                  />
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-300">
                  {{ t('admin.accounts.automation.table.name') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-300">
                  {{ t('admin.accounts.automation.table.email') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-300">
                  {{ t('admin.accounts.automation.table.error') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-300">
                  {{ t('admin.accounts.automation.table.updated') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="account in candidates" :key="account.id">
                <td class="px-4 py-3 align-top">
                  <input
                    type="checkbox"
                    class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :checked="selectedIds.includes(account.id)"
                    @change="toggleOne(account.id)"
                  />
                </td>
                <td class="px-4 py-3 align-top">
                  <div class="font-medium text-gray-900 dark:text-white">{{ account.name }}</div>
                </td>
                <td class="px-4 py-3 align-top text-sm text-gray-600 dark:text-gray-300">
                  {{ extractEmail(account) || '-' }}
                </td>
                <td class="px-4 py-3 align-top text-sm text-gray-600 dark:text-gray-300">
                  <div class="max-w-xl whitespace-pre-wrap break-words">{{ account.error_message || '-' }}</div>
                </td>
                <td class="px-4 py-3 align-top text-sm text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(account.updated_at) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
        <div class="border-b border-gray-200 bg-gray-50 px-4 py-3 text-sm font-medium text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200">
          {{ t('admin.accounts.automation.logs') }}
        </div>
        <div ref="logContainerRef" class="max-h-[260px] overflow-auto bg-gray-950 px-4 py-3 font-mono text-xs leading-6 text-gray-100">
          <div v-if="logEntries.length === 0" class="text-gray-400">
            {{ t('admin.accounts.automation.logsEmpty') }}
          </div>
          <div v-for="entry in logEntries" :key="entry.seq" class="break-words">
            <span class="text-gray-500">[{{ formatLogTime(entry.at) }}]</span>
            <span :class="logLevelClass(entry.level)">[{{ entry.level.toUpperCase() }}]</span>
            <span v-if="entry.account_id" class="text-sky-300">[#{{ entry.account_id }}]</span>
            <span>{{ entry.message }}</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-between gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="running || loading || selectedIds.length === 0"
          @click="handleRun"
        >
          {{ running ? t('admin.accounts.automation.refreshing') : t('admin.accounts.automation.refresh') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import { useAppStore } from '@/stores/app'
import type { OpenAIAutoReauthJobLogEntry, OpenAIAutoReauthJobStatus } from '@/api/admin/accounts'
import type { Account, AdminGroup } from '@/types'
import { formatRelativeTime } from '@/utils/format'

interface Props {
  show: boolean
  groups?: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  completed: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const running = ref(false)
const candidates = ref<Account[]>([])
const selectedIds = ref<number[]>([])
const resultSummary = ref('')
const logEntries = ref<OpenAIAutoReauthJobLogEntry[]>([])
const jobStatus = ref<OpenAIAutoReauthJobStatus | null>(null)
const currentJobId = ref('')
const logContainerRef = ref<HTMLElement | null>(null)
const proxyEnabled = ref(false)
const proxyEndpoint = ref('127.0.0.1:7890')
const searchQuery = ref('')
const selectedGroup = ref('')
const selectedStatus = ref('')

let pollTimer: ReturnType<typeof setTimeout> | null = null
let lastLogSeq = 0

const allSelected = computed(() => candidates.value.length > 0 && selectedIds.value.length === candidates.value.length)
const statusOptions = computed(() => [
  { value: '', label: t('admin.accounts.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') },
  { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') },
  { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') },
  { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') }
])
const groupOptions = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...((props.groups || []).map(group => ({
    value: String(group.id),
    label: group.name
  })))
])

const extractEmail = (account: Account) => {
  const candidates = [
    String(account.credentials?.email || ''),
    String(account.extra?.email || ''),
    String(account.extra?.email_address || ''),
    account.name
  ]
  return candidates.find(value => value.includes('@')) || ''
}

const stopPolling = () => {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

const formatLogTime = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString()
}

const logLevelClass = (level: string) => {
  switch (level) {
    case 'error':
      return 'text-red-300'
    case 'warn':
      return 'text-amber-300'
    default:
      return 'text-emerald-300'
  }
}

const scrollLogsToBottom = async () => {
  await nextTick()
  if (logContainerRef.value) {
    logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
  }
}

const loadCandidates = async () => {
  if (!props.show) return
  loading.value = true
  resultSummary.value = ''
  try {
    candidates.value = await adminAPI.accounts.getOpenAIAutoReauthCandidates({
      status: selectedStatus.value || '',
      group: selectedGroup.value || '',
      search: searchQuery.value.trim()
    })
    selectedIds.value = candidates.value.map(account => account.id)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.failedToLoad'))
  } finally {
    loading.value = false
  }
}

const toggleAll = (event: Event) => {
  const target = event.target as HTMLInputElement
  selectedIds.value = target.checked ? candidates.value.map(account => account.id) : []
}

const toggleOne = (id: number) => {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter(item => item !== id)
    : [...selectedIds.value, id]
}

const finalizeJob = async (status: OpenAIAutoReauthJobStatus) => {
  const skipped = status.skipped || 0
  resultSummary.value = t(
    status.failed > 0 || skipped > 0
      ? 'admin.accounts.automation.partial'
      : 'admin.accounts.automation.completed',
    {
      success: status.success,
      failed: status.failed,
      skipped
    }
  )
  if (status.failed > 0 || skipped > 0) {
    appStore.showError(resultSummary.value)
  } else {
    appStore.showSuccess(resultSummary.value)
  }
  emit('completed')
  await loadCandidates()
}

const pollJob = async () => {
  if (!currentJobId.value) return
  try {
    const status = await adminAPI.accounts.getOpenAIAutoReauthJob(currentJobId.value, lastLogSeq > 0 ? lastLogSeq : undefined)
    jobStatus.value = status
    if (status.logs.length > 0) {
      logEntries.value = [...logEntries.value, ...status.logs]
      lastLogSeq = status.logs[status.logs.length - 1]?.seq || lastLogSeq
      await scrollLogsToBottom()
    }
    if (status.status === 'completed') {
      running.value = false
      stopPolling()
      await finalizeJob(status)
      return
    }
    pollTimer = setTimeout(() => {
      void pollJob()
    }, 1000)
  } catch (error: any) {
    running.value = false
    stopPolling()
    appStore.showError(error?.message || t('admin.accounts.automation.failed'))
  }
}

const handleRun = async () => {
  if (selectedIds.value.length === 0) {
    appStore.showError(t('admin.accounts.automation.missingSelection'))
    return
  }
  running.value = true
  resultSummary.value = ''
  logEntries.value = []
  jobStatus.value = null
  currentJobId.value = ''
  lastLogSeq = 0
  stopPolling()
  try {
    const result = await adminAPI.accounts.batchOpenAIAutoReauth(selectedIds.value, {
      proxy_enabled: proxyEnabled.value,
      proxy_endpoint: proxyEndpoint.value.trim()
    })
    currentJobId.value = result.job_id
    await pollJob()
  } catch (error: any) {
    running.value = false
    appStore.showError(error?.message || t('admin.accounts.automation.failed'))
  }
}

watch(
  () => props.show,
  (show) => {
    if (!show) {
      stopPolling()
      candidates.value = []
      selectedIds.value = []
      resultSummary.value = ''
      logEntries.value = []
      jobStatus.value = null
      currentJobId.value = ''
      lastLogSeq = 0
      proxyEnabled.value = false
      proxyEndpoint.value = '127.0.0.1:7890'
      searchQuery.value = ''
      selectedGroup.value = ''
      selectedStatus.value = ''
    }
  }
)

onBeforeUnmount(() => {
  stopPolling()
})
</script>
