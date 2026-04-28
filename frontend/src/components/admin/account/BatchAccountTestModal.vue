<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.bulkTest.title')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="rounded-xl border border-primary-100 bg-primary-50/70 p-4 dark:border-primary-900/40 dark:bg-primary-900/10">
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="text-sm font-semibold text-primary-900 dark:text-primary-100">
              {{ t('admin.accounts.bulkTest.selectedCount', { count: results.length }) }}
            </div>
            <div class="mt-1 text-xs text-primary-700 dark:text-primary-300">
              {{ t('admin.accounts.bulkTest.modelHint') }}
            </div>
          </div>
          <div v-if="running" class="flex items-center gap-2 text-sm text-primary-800 dark:text-primary-200">
            <Icon name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <span>{{ t('admin.accounts.bulkTest.runningSummary', { current: activeRunOrdinal, total: results.length }) }}</span>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700">
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.bulkTest.summary.pending') }}</div>
          <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ pendingCount }}</div>
        </div>
        <div class="rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-900/40 dark:bg-blue-900/10">
          <div class="text-xs text-blue-700 dark:text-blue-300">{{ t('admin.accounts.bulkTest.summary.running') }}</div>
          <div class="mt-1 text-lg font-semibold text-blue-900 dark:text-blue-100">{{ runningCount }}</div>
        </div>
        <div class="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900/40 dark:bg-green-900/10">
          <div class="text-xs text-green-700 dark:text-green-300">{{ t('admin.accounts.bulkTest.summary.success') }}</div>
          <div class="mt-1 text-lg font-semibold text-green-900 dark:text-green-100">{{ successCount }}</div>
        </div>
        <div class="rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900/40 dark:bg-red-900/10">
          <div class="text-xs text-red-700 dark:text-red-300">{{ t('admin.accounts.bulkTest.summary.failed') }}</div>
          <div class="mt-1 text-lg font-semibold text-red-900 dark:text-red-100">{{ errorCount }}</div>
        </div>
      </div>

      <div class="max-h-[420px] overflow-y-auto rounded-xl border border-gray-200 bg-white dark:border-dark-500 dark:bg-dark-800">
        <div
          v-for="result in results"
          :key="result.id"
          class="border-b border-gray-100 px-4 py-3 last:border-b-0 dark:border-dark-600"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ result.name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">#{{ result.id }}</div>
            </div>
            <span :class="statusBadgeClass(result.status)" class="rounded-full px-2.5 py-1 text-xs font-semibold">
              {{ statusLabel(result.status) }}
            </span>
          </div>
          <div class="mt-3 grid gap-3 md:grid-cols-[minmax(0,1fr)_240px] md:items-start">
            <div class="text-sm text-gray-600 dark:text-gray-300">
              {{ result.message || t('admin.accounts.bulkTest.pendingMessage') }}
            </div>
            <div class="space-y-1">
              <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
                {{ t('admin.accounts.selectTestModel') }}
              </div>
              <Select
                v-model="result.selectedModelId"
                :options="result.availableModels"
                :disabled="running || result.loadingModels || result.availableModels.length === 0"
                value-key="id"
                label-key="display_name"
                :placeholder="result.loadingModels ? `${t('common.loading')}...` : t('admin.accounts.selectTestModel')"
              />
              <div v-if="result.loadingModels" class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('common.loading') }}...
              </div>
              <div v-else-if="result.modelError" class="text-xs text-red-500 dark:text-red-400">
                {{ result.modelError }}
              </div>
              <div v-else-if="result.availableModels.length === 0" class="text-xs text-amber-600 dark:text-amber-400">
                {{ t('admin.accounts.bulkTest.noModels') }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
        <button
          @click="startBatchTest"
          :disabled="!canStartBatchTest"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            !canStartBatchTest
              ? 'cursor-not-allowed bg-primary-400 text-white'
              : 'bg-primary-500 text-white hover:bg-primary-600'
          ]"
        >
          <Icon v-if="running" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
          <Icon v-else name="play" size="sm" :stroke-width="2" />
          <span>{{ running ? t('admin.accounts.testing') : t('admin.accounts.bulkActions.test') }}</span>
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
import { Icon } from '@/components/icons'
import { adminAPI } from '@/api/admin'
import { streamAccountTest, type AccountTestStreamEvent } from '@/utils/accountTestStream'
import type { AccountPlatform, ClaudeModel } from '@/types'

