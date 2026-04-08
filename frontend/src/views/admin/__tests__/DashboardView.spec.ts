import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { DashboardStats } from '@/types'
import DashboardView from '../DashboardView.vue'

const DASHBOARD_AUTO_REFRESH_STORAGE_KEY = 'admin.dashboard.auto_refresh'

const {
  getSnapshotV2,
  getUserUsageTrend,
  getUserSpendingRanking,
  localStorageMock,
  localStorageState,
  routerPush
} = vi.hoisted(() => {
  const storage = new Map<string, string>()
  const localStorageMock = {
    getItem: vi.fn((key: string) => storage.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      storage.set(key, value)
    }),
    removeItem: vi.fn((key: string) => {
      storage.delete(key)
    }),
    clear: vi.fn(() => {
      storage.clear()
    })
  }

  return {
    getSnapshotV2: vi.fn(),
    getUserUsageTrend: vi.fn(),
    getUserSpendingRanking: vi.fn(),
    localStorageMock,
    localStorageState: storage,
    routerPush: vi.fn()
  }
})

const messages: Record<string, string> = {
  'common.refresh': 'Refresh',
  'admin.dashboard.enableAutoRefresh': 'Enable auto refresh',
  'admin.dashboard.disableAutoRefresh': 'Disable auto refresh',
  'admin.dashboard.refreshInterval15s': '15 seconds',
  'admin.dashboard.refreshInterval30s': '30 seconds',
  'admin.dashboard.refreshInterval60s': '60 seconds',
  'admin.dashboard.timeRange': 'Time Range',
  'admin.dashboard.day': 'Day',
  'admin.dashboard.hour': 'Hour'
}

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getSnapshotV2,
      getUserUsageTrend,
      getUserSpendingRanking
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.dashboard.autoRefreshCountdown' && typeof params?.seconds === 'number') {
          return `Auto refresh: ${params.seconds}s`
        }
        return messages[key] ?? key
      }
    })
  }
})

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const createDashboardStats = (): DashboardStats => ({
  total_users: 0,
  today_new_users: 0,
  active_users: 0,
  hourly_active_users: 0,
  stats_updated_at: '',
  stats_stale: false,
  total_api_keys: 0,
  active_api_keys: 0,
  total_accounts: 0,
  normal_accounts: 0,
  error_accounts: 0,
  ratelimit_accounts: 0,
  overload_accounts: 0,
  total_requests: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_creation_tokens: 0,
  total_cache_read_tokens: 0,
  total_tokens: 0,
  total_cost: 0,
  total_actual_cost: 0,
  today_requests: 0,
  today_input_tokens: 0,
  today_output_tokens: 0,
  today_cache_creation_tokens: 0,
  today_cache_read_tokens: 0,
  today_tokens: 0,
  today_cost: 0,
  today_actual_cost: 0,
  average_duration_ms: 0,
  uptime: 0,
  rpm: 0,
  tpm: 0
})

const createSnapshotResponse = () => ({
  stats: createDashboardStats(),
  trend: [],
  models: []
})

const DateRangePickerStub = {
  name: 'DateRangePicker',
  emits: ['update:startDate', 'update:endDate', 'change'],
  template: '<div data-test="date-range-picker"></div>'
}

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: '<div data-test="select-stub">{{ modelValue }}</div>'
}

const mountDashboardView = () =>
  mount(DashboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        LoadingSpinner: true,
        Icon: true,
        DateRangePicker: DateRangePickerStub,
        Select: SelectStub,
        ModelDistributionChart: true,
        TokenUsageTrend: true,
        Line: true
      }
    }
  })

