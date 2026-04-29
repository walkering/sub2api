## Summary

已完成“分组账号使用量阈值调度”变更，实现结果与设计保持一致：

- 分组新增 `account_usage_threshold_percent` 配置，空值或 `<= 0` 视为关闭，`> 100` 拒绝写入。
- “用量”按账号总额度、日额度、周额度三类已启用配额中的最大使用比例计算；若日/周周期已到重置时间，则对应 `used` 按 `0` 参与计算。
- 达到阈值后，账号进入三态之一：
  - `schedulable`：未达到阈值，可继续参与新调度。
  - `sticky_only`：达到阈值但仍有活跃会话，仅允许当前粘性会话继续。
  - `not_schedulable`：达到阈值且无活跃会话，立即退出新调度。
- fresh 调度会跳过阈值账号；已有会话不会被强制中断；会话自然结束后，下次调度立即切换到组内其他可用账号。
- 前端已补充分组表单、账号状态提示、容量说明与测试，OpenSpec `tasks.md` 已全部勾选完成。

## Modified Files

### Backend

- `backend/migrations/093_add_group_account_usage_threshold.sql:1`：新增分组阈值字段迁移。
- `backend/ent/schema/group.go:144`：为分组模型增加 `account_usage_threshold_percent`。
- `backend/internal/service/group.go:57`：服务层 `Group` 结构体透出阈值字段。
- `backend/internal/service/admin_service.go:164,200,888,972,1026,1152`：分组创建/更新输入新增阈值字段，统一做 `0 < x <= 100` 归一化与校验。
- `backend/internal/service/admin_service_group_test.go:248,316`：覆盖创建/更新分组时阈值启用、关闭、非法值场景。
- `backend/internal/handler/admin/group_handler.go:112,147,262,313`：管理接口创建/更新请求接收并透传阈值字段。
- `backend/internal/handler/dto/types.go:98`：分组 DTO 返回阈值字段。
- `backend/internal/handler/dto/mappers.go:179`：分组 DTO 映射增加阈值字段。
- `backend/internal/repository/group_repo.go:62,161`：仓储层支持写入、清空阈值字段。
- `backend/internal/service/account.go:1646,1689`：实现最大配额使用比例计算与三态可调度性判断。
- `backend/internal/service/account_usage_threshold_schedulability.go:10,24,49`：抽出分组阈值判断与 sticky/fresh 共用准入逻辑。
- `backend/internal/service/scheduler_snapshot_service.go:634,658,696`：调度快照过滤掉达到阈值且无活跃会话的账号。
- `backend/internal/service/gateway_service.go:1407,1445,1628,1697,2266,2780,2843,2902,2960,3044,3109,3168,3227`：网关 fresh 与 sticky 路径统一接入阈值判定。
- `backend/internal/service/openai_gateway_service.go:1307,1532,1691,1717,1721`：OpenAI 专用调度在 fresh 重查、sticky 复用与 fallback 路径中统一尊重阈值。
- `backend/internal/service/openai_account_scheduler.go:334,597,711,715,738`：OpenAI 调度器在负载均衡与重查路径接入阈值。
- `backend/internal/service/openai_ws_forwarder.go:3843`：`previous_response_id` 粘性命中重查时保留 sticky-only 语义。
- `backend/internal/service/gemini_messages_compat_service.go:62,125,229,246,282,335,443`：Gemini 兼容调度接入同一套阈值判断。
- `backend/internal/service/account_quota_usage_threshold_test.go:9,94`：覆盖最大比例计算与三态判定。
- `backend/internal/service/usage_threshold_test_helpers_test.go:11`：补充活跃会话缓存测试桩。
- `backend/internal/service/openai_gateway_service_test.go:429,474,522,571`：覆盖 OpenAI fresh 跳过、sticky 续用、会话结束后切换、无限额账号不受影响。
- `backend/internal/service/gemini_multiplatform_test.go:985,1034,1079,1125`：覆盖 Gemini 同类场景。

### Frontend