type ResultStatus = 'pending' | 'running' | 'success' | 'error'
type SelectableClaudeModel = ClaudeModel & Record<string, unknown>

function toSelectableClaudeModel(model: ClaudeModel): SelectableClaudeModel {
  return { ...model }
}

interface BatchTestTarget {
  id: number
  name: string
  platform?: AccountPlatform
}

interface BatchTestResult extends BatchTestTarget {
  status: ResultStatus
  message: string
  availableModels: SelectableClaudeModel[]
  selectedModelId: string
  loadingModels: boolean
  modelError: string
}

const props = defineProps<{
  show: boolean
  accounts: BatchTestTarget[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'completed'): void
}>()

const { t } = useI18n()

const results = ref<BatchTestResult[]>([])
const running = ref(false)
const activeRunOrdinal = ref(0)
const modelLoadToken = ref(0)
let abortController: AbortController | null = null
const prioritizedGeminiModels = ['gemini-3.1-flash-image', 'gemini-2.5-flash-image', 'gemini-2.5-flash', 'gemini-2.5-pro', 'gemini-3-flash-preview', 'gemini-3-pro-preview', 'gemini-2.0-flash']

const pendingCount = computed(() => results.value.filter((item) => item.status === 'pending').length)
const runningCount = computed(() => results.value.filter((item) => item.status === 'running').length)
const successCount = computed(() => results.value.filter((item) => item.status === 'success').length)
const errorCount = computed(() => results.value.filter((item) => item.status === 'error').length)
const canStartBatchTest = computed(() =>
  !running.value &&
  results.value.length > 0 &&
  results.value.every((item) => !item.loadingModels && !!item.selectedModelId)
)

watch(
  () => props.show,
  async (show) => {
    if (show) {
      resetResults()
      await loadAvailableModels()
      return
    }
    modelLoadToken.value += 1
    abortRun()
  },
  { immediate: true }
)

function resetResults() {
  activeRunOrdinal.value = 0
  results.value = props.accounts.map((account) => ({
    id: account.id,
    name: account.name,
    status: 'pending',
    message: '',
    platform: account.platform,
    availableModels: [],
    selectedModelId: '',
    loadingModels: true,
    modelError: ''
  }))
}

function resetRunState() {
  activeRunOrdinal.value = 0
  results.value = results.value.map((item) => ({
    ...item,
    status: 'pending',
    message: ''
  }))
}

function abortRun() {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
}

function handleClose() {
  abortRun()
  emit('close')
}

function updateResult(id: number, patch: Partial<BatchTestResult>) {
  results.value = results.value.map((item) => (item.id === id ? { ...item, ...patch } : item))
}

function sortTestModels(models: ClaudeModel[], platform?: AccountPlatform): SelectableClaudeModel[] {
  if (platform !== 'gemini' && platform !== 'antigravity') {
    return models.map(toSelectableClaudeModel)
  }

  const priorityMap = new Map(prioritizedGeminiModels.map((id, index) => [id, index]))
  return [...models].sort((a, b) => {
    const aPriority = priorityMap.get(a.id) ?? Number.MAX_SAFE_INTEGER
    const bPriority = priorityMap.get(b.id) ?? Number.MAX_SAFE_INTEGER
    if (aPriority !== bPriority) return aPriority - bPriority
    return a.display_name.localeCompare(b.display_name)
  }).map(toSelectableClaudeModel)
}

function getDefaultModelId(models: SelectableClaudeModel[], platform?: AccountPlatform) {
  if (models.length === 0) return ''
  if (platform === 'gemini' || platform === 'antigravity') {
    return models[0]?.id || ''
  }
  const sonnetModel = models.find((model) => model.id.includes('sonnet'))
  return sonnetModel?.id || models[0]?.id || ''
}

