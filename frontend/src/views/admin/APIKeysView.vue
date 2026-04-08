<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-72">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="params.search"
              type="text"
              :placeholder="t('admin.apiKeys.searchPlaceholder')"
              class="input pl-10"
              @input="handleSearch"
            />
          </div>
          <div class="w-full sm:w-40">
            <Select
              v-model="params.status"
              :options="statusOptions"
              @change="handleFilterChange"
            />
          </div>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="load"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="apiKeys" :loading="loading">
          <template #cell-name="{ row }">
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
              <div class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">
                {{ maskKey(row.key) }}
              </div>
            </div>
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                {{ row.user?.email || `#${row.user_id}` }}
              </div>
              <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                {{ row.user?.username || '-' }}
              </div>
            </div>
          </template>

          <template #cell-group="{ row }">
            <GroupBadge
              v-if="row.group"
              :name="row.group.name"
              :platform="row.group.platform"
              :subscription-type="row.group.subscription_type"
              :rate-multiplier="row.group.rate_multiplier"
            />
            <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
          </template>

          <template #cell-status="{ value }">
            <span
              :class="[
                'badge',
                value === 'active'
                  ? 'badge-success'
                  : value === 'inactive'
                    ? 'badge-gray'
                    : 'badge-danger'
              ]"
            >
              {{ getStatusLabel(value) }}
            </span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-last_used_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ value ? formatRelativeTime(value) : t('common.time.never') }}
            </span>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { adminAPI } from '@/api/admin'
import { useTableLoader } from '@/composables/useTableLoader'
import type { Column } from '@/components/common/types'
import type { ApiKey } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime, formatRelativeTime } from '@/utils/format'

const { t } = useI18n()
const route = useRoute()

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.apiKeys.columns.name'), sortable: false },
  { key: 'user', label: t('admin.apiKeys.columns.user'), sortable: false },
  { key: 'group', label: t('admin.apiKeys.columns.group'), sortable: false },
  { key: 'status', label: t('admin.apiKeys.columns.status'), sortable: false },
  { key: 'created_at', label: t('admin.apiKeys.columns.createdAt'), sortable: true },
  { key: 'last_used_at', label: t('admin.apiKeys.columns.lastUsedAt'), sortable: true }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.apiKeys.allStatus') },
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') },
  { value: 'quota_exhausted', label: t('admin.apiKeys.status.quota_exhausted') },
  { value: 'expired', label: t('admin.apiKeys.status.expired') }
])

const {
  items: apiKeys,
  loading,
  params,
  pagination,
  load,
  reload,
  handlePageChange,
  handlePageSizeChange
} = useTableLoader<ApiKey, { search: string; status: string; group_id?: number }>({
  fetchFn: adminAPI.apiKeys.list,
  initialParams: {
    search: '',
    status: '',
    group_id: undefined
  }
})

const getSingleQueryValue = (value: unknown): string | undefined => {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string' && item.length > 0)
  }
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

const getNumericQueryValue = (value: unknown): number | undefined => {
  const raw = getSingleQueryValue(value)
  if (!raw) return undefined
  const parsed = Number(raw)
  return Number.isFinite(parsed) ? parsed : undefined
}

const applyRouteQueryFilters = () => {
  const search = getSingleQueryValue(route.query.search)
  const status = getSingleQueryValue(route.query.status)
  const groupID = getNumericQueryValue(route.query.group_id)

  if (search) {
    params.search = search
  }
  if (status) {
    params.status = status
  }
  params.group_id = groupID
}

const maskKey = (value: string) => {
  if (!value) return '-'
  if (value.length <= 16) return value
  return `${value.slice(0, 8)}...${value.slice(-6)}`
}

const getStatusLabel = (status: string) => {
  if (status === 'active') return t('common.active')
  if (status === 'inactive') return t('common.inactive')
  return t(`admin.apiKeys.status.${status}`)
}

let searchTimeout: number | undefined

const handleSearch = () => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = window.setTimeout(() => {
    void reload()
  }, 300)
}

const handleFilterChange = () => {
  void reload()
}

onMounted(() => {
  applyRouteQueryFilters()
  void load()
})

onUnmounted(() => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
})
</script>
