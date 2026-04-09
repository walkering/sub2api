## Summary

已完成 LinuxDo Connect 登录自动签到奖励能力实现。

- 新增后台开关 `linuxdo_connect_auto_checkin_bonus_enabled`，默认关闭。
- 保留并补齐 LinuxDo Connect 首登赠送订阅配置，允许为空，默认空数组。
- 在 LinuxDo OAuth 直接登录与邀请码补全注册两条成功路径中，开启开关后自动尝试当日签到，首次成功随机赠送 `1-5` 余额。
- 通过独立奖励记录表和 `(user_id, reward_date, source)` 唯一约束实现每日一次幂等控制。
- 登录响应新增 `auto_checkin_awarded` 与 `auto_checkin_bonus_amount`，前端仅在本次实际发奖时弹出“签到成功，赠送 X 余额”。
- 已补充后端与前端测试，并完成定向验证。

## Modified Files

- `backend/internal/service/domain_constants.go:107-108`：新增 LinuxDo Connect 赠送订阅与自动签到奖励两个设置键。
- `backend/internal/service/settings_view.go:33-34`：在 `SystemSettings` 中增加 LinuxDo Connect 自动签到开关和赠送订阅字段。
- `backend/internal/handler/dto/settings.go:53-54`：为后台设置 DTO 暴露自动签到开关和赠送订阅字段。
- `backend/internal/service/setting_service.go:461,469,813,823,857-858,944-945`：实现设置读写、默认值初始化、LinuxDo 赠送订阅解析，以及自动签到开关读取。
- `backend/internal/handler/admin/setting_handler.go:174-175,812-815`：将两个字段接入管理后台设置接口与变更字段检测。
- `backend/internal/server/api_contract_test.go:474,525`：更新 API 契约默认返回，覆盖新开关和空赠送订阅数组。
- `backend/internal/service/setting_service_update_test.go:183-287`：增加 LinuxDo 赠送订阅保存、去重拒绝和归一化测试。
- `backend/internal/service/linuxdo_auto_checkin_reward.go:1-25`：新增自动签到奖励结果、创建输入和仓储接口定义。
- `backend/internal/repository/linuxdo_auto_checkin_reward_repo.go:1-38`：实现奖励记录插入仓储，并将唯一约束冲突转换为已签到错误。
- `backend/migrations/091_add_linuxdo_auto_checkin_rewards.sql:1-11`：新增奖励记录表、金额约束、唯一约束和查询索引。
- `backend/internal/repository/wire.go:69`：注册自动签到奖励仓储 Provider。
- `backend/internal/service/wire.go:387-402`：在 `AuthService` 装配奖励仓储依赖。
- `backend/cmd/server/wire_gen.go:71-72`：生成后的 Wire 代码接入奖励仓储实例。
- `backend/internal/service/auth_service.go:462-778,853-860`：扩展 LinuxDo OAuth 登录返回值；新增自动签到发奖逻辑、事务包裹、随机 `1-5` 奖励和 LinuxDo Connect 赠送订阅分配。
- `backend/internal/handler/auth_linuxdo_oauth.go:237-241,267-279`：将自动签到结果追加到 OAuth 回调 fragment 与补全注册 JSON 响应。
- `backend/internal/service/auth_service_register_test.go:644-778`：覆盖开关关闭、首次奖励、同日幂等和邀请码补全路径奖励测试。
- `backend/internal/service/admin_service_delete_test.go:111`：为测试桩补充 `UpdateBalance` 支持，满足自动签到余额更新用例。
- `frontend/src/api/admin/settings.ts:32,65,108,136`：同步管理后台设置类型和更新载荷字段。
- `frontend/src/views/admin/SettingsView.vue:1033-1099,2231-2232,2460-2594`：新增自动签到开关 UI、LinuxDo Connect 赠送订阅配置 UI，以及订阅归一化与重复校验复用逻辑。
- `frontend/src/i18n/locales/zh.ts:443`：新增签到成功提示与 LinuxDo 设置文案。
- `frontend/src/i18n/locales/en.ts:444`：新增英文签到成功提示与 LinuxDo 设置文案。
- `frontend/src/api/auth.ts:347-348`：为 LinuxDo OAuth 补全注册响应增加自动签到结果字段。
- `frontend/src/views/auth/LinuxDoCallbackView.vue:109-199`：根据 fragment 或补全注册响应判断是否展示签到成功提示，否则保持普通登录成功提示。
- `frontend/src/views/auth/__tests__/LinuxDoCallbackView.spec.ts:1-112`：增加回调页提示行为测试，覆盖奖励成功与未奖励两种结果。