- `frontend/src/types/index.ts:375,507,533`：前端分组类型与创建/更新请求增加阈值字段。
- `frontend/src/utils/accountUsageThreshold.ts:1,84,110`：实现前端最大使用比例与阈值命中状态计算。
- `frontend/src/components/account/AccountCapacityCell.vue:80,101,326`：在容量列展示命中阈值的百分比徽标与 tooltip。
- `frontend/src/components/account/AccountStatusIndicator.vue:155,182,308,368,393`：在账号状态区展示“阈值停调度/阈值仅粘性”状态。
- `frontend/src/views/admin/GroupsView.vue:509,1240,2091,2332,2481,2520,2543,2556,2590,2624,2635,2648`：创建/编辑分组表单新增阈值输入、说明、回填、重置与前端校验。
- `frontend/src/i18n/locales/zh.ts:1702,2211,2297`：补充中文表单说明、容量提示、状态提示。
- `frontend/src/i18n/locales/en.ts:1594,2100,2172`：补充英文表单说明、容量提示、状态提示。
- `frontend/src/utils/__tests__/accountUsageThreshold.spec.ts:1,72,93,114,129`：覆盖重置周期、无上限、sticky_only、not_schedulable、多分组取最严格阈值等规则。
- `frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts:30,132,167,196,222`：修正旧 i18n 断言并补充阈值状态徽标测试。

### OpenSpec

- `openspec/changes/add-group-account-usage-threshold-scheduling/tasks.md:3`：`1.1` 至 `3.3` 全部勾选完成。

## Verification

- 通过：`cd frontend && npm run test:run -- src/utils/__tests__/accountUsageThreshold.spec.ts src/components/account/__tests__/AccountStatusIndicator.spec.ts`
- 通过：`cd frontend && npm run typecheck`
- 已有通过记录，本次未重复执行：`cd backend && go test -tags unit ./internal/service -run "UsageThreshold|StickySession|SelectAccountWithLoadAwareness|SelectBestGeminiAccount"`

## Unified Diff

### Backend: 分组配置、存储与接口透传

```diff
diff --git a/backend/migrations/093_add_group_account_usage_threshold.sql b/backend/migrations/093_add_group_account_usage_threshold.sql
@@
+ALTER TABLE groups
+    ADD COLUMN IF NOT EXISTS account_usage_threshold_percent DECIMAL(10,4);

diff --git a/backend/ent/schema/group.go b/backend/ent/schema/group.go
@@
+		field.Float("account_usage_threshold_percent").
+			Optional().
+			Nillable().
+			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
+			Comment("账号使用量调度阈值百分比，<=0 视为关闭"),

diff --git a/backend/internal/service/admin_service.go b/backend/internal/service/admin_service.go
@@
+	AccountUsageThresholdPercent *float64
@@
+	accountUsageThresholdPercent, err := normalizeAccountUsageThresholdPercent(input.AccountUsageThresholdPercent)
+	if err != nil {
+		return nil, err
+	}
@@
+		AccountUsageThresholdPercent:    accountUsageThresholdPercent,
@@
+func normalizeAccountUsageThresholdPercent(value *float64) (*float64, error) {
+	if value == nil || *value <= 0 {
+		return nil, nil
+	}
+	if *value > 100 {
+		return nil, infraerrors.BadRequest(
+			"GROUP_ACCOUNT_USAGE_THRESHOLD_INVALID",
+			"account_usage_threshold_percent must be between 0 and 100",
+		)
+	}
+	return value, nil
+}

diff --git a/backend/internal/handler/admin/group_handler.go b/backend/internal/handler/admin/group_handler.go
@@
+	AccountUsageThresholdPercent optionalLimitField `json:"account_usage_threshold_percent"`
@@
+		AccountUsageThresholdPercent:    req.AccountUsageThresholdPercent.ToServiceInput(),

diff --git a/backend/internal/handler/dto/types.go b/backend/internal/handler/dto/types.go
@@
+	AccountUsageThresholdPercent *float64 `json:"account_usage_threshold_percent,omitempty"`

diff --git a/backend/internal/repository/group_repo.go b/backend/internal/repository/group_repo.go
@@
-		SetDefaultMappedModel(groupIn.DefaultMappedModel)
+		SetDefaultMappedModel(groupIn.DefaultMappedModel).
+		SetNillableAccountUsageThresholdPercent(groupIn.AccountUsageThresholdPercent)
@@
+	if groupIn.AccountUsageThresholdPercent != nil {
+		builder = builder.SetAccountUsageThresholdPercent(*groupIn.AccountUsageThresholdPercent)
+	} else {
+		builder = builder.ClearAccountUsageThresholdPercent()
+	}
```