describe('admin DashboardView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      configurable: true
    })
    localStorageState.clear()
    localStorageMock.getItem.mockClear()
    localStorageMock.setItem.mockClear()
    localStorageMock.removeItem.mockClear()
    localStorageMock.clear.mockClear()

    getSnapshotV2.mockReset()
    getUserUsageTrend.mockReset()
    getUserSpendingRanking.mockReset()
    routerPush.mockReset()

    getSnapshotV2.mockResolvedValue(createSnapshotResponse())
    getUserUsageTrend.mockResolvedValue({
      trend: [],
      start_date: '',
      end_date: '',
      granularity: 'hour'
    })
    getUserSpendingRanking.mockResolvedValue({
      ranking: [],
      total_actual_cost: 0,
      total_requests: 0,
      total_tokens: 0,
      start_date: '',
      end_date: ''
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('uses last 24 hours as default dashboard range and keeps auto refresh disabled by default', async () => {
    const wrapper = mountDashboardView()

    await flushPromises()

    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)

    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour'
    }))
    expect(localStorageMock.getItem).toHaveBeenCalledWith(DASHBOARD_AUTO_REFRESH_STORAGE_KEY)
    expect(wrapper.text()).toContain('Enable auto refresh')

    vi.advanceTimersByTime(30000)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
  })

  it('restores saved auto refresh settings from localStorage', async () => {
    localStorageState.set(
      DASHBOARD_AUTO_REFRESH_STORAGE_KEY,
      JSON.stringify({ enabled: true, interval_seconds: 30 })
    )

    const wrapper = mountDashboardView()
    await flushPromises()

    expect(wrapper.text()).toContain('Disable auto refresh')
    expect(wrapper.text()).toContain('Auto refresh: 30s')
  })

  it('reuses the latest filters when auto refresh triggers', async () => {
    localStorageState.set(
      DASHBOARD_AUTO_REFRESH_STORAGE_KEY,
      JSON.stringify({ enabled: true, interval_seconds: 30 })
    )

    const wrapper = mountDashboardView()
    await flushPromises()

    const datePicker = wrapper.findComponent({ name: 'DateRangePicker' })
    datePicker.vm.$emit('update:startDate', '2026-04-01')
    datePicker.vm.$emit('update:endDate', '2026-04-03')
    datePicker.vm.$emit('change', {
      startDate: '2026-04-01',
      endDate: '2026-04-03',
      preset: null
    })

    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(2)
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: '2026-04-01',
      end_date: '2026-04-03',
      granularity: 'day'
    }))

    vi.advanceTimersByTime(30000)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(3)
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: '2026-04-01',
      end_date: '2026-04-03',
      granularity: 'day'
    }))
  })

  it('skips overlapping auto refresh requests while a page refresh is still running', async () => {
    localStorageState.set(
      DASHBOARD_AUTO_REFRESH_STORAGE_KEY,
      JSON.stringify({ enabled: true, interval_seconds: 15 })
    )

    let resolveSnapshot: ((value: ReturnType<typeof createSnapshotResponse>) => void) | null = null
    const pendingSnapshot = new Promise<ReturnType<typeof createSnapshotResponse>>((resolve) => {
      resolveSnapshot = resolve
    })

    getSnapshotV2
      .mockResolvedValueOnce(createSnapshotResponse())
      .mockImplementationOnce(() => pendingSnapshot)
      .mockResolvedValue(createSnapshotResponse())

    const wrapper = mountDashboardView()
    await flushPromises()

    expect(wrapper.text()).toContain('Auto refresh: 15s')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(15000)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(15000)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(2)

    resolveSnapshot?.(createSnapshotResponse())
    await flushPromises()

    vi.advanceTimersByTime(15000)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(3)
  })

  it('navigates metric drilldowns to the expected filtered admin pages', async () => {
    const wrapper = mountDashboardView()
    await flushPromises()

    const today = formatLocalDate(new Date())

    await wrapper.get('[data-test="dashboard-total-api-keys"]').trigger('click')
    await wrapper.get('[data-test="dashboard-active-api-keys"]').trigger('click')
    await wrapper.get('[data-test="dashboard-total-accounts"]').trigger('click')
    await wrapper.get('[data-test="dashboard-normal-accounts"]').trigger('click')
    await wrapper.get('[data-test="dashboard-error-accounts"]').trigger('click')
    await wrapper.get('[data-test="dashboard-today-requests"]').trigger('click')
    await wrapper.get('[data-test="dashboard-today-new-users"]').trigger('click')
    await wrapper.get('[data-test="dashboard-total-users"]').trigger('click')
    await wrapper.get('[data-test="dashboard-active-users"]').trigger('click')

    expect(routerPush).toHaveBeenNthCalledWith(1, {
      path: '/admin/api-keys',
      query: {}
    })
    expect(routerPush).toHaveBeenNthCalledWith(2, {
      path: '/admin/api-keys',
      query: { status: 'active' }
    })
    expect(routerPush).toHaveBeenNthCalledWith(3, {
      path: '/admin/accounts',
      query: {}
    })
    expect(routerPush).toHaveBeenNthCalledWith(4, {
      path: '/admin/accounts',
      query: { status: 'active' }
    })
    expect(routerPush).toHaveBeenNthCalledWith(5, {
      path: '/admin/accounts',
      query: { status: 'error' }
    })
    expect(routerPush).toHaveBeenNthCalledWith(6, {
      path: '/admin/usage',
      query: {
        start_date: today,
        end_date: today
      }
    })
    expect(routerPush).toHaveBeenNthCalledWith(7, {
      path: '/admin/users',
      query: { created_scope: 'today' }
    })
    expect(routerPush).toHaveBeenNthCalledWith(8, {
      path: '/admin/users',
      query: {}
    })
    expect(routerPush).toHaveBeenNthCalledWith(9, {
      path: '/admin/users',
      query: { activity_scope: 'today_active' }
    })
  })
})