async function loadAvailableModels() {
  const currentToken = modelLoadToken.value + 1
  modelLoadToken.value = currentToken

  await Promise.all(results.value.map(async (result) => {
    updateResult(result.id, {
      loadingModels: true,
      modelError: '',
      availableModels: [],
      selectedModelId: ''
    })

    try {
      const models = await adminAPI.accounts.getAvailableModels(result.id)
      if (modelLoadToken.value !== currentToken) return

      const sortedModels = sortTestModels(models, result.platform)
      updateResult(result.id, {
        availableModels: sortedModels,
        selectedModelId: getDefaultModelId(sortedModels, result.platform),
        loadingModels: false,
        modelError: ''
      })
    } catch (error) {
      if (modelLoadToken.value !== currentToken) return
      console.error('Failed to load available models for batch account test:', error)
      updateResult(result.id, {
        loadingModels: false,
        modelError: t('admin.accounts.bulkTest.loadModelsFailed')
      })
    }
  }))
}

function statusLabel(status: ResultStatus) {
  switch (status) {
    case 'running':
      return t('admin.accounts.bulkTest.summary.running')
    case 'success':
      return t('admin.accounts.bulkTest.summary.success')
    case 'error':
      return t('admin.accounts.bulkTest.summary.failed')
    default:
      return t('admin.accounts.bulkTest.summary.pending')
  }
}

function statusBadgeClass(status: ResultStatus) {
  switch (status) {
    case 'running':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300'
    case 'success':
      return 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-300'
    case 'error':
      return 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
  }
}

function buildSuccessMessage(model: string, content: string, imageCount: number) {
  const trimmed = content.trim()
  if (trimmed) {
    return trimmed.length > 120 ? `${trimmed.slice(0, 117)}...` : trimmed
  }
  if (imageCount > 0) {
    return t('admin.accounts.bulkTest.imageGenerated', { count: imageCount })
  }
  if (model) {
    return t('admin.accounts.bulkTest.successWithModel', { model })
  }
  return t('admin.accounts.testCompleted')
}

async function startBatchTest() {
  if (running.value || props.accounts.length === 0) return

  resetRunState()
  running.value = true
  abortRun()
  abortController = new AbortController()
  const token = localStorage.getItem('auth_token')
  let finished = false
  const targets = results.value.map(({ id, name, selectedModelId }) => ({ id, name, selectedModelId }))

  try {
    for (const [index, account] of targets.entries()) {
      if (abortController.signal.aborted) return
      activeRunOrdinal.value = index + 1

      let lastModel = ''
      let contentBuffer = ''
      let imageCount = 0

      updateResult(account.id, {
        status: 'running',
        message: t('admin.accounts.bulkTest.runningMessage')
      })

      try {
        const result = await streamAccountTest({
          accountId: account.id,
          authToken: token,
          modelId: account.selectedModelId,
          signal: abortController.signal,
          onEvent: (event: AccountTestStreamEvent) => {
            if (event.type === 'test_start') {
              lastModel = event.model || ''
              updateResult(account.id, {
                message: lastModel
                  ? t('admin.accounts.bulkTest.testingWithModel', { model: lastModel })
                  : t('admin.accounts.bulkTest.runningMessage')
              })
              return
            }

            if (event.type === 'content' && event.text) {
              contentBuffer += event.text
              updateResult(account.id, {
                message: buildSuccessMessage(lastModel, contentBuffer, imageCount)
              })
              return
            }

            if (event.type === 'image') {
              imageCount += 1
              updateResult(account.id, {
                message: t('admin.accounts.bulkTest.imageReceived', { count: imageCount })
              })
            }
          }
        })

        updateResult(account.id, result.success
          ? {
              status: 'success',
              message: buildSuccessMessage(lastModel, contentBuffer, imageCount)
            }
          : {
              status: 'error',
              message: result.error || t('admin.accounts.testFailed')
            })
      } catch (error: unknown) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return
        }
        updateResult(account.id, {
          status: 'error',
          message: error instanceof Error ? error.message : t('admin.accounts.testFailed')
        })
      }
    }

    finished = true
  } finally {
    running.value = false
    activeRunOrdinal.value = 0
    abortController = null
    if (finished) {
      emit('completed')
    }
  }
}
</script>