### Backend: 调度三态与 sticky-only 续用

```diff
diff --git a/backend/internal/service/account.go b/backend/internal/service/account.go
@@
+func (a *Account) GetMaxQuotaUsageRatio() (float64, bool) {
+	...
+	updateRatio(a.GetQuotaUsed(), a.GetQuotaLimit())
+	dailyUsed := a.GetQuotaDailyUsed()
+	if a.IsDailyQuotaPeriodExpired() {
+		dailyUsed = 0
+	}
+	updateRatio(dailyUsed, a.GetQuotaDailyLimit())
+	weeklyUsed := a.GetQuotaWeeklyUsed()
+	if a.IsWeeklyQuotaPeriodExpired() {
+		weeklyUsed = 0
+	}
+	updateRatio(weeklyUsed, a.GetQuotaWeeklyLimit())
+	return maxRatio, hasRatio
+}
+
+func (a *Account) CheckQuotaUsageThresholdSchedulability(thresholdPercent float64, activeSessions int) WindowCostSchedulability {
+	if thresholdPercent <= 0 {
+		return WindowCostSchedulable
+	}
+	maxRatio, ok := a.GetMaxQuotaUsageRatio()
+	if !ok || maxRatio < thresholdPercent/100 {
+		return WindowCostSchedulable
+	}
+	if activeSessions > 0 {
+		return WindowCostStickyOnly
+	}
+	return WindowCostNotSchedulable
+}

diff --git a/backend/internal/service/account_usage_threshold_schedulability.go b/backend/internal/service/account_usage_threshold_schedulability.go
@@
+func checkAccountUsageThresholdSchedulability(...) WindowCostSchedulability {
+	...
+	activeSessions := 0
+	if sessionLimitCache != nil {
+		count, err := sessionLimitCache.GetActiveSessionCount(ctx, account.ID)
+		if err != nil {
+			return WindowCostSchedulable
+		}
+		activeSessions = count
+	}
+	return account.CheckQuotaUsageThresholdSchedulability(*group.AccountUsageThresholdPercent, activeSessions)
+}
+
+func isAccountAllowedByUsageThreshold(..., isSticky bool) bool {
+	switch checkAccountUsageThresholdSchedulability(...) {
+	case WindowCostSchedulable:
+		return true
+	case WindowCostStickyOnly:
+		return isSticky
+	case WindowCostNotSchedulable:
+		return false
+	default:
+		return true
+	}
+}

diff --git a/backend/internal/service/scheduler_snapshot_service.go b/backend/internal/service/scheduler_snapshot_service.go
@@
-		return filtered, nil
+		return s.filterAccountsForUsageThreshold(ctx, groupID, filtered)
@@
+func (s *SchedulerSnapshotService) filterAccountsForUsageThreshold(ctx context.Context, groupID int64, accounts []Account) ([]Account, error) {
+	...
+	if len(hitAccountIDs) > 0 && s.sessionLimitCache != nil {
+		counts, err := s.sessionLimitCache.GetActiveSessionCountBatch(ctx, hitAccountIDs, idleTimeouts)
+		if err != nil {
+			return accounts, nil
+		}
+		activeSessions = counts
+	}
+	...
+	if schedulability == WindowCostSchedulable {
+		filtered = append(filtered, account)
+	}
+	return filtered, nil
+}

diff --git a/backend/internal/service/openai_gateway_service.go b/backend/internal/service/openai_gateway_service.go
@@
-	account = s.recheckSelectedOpenAIAccountFromDB(ctx, account, requestedModel)
+	account = s.recheckSelectedOpenAIAccountFromDB(ctx, groupID, account, requestedModel, true)
@@
+	if !s.isAccountSchedulableForUsageThreshold(ctx, groupID, acc, false) {
+		continue
+	}
@@
-func (s *OpenAIGatewayService) resolveFreshSchedulableOpenAIAccount(ctx context.Context, account *Account, requestedModel string) *Account {
+func (s *OpenAIGatewayService) resolveFreshSchedulableOpenAIAccount(ctx context.Context, groupID *int64, account *Account, requestedModel string) *Account {
@@
+	if !s.isAccountSchedulableForUsageThreshold(ctx, groupID, fresh, false) {
+		return nil
+	}
@@
+func (s *OpenAIGatewayService) isAccountSchedulableForUsageThreshold(ctx context.Context, groupID *int64, account *Account, isSticky bool) bool {
+	return isAccountAllowedByUsageThreshold(ctx, s.groupRepo, s.sessionLimitCache, groupID, account, isSticky)
+}

diff --git a/backend/internal/service/gemini_messages_compat_service.go b/backend/internal/service/gemini_messages_compat_service.go
@@
+	sessionLimitCache         SessionLimitCache
@@
-	selected := s.selectBestGeminiAccount(ctx, accounts, requestedModel, excludedIDs, platform, useMixedScheduling)
+	selected := s.selectBestGeminiAccount(ctx, groupID, accounts, requestedModel, excludedIDs, platform, useMixedScheduling)
@@
-	if !s.isAccountUsableForRequest(ctx, account, requestedModel, platform, useMixedScheduling) {
+	if !s.isAccountUsableForRequest(ctx, groupID, account, requestedModel, platform, useMixedScheduling, true) {
+		_ = s.cache.DeleteSessionAccountID(ctx, derefGroupID(groupID), cacheKey)
+		return nil
+	}
@@
+	if !s.isAccountSchedulableForUsageThreshold(ctx, groupID, account, isSticky) {
+		return false
+	}
```

