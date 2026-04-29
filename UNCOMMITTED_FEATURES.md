# Uncommitted Features Summary (2026-04-11)

**Overview**
- Working tree has modified files plus several new (untracked) files and two OpenSpec change folders.
- Main feature groups found in the uncommitted changes:
- 1) 账号公平轮转调度开关
- 2) 分组账号使用量阈值调度
- 3) 代理一键应用到分组账号
- 4) 错误账号定时清理服务

**Feature 1: 账号公平轮转调度开关**
说明: 新增系统设置 `account_fair_rotation_enabled`，默认关闭。开启后仅在“最高优先级候选集合”内轮转，保留粘性会话优先。

**Primary Backend Files**
- `backend/internal/service/domain_constants.go`
- `backend/internal/service/settings_view.go`
- `backend/internal/handler/dto/settings.go`
- `backend/internal/service/setting_service.go`
- `backend/internal/handler/admin/setting_handler.go`
- `backend/internal/service/account_fair_rotation.go` (new)
- `backend/internal/service/account_fair_rotation_test.go` (new)
- `backend/internal/service/setting_service_update_test.go`
- `backend/internal/server/api_contract_test.go`

**Primary Frontend Files**
- `frontend/src/api/admin/settings.ts`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/i18n/locales/zh.ts` (shared)
- `frontend/src/i18n/locales/en.ts` (shared)

**OpenSpec**
- `openspec/changes/add-fair-account-rotation-toggle/` (new folder)

**Feature 2: 分组账号使用量阈值调度**
说明: 分组新增 `account_usage_threshold_percent`，空值或 `<= 0` 关闭；`> 100` 拒绝。基于总/日/周额度的最大使用比例计算阈值命中，并在调度中区分 `schedulable` / `sticky_only` / `not_schedulable` 三态。

**Primary Backend Files**
- `backend/migrations/093_add_group_account_usage_threshold.sql` (new)
- `backend/ent/schema/group.go`
- `backend/ent/group.go`
- `backend/ent/group/group.go`
- `backend/ent/group/where.go`
- `backend/ent/group_create.go`
- `backend/ent/group_update.go`
- `backend/ent/migrate/schema.go`
- `backend/ent/mutation.go`
- `backend/internal/service/group.go`
- `backend/internal/service/admin_service.go` (shared with Feature 3)
- `backend/internal/handler/admin/group_handler.go`
- `backend/internal/handler/dto/types.go`
- `backend/internal/handler/dto/mappers.go`
- `backend/internal/repository/group_repo.go`
- `backend/internal/repository/api_key_repo.go`
- `backend/internal/service/account.go`
- `backend/internal/service/account_usage_threshold_schedulability.go` (new)
- `backend/internal/service/scheduler_snapshot_service.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/openai_ws_forwarder.go`
- `backend/internal/service/account_quota_usage_threshold_test.go` (new)
- `backend/internal/service/usage_threshold_test_helpers_test.go` (new)
- `backend/internal/service/admin_service_group_test.go`
- `backend/internal/service/openai_gateway_record_usage_test.go`
- `backend/internal/service/openai_ws_protocol_forward_test.go`
- `backend/internal/handler/gateway_handler_warmup_intercept_unit_test.go`
- `backend/internal/repository/scheduler_snapshot_outbox_integration_test.go`

**Primary Frontend Files**
- `frontend/src/types/index.ts` (shared)
- `frontend/src/utils/accountUsageThreshold.ts` (new)
- `frontend/src/utils/__tests__/accountUsageThreshold.spec.ts` (new)
- `frontend/src/components/account/AccountCapacityCell.vue`
- `frontend/src/components/account/AccountStatusIndicator.vue`
- `frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts`
- `frontend/src/views/admin/GroupsView.vue`
- `frontend/src/i18n/locales/zh.ts` (shared)
- `frontend/src/i18n/locales/en.ts` (shared)

**OpenSpec**
- `openspec/changes/add-group-account-usage-threshold-scheduling/` (new folder)

**Feature 3: 代理一键应用到分组账号**
说明: 新增管理接口 `POST /api/v1/admin/proxies/:id/apply-to-groups`，将某个代理批量应用到所选分组的所有账号，返回成功/失败统计。

**Primary Backend Files**
- `backend/internal/handler/admin/proxy_handler.go`
- `backend/internal/server/routes/admin.go`
- `backend/internal/service/admin_service.go` (shared with Feature 2)
- `backend/internal/handler/admin/admin_service_stub_test.go`
- `backend/internal/service/admin_service_bulk_update_test.go`

**Primary Frontend Files**
- `frontend/src/api/admin/proxies.ts`
- `frontend/src/views/admin/ProxiesView.vue`
- `frontend/src/components/admin/proxy/ProxyApplyGroupsModal.vue` (new)
- `frontend/src/types/index.ts` (shared)
- `frontend/src/i18n/locales/zh.ts` (shared)
- `frontend/src/i18n/locales/en.ts` (shared)

**Feature 4: 错误账号定时清理服务**
说明: 后台服务每 5 分钟扫描并删除 `status=error` 的账号，带分页处理并输出日志。

**Primary Backend Files**
- `backend/internal/service/error_account_cleanup_service.go` (new)
- `backend/internal/service/error_account_cleanup_service_test.go` (new)
- `backend/internal/service/wire.go` (shared infra)
- `backend/cmd/server/wire.go` (shared infra)
- `backend/cmd/server/wire_gen.go` (shared infra)
- `backend/cmd/server/wire_gen_test.go` (shared infra)

**Shared / Cross-Feature Files (manual split needed if keeping only one feature)**
- `backend/internal/service/gateway_service.go` (fair rotation + usage threshold)
- `backend/internal/service/openai_gateway_service.go` (fair rotation + usage threshold)
- `backend/internal/service/gemini_messages_compat_service.go` (fair rotation + usage threshold)
- `backend/internal/service/openai_gateway_service_test.go` (fair rotation + usage threshold)
- `backend/internal/service/gateway_multiplatform_test.go` (fair rotation + usage threshold)
- `backend/internal/service/gemini_multiplatform_test.go` (fair rotation + usage threshold)
- `backend/internal/service/wire.go` (usage threshold + error cleanup)
- `backend/cmd/server/wire.go` (usage threshold + error cleanup)
- `backend/cmd/server/wire_gen.go` (usage threshold + error cleanup + gateway dependencies)
- `backend/cmd/server/wire_gen_test.go` (usage threshold + error cleanup)
- `frontend/src/types/index.ts` (usage threshold + proxy apply)
- `frontend/src/i18n/locales/zh.ts` (all three frontend features)
- `frontend/src/i18n/locales/en.ts` (all three frontend features)

**Non-feature / Tooling**
- `.mcp-ssh.lock` (tool state)
- `backend/go.sum` (dependency checksum update)

**Revert/Keep Tips**
- Reverting a whole feature is easiest by restoring the feature’s primary files plus deleting its untracked files.
- If you want to keep one of the two scheduling features (fair rotation or usage threshold), you must manually edit the shared gateway files listed above.
- If you want to keep Feature 2 but drop Feature 3, split `backend/internal/service/admin_service.go` and `frontend/src/types/index.ts` changes by feature before restoring.
