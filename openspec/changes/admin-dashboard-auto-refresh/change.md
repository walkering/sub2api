## Summary

已完成 `/admin/dashboard` 自动刷新能力的前端实现，并同步更新 OpenSpec 任务状态。

- 在总览页现有刷新控制区增加自动刷新开关、15/30/60 秒预设间隔和倒计时展示。
- 自动刷新配置使用浏览器本地持久化，页面重新进入后恢复启用状态与间隔，并在组件卸载时停止定时器。
- 整页刷新统一收口到 `runDashboardRefresh()`，使用 in-flight guard 避免自动刷新、手动刷新和筛选刷新重叠请求；非自动触发在冲突时排队补刷一次。
- 成功刷新后统一重置倒计时，自动刷新始终沿用当前 `startDate`、`endDate` 和 `granularity`。
- 已补充中英文文案与单测，并在 `tasks.md` 中勾选 `1.1`-`3.1`；`3.2` 因本机 OOM 无法完成手动验证，暂未勾选。

## Modified Files

- `frontend/src/views/admin/DashboardView.vue:223`：刷新按钮改为走统一整页刷新入口，并在同一区域增加自动刷新开关、间隔选择和倒计时展示。
- `frontend/src/views/admin/DashboardView.vue:392`：新增自动刷新存储 key、预设间隔、启用状态与倒计时状态。
- `frontend/src/views/admin/DashboardView.vue:667`：新增本地配置恢复、配置持久化、启停控制与倒计时重置逻辑。
- `frontend/src/views/admin/DashboardView.vue:832`：新增 `runDashboardRefresh()`、重叠请求保护、手动/筛选刷新入口与基于 `useIntervalFn` 的自动刷新调度。
- `frontend/src/views/admin/DashboardView.vue:898`：组件卸载时停止自动刷新定时器。
- `frontend/src/i18n/locales/en.ts:970`：补充自动刷新英文文案。
- `frontend/src/i18n/locales/zh.ts:983`：补充自动刷新中文文案。
- `frontend/src/views/admin/__tests__/DashboardView.spec.ts:198`：覆盖默认关闭且不自动触发刷新。
- `frontend/src/views/admin/__tests__/DashboardView.spec.ts:221`：覆盖从 `localStorage` 恢复自动刷新配置。
- `frontend/src/views/admin/__tests__/DashboardView.spec.ts:234`：覆盖自动刷新沿用最新日期范围与粒度。
- `frontend/src/views/admin/__tests__/DashboardView.spec.ts:272`：覆盖整页刷新 in-flight guard，确保自动刷新不会并发重入。
- `openspec/changes/admin-dashboard-auto-refresh/tasks.md:3`：勾选 `1.1`-`3.1`，保留 `3.2` 未完成。

## Verification

- 通过：`npm run lint:check -- src/views/admin/DashboardView.vue src/views/admin/__tests__/DashboardView.spec.ts`
- 通过：`git diff --check -- frontend/src/views/admin/DashboardView.vue frontend/src/views/admin/__tests__/DashboardView.spec.ts frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts`
- 受本机 OOM 影响未完成：`npm run test:run -- src/views/admin/__tests__/DashboardView.spec.ts`
- 受本机 OOM 影响未完成：`npm run typecheck`
- 受本机链接阶段 OOM 影响未完成：本地启动后端手工验证 `/admin/dashboard`，报错 `runtime: VirtualAlloc ... fatal error: out of memory`

## Unified Diff

### `frontend/src/views/admin/DashboardView.vue`