### Backend: 测试覆盖

```diff
diff --git a/backend/internal/service/admin_service_group_test.go b/backend/internal/service/admin_service_group_test.go
@@
+func TestAdminService_CreateGroup_AccountUsageThresholdPercent(t *testing.T) { ... }
+func TestAdminService_UpdateGroup_AccountUsageThresholdPercent(t *testing.T) { ... }

diff --git a/backend/internal/service/account_quota_usage_threshold_test.go b/backend/internal/service/account_quota_usage_threshold_test.go
@@
+func TestGetMaxQuotaUsageRatio(t *testing.T) { ... }
+func TestCheckQuotaUsageThresholdSchedulability(t *testing.T) { ... }

diff --git a/backend/internal/service/openai_gateway_service_test.go b/backend/internal/service/openai_gateway_service_test.go
@@
+func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_UsageThresholdSkipsFreshCandidate(t *testing.T) { ... }
+func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_UsageThresholdStickySessionContinues(t *testing.T) { ... }
+func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_UsageThresholdStickySessionEndsThenSwitches(t *testing.T) { ... }
+func TestOpenAIGatewayService_SelectAccountForModelWithExclusions_UsageThresholdIgnoresUnlimitedAccount(t *testing.T) { ... }

diff --git a/backend/internal/service/gemini_multiplatform_test.go b/backend/internal/service/gemini_multiplatform_test.go
@@
+func TestGeminiMessagesCompatService_SelectAccountForModelWithExclusions_UsageThresholdSkipsFreshCandidate(t *testing.T) { ... }
+func TestGeminiMessagesCompatService_SelectAccountForModelWithExclusions_UsageThresholdStickySessionContinues(t *testing.T) { ... }
+func TestGeminiMessagesCompatService_SelectAccountForModelWithExclusions_UsageThresholdStickySessionEndsThenSwitches(t *testing.T) { ... }
+func TestGeminiMessagesCompatService_SelectAccountForModelWithExclusions_UsageThresholdIgnoresUnlimitedAccount(t *testing.T) { ... }
```

### Frontend: 类型、状态计算与展示

