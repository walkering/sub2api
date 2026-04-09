<template>
  <BaseDialog
    :show="show"
    :title="t('admin.scheduledTests.realtimeLogsTitle', { name: groupName || `#${groupId ?? ''}` })"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduledTests.realtimeLogsHint') }}
        </div>
        <button class="btn btn-secondary" :disabled="loadingJobs || !groupId" @click="loadJobs">
          {{ t('common.refresh') }}
        </button>
      </div>

      <div class="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
          <div class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.scheduledTests.recentJobs') }}
          </div>
          <div v-if="loadingJobs" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('common.loading') }}...
          </div>
          <div v-else-if="jobs.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.scheduledTests.noJobs') }}
          </div>
          <div v-else class="space-y-2">
            <button
              v-for="job in jobs"
              :key="job.id"
              class="w-full rounded-lg border px-3 py-2 text-left transition"
              :class="selectedJobId === job.id
                ? 'border-primary-500 bg-primary-50 dark:border-primary-500 dark:bg-primary-500/10'
                : 'border-gray-200 hover:border-primary-300 dark:border-dark-600 dark:hover:border-primary-500/50'"
              @click="selectJob(job.id)"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">
                  #{{ job.id }}
                </div>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ job.status }}</span>
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ formatDateTime(job.created_at) }}
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.scheduledTests.progressSummary', {
                  done: job.succeeded_accounts + job.failed_accounts,
                  total: job.total_accounts
                }) }}
              </div>
            </button>
          </div>
        </div>

        <div class="min-w-0 space-y-4">
          <div v-if="!snapshot" class="rounded-xl border border-dashed border-gray-300 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
            {{ t('admin.scheduledTests.selectJobFirst') }}
          </div>

          <template v-else>
            <div class="grid gap-3 md:grid-cols-5">
              <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduledTests.jobStatus') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ snapshot.job.status }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduledTests.pending') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ snapshot.job.pending_accounts }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduledTests.running') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ snapshot.job.running_accounts }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduledTests.succeeded') }}</div>
                <div class="mt-1 text-sm font-medium text-emerald-600 dark:text-emerald-400">{{ snapshot.job.succeeded_accounts }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.scheduledTests.failed') }}</div>
                <div class="mt-1 text-sm font-medium text-red-600 dark:text-red-400">{{ snapshot.job.failed_accounts }}</div>
              </div>
            </div>

            <div class="rounded-xl border border-gray-200 dark:border-dark-600">
              <div class="border-b border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 dark:border-dark-600 dark:text-gray-300">
                {{ t('admin.scheduledTests.jobLogs') }}
              </div>
              <div class="max-h-72 overflow-y-auto px-4 py-3">
                <div v-if="snapshot.logs.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.scheduledTests.noLogs') }}
                </div>
                <div v-else class="space-y-3">
                  <div v-for="log in snapshot.logs" :key="log.id" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50">
                    <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
                      <span>{{ log.event_type }}</span>
                      <span>{{ formatDateTime(log.created_at) }}</span>
                    </div>
                    <div class="mt-1 text-sm text-gray-800 dark:text-gray-200">{{ log.message }}</div>
                    <div v-if="log.error_message" class="mt-1 text-xs text-red-500 dark:text-red-400">
                      {{ log.error_message }}
                    </div>
                    <div v-else-if="log.response_text" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                      {{ log.response_text }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-xl border border-gray-200 dark:border-dark-600">
              <div class="border-b border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 dark:border-dark-600 dark:text-gray-300">
                {{ t('admin.scheduledTests.accountResults') }}
              </div>
              <div class="overflow-x-auto">
                <table class="min-w-full text-sm">
                  <thead class="bg-gray-50 dark:bg-dark-700/50">
                    <tr>
                      <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.scheduledTests.account') }}</th>
                      <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.scheduledTests.status') }}</th>
                      <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.scheduledTests.scheduledFor') }}</th>
                      <th class="px-4 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.scheduledTests.latencyMs') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="item in snapshot.items" :key="item.id" class="border-t border-gray-100 dark:border-dark-700">
                      <td class="px-4 py-2 text-gray-900 dark:text-gray-100">{{ item.account_name }}</td>
                      <td class="px-4 py-2 text-gray-600 dark:text-gray-300">{{ item.status }}</td>
                      <td class="px-4 py-2 text-gray-600 dark:text-gray-300">{{ formatDateTime(item.scheduled_for) }}</td>
                      <td class="px-4 py-2 text-gray-600 dark:text-gray-300">{{ item.latency_ms || '—' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { AccountTestJob, AccountTestJobSnapshot } from '@/types'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  groupId: number | null
  groupName?: string | null
  initialJobId?: number | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const jobs = ref<AccountTestJob[]>([])
const snapshot = ref<AccountTestJobSnapshot | null>(null)
const selectedJobId = ref<number | null>(null)
const loadingJobs = ref(false)
let controller: AbortController | null = null

watch(
  () => props.show,
  async (visible) => {
    if (!visible) {
      stopStream()
      snapshot.value = null
      selectedJobId.value = null
      return
    }
    await loadJobs()
  }
)

watch(
  () => props.initialJobId,
  async (jobId) => {
    if (props.show && jobId) {
      await selectJob(jobId)
    }
  }
)

const loadJobs = async () => {
  if (!props.groupId) {
    jobs.value = []
    return
  }
  loadingJobs.value = true
  try {
    jobs.value = await adminAPI.scheduledTests.listGroupJobs(props.groupId, 20)
    const nextJobId = props.initialJobId ?? jobs.value[0]?.id ?? null
    if (nextJobId) {
      await selectJob(nextJobId)
    } else {
      snapshot.value = null
      selectedJobId.value = null
    }
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.loadFailed'))
  } finally {
    loadingJobs.value = false
  }
}

const selectJob = async (jobId: number) => {
  selectedJobId.value = jobId
  stopStream()
  try {
    snapshot.value = await adminAPI.scheduledTests.getJobSnapshot(jobId)
    startStream(jobId)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.scheduledTests.loadFailed'))
  }
}

const startStream = async (jobId: number) => {
  controller = new AbortController()
  try {
    const response = await fetch(`/api/v1/admin/test-jobs/${jobId}/logs/stream`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`
      },
      signal: controller.signal
    })

    if (!response.ok || !response.body) {
      return
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const payload = line.slice(6).trim()
        if (!payload) continue
        try {
          const parsed = JSON.parse(payload)
          if (parsed.type === 'snapshot' && parsed.snapshot) {
            snapshot.value = parsed.snapshot as AccountTestJobSnapshot
          }
        } catch {
          // ignore malformed line
        }
      }
    }
  } catch {
    // ignore stream cancellation/network close
  }
}

const stopStream = () => {
  if (controller) {
    controller.abort()
    controller = null
  }
}

const handleClose = () => {
  stopStream()
  emit('close')
}

const formatDateTime = (value: string | null) => {
  if (!value) return '—'
  return new Date(value).toLocaleString()
}
</script>
