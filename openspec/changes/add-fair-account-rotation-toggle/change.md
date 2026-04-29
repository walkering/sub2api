## Summary

已完成“账号公平轮转调度开关”变更，实现结果与规格保持一致：

- 系统设置新增 `account_fair_rotation_enabled`，默认 `false`，管理后台可读取、保存并展示中英文文案。
- 开关关闭时，各调度入口继续沿用原有账号选择顺序，不引入额外轮转副作用。
- 开关开启时，仅在当前请求经过既有过滤后得到的最高优先级候选集合内执行公平轮转；不会让低优先级账号越级抢占。
- 粘性会话命中时继续复用原账号，不受公平轮转影响；只有新的非粘性选择才进入轮转逻辑。
- 后端已补设置、轮转策略与多入口回归测试，契约测试同步补齐新字段断言，`tasks.md` 已全部勾选完成。

## Modified Files

### Backend

- `backend/internal/service/domain_constants.go:214`：新增设置键 `SettingKeyAccountFairRotationEnabled`。
- `backend/internal/service/settings_view.go:76`：`SystemSettings` 增加 `AccountFairRotationEnabled` 字段。
- `backend/internal/handler/dto/settings.go:95`：管理端设置 DTO 暴露 `account_fair_rotation_enabled`。
- `backend/internal/service/setting_service.go:114,546,927,1066,1501`：新增开关缓存读取、默认值初始化、保存写回和 `IsAccountFairRotationEnabled` 查询逻辑。
- `backend/internal/handler/admin/setting_handler.go:123,204,287,672,794,965`：管理接口读取、更新、审计 diff 全链路接入公平轮转开关。
- `backend/internal/service/account_fair_rotation.go:17,21,47,67`：新增公平轮转辅助逻辑，按作用域维护轮转游标，并只在最高优先级候选集内轮转。
- `backend/internal/service/gateway_service.go:1320,1543,1749,1776,1852,1888,2834,2968,3108,3147,3281,3422`：Gateway 多个账号选择入口在开关开启时切入公平轮转，关闭时保留旧排序；sticky 复用优先。
- `backend/internal/service/openai_gateway_service.go:1334,1375,1437,1585,1636,1695`：OpenAI fresh/load/fallback 路径统一接入公平轮转辅助逻辑，并保持 sticky 会话优先。
- `backend/internal/service/gemini_messages_compat_service.go:348,379`：Gemini 兼容入口接入同一套公平轮转选择逻辑。
- `backend/cmd/server/wire_gen.go:138,140,183,184`：依赖注入补充 `sessionLimitCache`、`groupRepository` 和 `settingService` 传递，保证新逻辑可构建。
- `backend/internal/server/api_contract_test.go:541,615,680,750,752`：设置接口契约断言补齐 `account_fair_rotation_enabled` 与 `enable_cch_signing`。
- `backend/internal/service/account_fair_rotation_test.go:16,37,59,100`：覆盖轮转顺序、最高优先级约束和设置缓存默认值。
- `backend/internal/service/setting_service_update_test.go:221`：覆盖开关开启/关闭保存与缓存刷新。
- `backend/internal/service/openai_gateway_service_test.go:429,458,481,510`：覆盖 OpenAI 入口的轮转、sticky 保持、高优先级保护和关闭兼容。
- `backend/internal/service/gateway_multiplatform_test.go:3297,3329,3361,3403`：覆盖 Gateway 多平台入口的轮转、sticky 保持和优先级保护。
- `backend/internal/service/gemini_multiplatform_test.go:1163,1195,1223`：覆盖 Gemini 入口的轮转、sticky 保持和优先级保护。

### Frontend

- `frontend/src/api/admin/settings.ts:90,153`：系统设置类型和更新请求新增 `account_fair_rotation_enabled`。
- `frontend/src/views/admin/SettingsView.vue:1436,1439,1443,2386,2741`：系统设置页新增公平轮转开关、默认表单值和保存透传。
- `frontend/src/i18n/locales/zh.ts:4575,4576`：补充中文文案。
- `frontend/src/i18n/locales/en.ts:4417,4418`：补充英文文案。

### OpenSpec

- `openspec/changes/add-fair-account-rotation-toggle/tasks.md:3,4,8,9,10,14,15,19,20,21`：所有任务项已勾选完成。

## Verification