## Diffs

```diff
diff --git a/backend/internal/service/domain_constants.go b/backend/internal/service/domain_constants.go
@@
+    SettingKeyLinuxDoConnectGiftSubs  = "linuxdo_connect_gift_subscriptions"
+    SettingKeyLinuxDoAutoCheckinBonus = "linuxdo_connect_auto_checkin_bonus_enabled"

diff --git a/backend/internal/service/setting_service.go b/backend/internal/service/setting_service.go
@@
+    updates[SettingKeyLinuxDoAutoCheckinBonus] = strconv.FormatBool(settings.LinuxDoConnectAutoCheckinBonusEnabled)
+    linuxDoGiftSubsJSON, err := json.Marshal(settings.LinuxDoConnectGiftSubscriptions)
+    updates[SettingKeyLinuxDoConnectGiftSubs] = string(linuxDoGiftSubsJSON)
@@
+    func (s *SettingService) GetLinuxDoConnectGiftSubscriptions(ctx context.Context) []DefaultSubscriptionSetting
+    func (s *SettingService) IsLinuxDoConnectAutoCheckinBonusEnabled(ctx context.Context) bool
@@
+    SettingKeyLinuxDoConnectGiftSubs:  "[]",
+    SettingKeyLinuxDoAutoCheckinBonus: "false",

diff --git a/backend/migrations/091_add_linuxdo_auto_checkin_rewards.sql b/backend/migrations/091_add_linuxdo_auto_checkin_rewards.sql
@@
+CREATE TABLE IF NOT EXISTS linuxdo_auto_checkin_rewards (
+    id BIGSERIAL PRIMARY KEY,
+    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
+    reward_date DATE NOT NULL,
+    source VARCHAR(50) NOT NULL,
+    bonus_amount INT NOT NULL CHECK (bonus_amount BETWEEN 1 AND 5),
+    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
+    UNIQUE (user_id, reward_date, source)
+);

diff --git a/backend/internal/service/auth_service.go b/backend/internal/service/auth_service.go
@@
+func (s *AuthService) LoginOrRegisterOAuth(ctx context.Context, email, username string) (string, *User, AutoCheckinResult, error)
+func (s *AuthService) LoginOrRegisterOAuthWithTokenPair(ctx context.Context, email, username, invitationCode string) (*TokenPair, *User, AutoCheckinResult, error)
@@
+autoCheckinResult, err := s.tryLinuxDoConnectAutoCheckinBonus(ctx, user.ID)
+if err != nil {
+    return nil, nil, AutoCheckinResult{}, err
+}
@@
+func (s *AuthService) tryLinuxDoConnectAutoCheckinBonus(ctx context.Context, userID int64) (AutoCheckinResult, error) {
+    bonusAmount := 1 + randInt(5)
+    if err := s.autoCheckinRepo.Create(runCtx, input); err != nil { ... }
+    if err := s.userRepo.UpdateBalance(runCtx, userID, float64(bonusAmount)); err != nil { ... }
+}

diff --git a/backend/internal/handler/auth_linuxdo_oauth.go b/backend/internal/handler/auth_linuxdo_oauth.go
@@
+fragment.Set("auto_checkin_awarded", strconv.FormatBool(autoCheckinResult.Awarded))
+fragment.Set("auto_checkin_bonus_amount", strconv.Itoa(autoCheckinResult.BonusAmount))
@@
+"auto_checkin_awarded":      autoCheckinResult.Awarded,
+"auto_checkin_bonus_amount": autoCheckinResult.BonusAmount,

diff --git a/frontend/src/views/auth/LinuxDoCallbackView.vue b/frontend/src/views/auth/LinuxDoCallbackView.vue
@@
+function showLinuxDoLoginSuccess(awarded: boolean, amount: number) {
+  if (awarded && amount > 0) {
+    appStore.showSuccess(t('auth.linuxdo.autoCheckinSuccess', { amount }))
+    return
+  }
+  appStore.showSuccess(t('auth.loginSuccess'))
+}
@@
+showLinuxDoLoginSuccess(tokenData.auto_checkin_awarded, tokenData.auto_checkin_bonus_amount)
@@
+const autoCheckinAwarded = params.get('auto_checkin_awarded') === 'true'
+const autoCheckinBonusAmount = parseInt(params.get('auto_checkin_bonus_amount') || '0', 10)
+showLinuxDoLoginSuccess(autoCheckinAwarded, autoCheckinBonusAmount)
```