```diff
diff --git a/frontend/src/types/index.ts b/frontend/src/types/index.ts
@@
+  account_usage_threshold_percent?: number | null

diff --git a/frontend/src/utils/accountUsageThreshold.ts b/frontend/src/utils/accountUsageThreshold.ts
@@
+export const getAccountMaxQuotaUsageRatio = (...) => {
+  ...
+  const totalRatio = getQuotaRatio(account.quota_used, account.quota_limit)
+  const dailyRatio = getQuotaRatio(
+    getResetAwareUsed(account.quota_daily_used, account.quota_daily_reset_at, nowMs),
+    account.quota_daily_limit,
+  )
+  const weeklyRatio = getQuotaRatio(
+    getResetAwareUsed(account.quota_weekly_used, account.quota_weekly_reset_at, nowMs),
+    account.quota_weekly_limit,
+  )
+  ...
+}
+
+export const getAccountUsageThresholdStatus = (...) => {
+  ...
+  const state: AccountUsageThresholdState = activeSessions > 0 ? 'sticky_only' : 'not_schedulable'
+  ...
+  if (!matched || thresholdPercent < matched.thresholdPercent || (thresholdPercent === matched.thresholdPercent && group.id < matched.groupId)) {
+    matched = candidate
+  }
+  return matched
+}

diff --git a/frontend/src/components/account/AccountCapacityCell.vue b/frontend/src/components/account/AccountCapacityCell.vue
@@
+    <div v-if="usageThresholdStatus" class="flex items-center gap-1">
+      <span :class="[ ..., usageThresholdClass ]" :title="usageThresholdTooltip">
+        <span class="font-mono">{{ formatPercent(usageThresholdStatus.usageRatio * 100) }}%</span>
+      </span>
+    </div>
@@
+import { getAccountUsageThresholdStatus } from '@/utils/accountUsageThreshold'
@@
+const usageThresholdStatus = computed(() => getAccountUsageThresholdStatus(props.account))
+const usageThresholdTooltip = computed(() => { ... })

diff --git a/frontend/src/components/account/AccountStatusIndicator.vue b/frontend/src/components/account/AccountStatusIndicator.vue
@@
+    <div v-if="usageThresholdStatus" class="group relative">
+      <span :class="[ ..., usageThresholdBadgeClass ]">
+        <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
+        {{ usageThresholdText }}
+      </span>
+      <div ...>
+        {{ usageThresholdTooltip }}
+      </div>
+    </div>
@@
+import { getAccountUsageThresholdStatus } from '@/utils/accountUsageThreshold'
@@
+const usageThresholdStatus = computed(() => getAccountUsageThresholdStatus(props.account))
+const usageThresholdText = computed(() => ...)
+const usageThresholdTooltip = computed(() => ...)
```

### Frontend: 分组表单、文案与测试