```diff
diff --git a/frontend/src/views/admin/DashboardView.vue b/frontend/src/views/admin/DashboardView.vue
@@
-              <button @click="loadDashboardStats" :disabled="chartsLoading" class="btn btn-secondary">
+              <button @click="handleManualRefresh" :disabled="pageRefreshInFlight" class="btn btn-secondary">
                 {{ t('common.refresh') }}
               </button>
+              <div class="flex flex-wrap items-center gap-2">
+                <button
+                  type="button"
+                  class="btn"
+                  :class="autoRefreshEnabled ? 'btn-primary' : 'btn-secondary'"
+                  :aria-pressed="autoRefreshEnabled"
+                  @click="toggleAutoRefresh"
+                >
+                  <Icon name="refresh" size="sm" :class="autoRefreshEnabled ? 'animate-spin' : ''" />
+                  <span>
+                    {{
+                      autoRefreshEnabled
+                        ? t('admin.dashboard.disableAutoRefresh')
+                        : t('admin.dashboard.enableAutoRefresh')
+                    }}
+                  </span>
+                </button>
+                <Select
+                  v-model="autoRefreshIntervalSeconds"
+                  :options="autoRefreshIntervalOptions"
+                  @change="handleAutoRefreshIntervalChange"
+                />
+                <span v-if="autoRefreshEnabled">
+                  {{ t('admin.dashboard.autoRefreshCountdown', { seconds: autoRefreshCountdown }) }}
+                </span>
+              </div>
@@
-                    @change="loadChartData"
+                    @change="handleGranularityChange"
@@
+const DASHBOARD_AUTO_REFRESH_STORAGE_KEY = 'admin.dashboard.auto_refresh'
+const autoRefreshIntervals = [15, 30, 60] as const
+const autoRefreshEnabled = ref(false)
+const autoRefreshIntervalSeconds = ref<AutoRefreshIntervalSeconds>(30)
+const autoRefreshCountdown = ref(0)
@@
+const restoreAutoRefreshSettings = () => { ... }
+const setAutoRefreshEnabled = (enabled: boolean) => { ... }
+const setAutoRefreshInterval = (seconds: AutoRefreshIntervalSeconds) => { ... }
@@
-  loadChartData()
+  void runDashboardRefresh('filters')
@@
-const loadChartData = async () => {
-  await Promise.all([
-    loadDashboardSnapshot(false),
-    loadUsersTrend(),
-    loadUserSpendingRanking()
-  ])
-}
+const runDashboardRefresh = async (reason: DashboardRefreshReason): Promise<boolean> => {
+  if (pageRefreshInFlight.value) {
+    if (reason !== 'auto') {
+      queuedPageRefresh = true
+    }
+    return false
+  }
+  pageRefreshInFlight.value = true
+  try {
+    await loadDashboardStats()
+    resetAutoRefreshCountdown()
+    return true
+  } finally {
+    pageRefreshInFlight.value = false
+    if (queuedPageRefresh) {
+      queuedPageRefresh = false
+      void runDashboardRefresh('filters')
+    }
+  }
+}
+
+const { pause: pauseAutoRefresh, resume: resumeAutoRefresh } = useIntervalFn(() => {
+  if (!autoRefreshEnabled.value) return
+  if (autoRefreshCountdown.value > 0) {
+    autoRefreshCountdown.value -= 1
+  }
+  if (autoRefreshCountdown.value > 0 || pageRefreshInFlight.value) return
+  void runDashboardRefresh('auto')
+}, 1000, { immediate: false })
@@
-onMounted(() => {
-  loadDashboardStats()
-})
+onMounted(() => {
+  restoreAutoRefreshSettings()
+  if (autoRefreshEnabled.value) {
+    resumeAutoRefresh()
+  } else {
+    pauseAutoRefresh()
+  }
+  void runDashboardRefresh('initial')
+})
+
+onUnmounted(() => {
+  pauseAutoRefresh()
+})
```

### `frontend/src/i18n/locales/en.ts` / `frontend/src/i18n/locales/zh.ts`

```diff
diff --git a/frontend/src/i18n/locales/en.ts b/frontend/src/i18n/locales/en.ts
@@
+      autoRefresh: 'Auto Refresh',
+      enableAutoRefresh: 'Enable auto refresh',
+      disableAutoRefresh: 'Disable auto refresh',
+      refreshInterval: 'Refresh Interval',
+      refreshInterval15s: '15 seconds',
+      refreshInterval30s: '30 seconds',
+      refreshInterval60s: '60 seconds',
+      autoRefreshCountdown: 'Auto refresh: {seconds}s',

diff --git a/frontend/src/i18n/locales/zh.ts b/frontend/src/i18n/locales/zh.ts
@@
+      autoRefresh: '自动刷新',
+      enableAutoRefresh: '启用自动刷新',
+      disableAutoRefresh: '关闭自动刷新',
+      refreshInterval: '刷新间隔',
+      refreshInterval15s: '15 秒',
+      refreshInterval30s: '30 秒',
+      refreshInterval60s: '60 秒',
+      autoRefreshCountdown: '自动刷新：{seconds}s',
```