- 通过：`cd backend && go test ./internal/service -tags unit -run "FairRotation|OpenAISelectAccountWithLoadAwareness|GatewayService_SelectAccountWithLoadAwareness|GatewayService_SelectAccountForModelWithExclusions_FairRotationEnabled|GeminiMessagesCompatService_SelectAccountForModelWithExclusions_FairRotationEnabled|GeminiMessagesCompatService_SelectAccountForModelWithExclusions_UsageThreshold|SettingService_UpdateSettings_AccountFairRotationEnabled|SettingService_IsAccountFairRotationEnabled"`
- 通过：`cd backend && go test ./cmd/server -tags unit -run "Wire|ProvideCleanup|ProvideServiceBuildInfo"`
- 通过：`cd backend && go test ./internal/server -tags unit -run "APIContract|Settings"`

## Unified Diff

### Backend: 设置开关接线

```diff
diff --git a/backend/internal/service/domain_constants.go b/backend/internal/service/domain_constants.go
@@
+	// SettingKeyAccountFairRotationEnabled 启用账号雨露均沾轮转（默认 false）
+	SettingKeyAccountFairRotationEnabled = "account_fair_rotation_enabled"

diff --git a/backend/internal/service/setting_service.go b/backend/internal/service/setting_service.go
@@
+var accountFairRotationCache atomic.Value
+var accountFairRotationSF singleflight.Group
+
+updates[SettingKeyAccountFairRotationEnabled] = strconv.FormatBool(settings.AccountFairRotationEnabled)
+result.AccountFairRotationEnabled = settings[SettingKeyAccountFairRotationEnabled] == "true"
+
+func (s *SettingService) IsAccountFairRotationEnabled(ctx context.Context) bool { ... }

diff --git a/backend/internal/handler/admin/setting_handler.go b/backend/internal/handler/admin/setting_handler.go
@@
+	AccountFairRotationEnabled  bool `json:"account_fair_rotation_enabled"`
@@
+		AccountFairRotationEnabled:            req.AccountFairRotationEnabled,
```

### Backend: 调度入口只在最高优先级候选集中公平轮转

```diff
diff --git a/backend/internal/service/account_fair_rotation.go b/backend/internal/service/account_fair_rotation.go
@@
+func buildFairRotationScope(parts ...string) string { ... }
+func fairRotationOrderAccounts(scope string, candidates []*Account, preferOAuth bool) []*Account { ... }
+func fairRotationOrderAccountLoads(scope string, candidates []accountWithLoad, preferOAuth bool) []accountWithLoad { ... }

diff --git a/backend/internal/service/gateway_service.go b/backend/internal/service/gateway_service.go
@@
+fairEnabled := isAccountFairRotationEnabled(s.settingService, ctx)
+orderedAvailable := fairRotationOrderAccountLoads(buildFairRotationScope(...), available, preferOAuth)
+orderedFallback = fairRotationOrderAccounts(buildFairRotationScope(...), fallbackCandidates, preferOAuth)

diff --git a/backend/internal/service/openai_gateway_service.go b/backend/internal/service/openai_gateway_service.go
@@
+fairEnabled := isAccountFairRotationEnabled(s.settingService, ctx)
+ordered := fairRotationOrderAccounts(buildFairRotationScope(...), accounts, preferOAuth)
+orderedAvailable = fairRotationOrderAccountLoads(buildFairRotationScope(...), available, preferOAuth)

diff --git a/backend/internal/service/gemini_messages_compat_service.go b/backend/internal/service/gemini_messages_compat_service.go
@@
+fairEnabled := isAccountFairRotationEnabled(s.settingService, ctx)
+ordered := fairRotationOrderAccounts(buildFairRotationScope(...), candidates, preferOAuth)
```

### Frontend / Tests / OpenSpec

```diff
diff --git a/frontend/src/views/admin/SettingsView.vue b/frontend/src/views/admin/SettingsView.vue
@@
+{{ t('admin.settings.scheduling.accountFairRotation') }}
+<input v-model="form.account_fair_rotation_enabled" type="checkbox" />
+
diff --git a/frontend/src/i18n/locales/zh.ts b/frontend/src/i18n/locales/zh.ts
@@
+accountFairRotation: '账号雨露均沾调度'
+accountFairRotationHint: '开启后，新请求会在同一最高优先级候选账号之间轮转...'
+
diff --git a/backend/internal/service/account_fair_rotation_test.go b/backend/internal/service/account_fair_rotation_test.go
@@
+func TestFairRotationOrderAccounts_RotatesTopPriorityOnly(t *testing.T) { ... }
+func TestFairRotationOrderAccountLoads_RotatesLowestLoadWithinTopPriority(t *testing.T) { ... }
+
diff --git a/openspec/changes/add-fair-account-rotation-toggle/tasks.md b/openspec/changes/add-fair-account-rotation-toggle/tasks.md
@@
- [ ] 4.3 增加回归测试...
+ [x] 4.3 增加回归测试...
```