```diff
diff --git a/frontend/src/views/admin/GroupsView.vue b/frontend/src/views/admin/GroupsView.vue
@@
+        <div class="border-t pt-4">
+          <label class="input-label">{{ t('admin.groups.usageThreshold.label') }}</label>
+          <input
+            v-model.number="createForm.account_usage_threshold_percent"
+            type="number"
+            step="0.1"
+            min="0"
+            max="100"
+            class="input"
+            :placeholder="t('admin.groups.usageThreshold.placeholder')"
+          />
+          <p class="input-hint">{{ t('admin.groups.usageThreshold.hint') }}</p>
+        </div>
@@
+const normalizeOptionalPercent = (value: number | string | null | undefined): number | null => {
+  ...
+}
@@
+  const accountUsageThresholdPercent = normalizeOptionalPercent(
+    createForm.account_usage_threshold_percent as number | string | null
+  )
+  if (accountUsageThresholdPercent !== null && accountUsageThresholdPercent > 100) {
+    appStore.showError(t('admin.groups.usageThreshold.invalid'))
+    return
+  }
@@
+      account_usage_threshold_percent: accountUsageThresholdPercent,

diff --git a/frontend/src/i18n/locales/zh.ts b/frontend/src/i18n/locales/zh.ts
@@
+      usageThreshold: {
+        label: '账号使用量调度阈值',
+        placeholder: '留空或 <= 0 表示关闭',
+        hint: '按账号总额度、日额度、周额度中最高使用比例判断。达到阈值后不再参与新调度；若仍有活跃会话，仅允许当前会话继续。',
+        invalid: '账号使用量调度阈值必须大于 0 且不能超过 100'
+      },
@@
+        usageThreshold: {
+          blocked: '账号在分组 {group} 的使用量已达 {usage}%（阈值 {threshold}%），当前无活跃会话，已停止新调度',
+          stickyOnly: '账号在分组 {group} 的使用量已达 {usage}%（阈值 {threshold}%），仍有活跃会话，仅允许当前会话继续'
+        },
@@
+        usageThresholdStopped: '阈值停调度',
+        usageThresholdStickyOnly: '阈值仅粘性',
+        usageThresholdStoppedHint: '分组 {group} 使用量 {usage}% 已达到阈值 {threshold}%，当前无活跃会话，账号不再参与新调度',
+        usageThresholdStickyOnlyHint: '分组 {group} 使用量 {usage}% 已达到阈值 {threshold}%，但仍有活跃会话，仅允许当前会话继续',

diff --git a/frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts b/frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts
@@
-    expect(wrapper.text()).toContain('account.creditsExhausted')
+    expect(wrapper.text()).toContain('admin.accounts.status.creditsExhausted')
@@
+  it('达到使用量阈值且仍有活跃会话时显示 sticky_only 标识', () => { ... })
+  it('达到使用量阈值且无活跃会话时显示 stopped 标识', () => { ... })

diff --git a/frontend/src/utils/__tests__/accountUsageThreshold.spec.ts b/frontend/src/utils/__tests__/accountUsageThreshold.spec.ts
@@
+  it('日/周配额已到重置时间时按 0 参与最大比例计算', () => { ... })
+  it('没有任何可计算配额上限时返回 null', () => { ... })
+  it('达到阈值且仍有活跃会话时返回 sticky_only', () => { ... })
+  it('达到阈值且无活跃会话时返回 not_schedulable', () => { ... })
+  it('多分组命中时选择最严格阈值', () => { ... })
+  it('阈值相同且都命中时选择较小的分组 ID', () => { ... })
```

### OpenSpec: 任务回填

```diff
diff --git a/openspec/changes/add-group-account-usage-threshold-scheduling/tasks.md b/openspec/changes/add-group-account-usage-threshold-scheduling/tasks.md
@@
- [ ] 2.1 基于账号总额度、日额度、周额度实现“最大使用比例”计算与阈值命中判定
- [ ] 2.2 在调度快照中过滤达到阈值且无活跃会话的账号，禁止其参与新的调度
- [ ] 2.3 在网关粘性账号复用路径中支持“达到阈值但仍有活跃会话”的 sticky-only 续用
- [ ] 2.4 补充调度测试，覆盖新请求避开阈值账号、进行中会话继续、会话结束后切换账号和无配额上限账号
- [ ] 3.1 在分组管理界面增加账号使用量阈值输入、说明文案与前端校验
- [ ] 3.2 在账号状态展示中补充“因使用量阈值仅允许粘性会话/停止调度”的可见反馈
- [ ] 3.3 更新相关文档或操作说明，明确阈值的计算口径、关闭规则和会话续用行为
+ [x] 2.1 基于账号总额度、日额度、周额度实现“最大使用比例”计算与阈值命中判定
+ [x] 2.2 在调度快照中过滤达到阈值且无活跃会话的账号，禁止其参与新的调度
+ [x] 2.3 在网关粘性账号复用路径中支持“达到阈值但仍有活跃会话”的 sticky-only 续用
+ [x] 2.4 补充调度测试，覆盖新请求避开阈值账号、进行中会话继续、会话结束后切换账号和无配额上限账号
+ [x] 3.1 在分组管理界面增加账号使用量阈值输入、说明文案与前端校验
+ [x] 3.2 在账号状态展示中补充“因使用量阈值仅允许粘性会话/停止调度”的可见反馈
+ [x] 3.3 更新相关文档或操作说明，明确阈值的计算口径、关闭规则和会话续用行为
```