### `frontend/src/views/admin/__tests__/DashboardView.spec.ts`

```diff
diff --git a/frontend/src/views/admin/__tests__/DashboardView.spec.ts b/frontend/src/views/admin/__tests__/DashboardView.spec.ts
@@
+const DASHBOARD_AUTO_REFRESH_STORAGE_KEY = 'admin.dashboard.auto_refresh'
+const { getSnapshotV2, getUserUsageTrend, getUserSpendingRanking, localStorageMock, localStorageState } =
+  vi.hoisted(() => { ... })
@@
+it('uses last 24 hours as default dashboard range and keeps auto refresh disabled by default', async () => { ... })
+it('restores saved auto refresh settings from localStorage', async () => { ... })
+it('reuses the latest filters when auto refresh triggers', async () => { ... })
+it('skips overlapping auto refresh requests while a page refresh is still running', async () => { ... })
```

### `openspec/changes/admin-dashboard-auto-refresh/tasks.md`

```diff
diff --git a/openspec/changes/admin-dashboard-auto-refresh/tasks.md b/openspec/changes/admin-dashboard-auto-refresh/tasks.md
@@
- [ ] 1.1 在 `frontend/src/views/admin/DashboardView.vue` 的现有刷新控制区增加自动刷新开关、预设间隔选择和倒计时展示。
- [ ] 1.2 为 `/admin/dashboard` 增加浏览器本地持久化逻辑，恢复自动刷新启用状态与刷新间隔，并在卸载时正确停止定时器。
- [ ] 1.3 补充自动刷新相关 i18n 文案，确保启用、关闭、倒计时和间隔选项具备明确提示。
+ [x] 1.1 在 `frontend/src/views/admin/DashboardView.vue` 的现有刷新控制区增加自动刷新开关、预设间隔选择和倒计时展示。
+ [x] 1.2 为 `/admin/dashboard` 增加浏览器本地持久化逻辑，恢复自动刷新启用状态与刷新间隔，并在卸载时正确停止定时器。
+ [x] 1.3 补充自动刷新相关 i18n 文案，确保启用、关闭、倒计时和间隔选项具备明确提示。
@@
- [ ] 2.1 为 `DashboardView` 引入统一的整页刷新入口与 in-flight guard，避免自动刷新、手动刷新和筛选切换产生重叠请求。
- [ ] 2.2 让自动刷新复用现有总览数据加载逻辑，并确保刷新始终沿用当前 `startDate`、`endDate` 和 `granularity`。
- [ ] 2.3 在自动刷新成功、手动刷新成功以及筛选触发刷新成功后重置倒计时，保证刷新周期一致。
+ [x] 2.1 为 `DashboardView` 引入统一的整页刷新入口与 in-flight guard，避免自动刷新、手动刷新和筛选切换产生重叠请求。
+ [x] 2.2 让自动刷新复用现有总览数据加载逻辑，并确保刷新始终沿用当前 `startDate`、`endDate` 和 `granularity`。
+ [x] 2.3 在自动刷新成功、手动刷新成功以及筛选触发刷新成功后重置倒计时，保证刷新周期一致。
@@
- [ ] 3.1 扩展 `frontend/src/views/admin/__tests__/DashboardView.spec.ts`，覆盖默认关闭、配置恢复、自动刷新触发和重叠请求保护。
+ [x] 3.1 扩展 `frontend/src/views/admin/__tests__/DashboardView.spec.ts`，覆盖默认关闭、配置恢复、自动刷新触发和重叠请求保护。
  - [ ] 3.2 手动验证 `/admin/dashboard` 的启用/关闭、间隔切换、日期范围切换和手动刷新后的倒计时行为。
```
