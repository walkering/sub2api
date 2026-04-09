# Change Record

## Modified Files

- `backend/internal/service/domain_constants.go:107`
  - 新增 `SettingKeyLinuxDoConnectGiftSubs` 设置键。
- `backend/internal/service/settings_view.go:33`
  - 在服务层 `SystemSettings` 视图中加入 `LinuxDoConnectGiftSubscriptions`。
- `backend/internal/handler/dto/settings.go:53`
  - 在管理后台设置 DTO 中暴露 `linuxdo_connect_gift_subscriptions`。
- `backend/internal/service/setting_service.go:408`
  - 复用订阅校验逻辑处理默认订阅和 LinuxDo Connect 赠送订阅。
- `backend/internal/service/setting_service.go:464`
  - 在设置持久化时写入 `linuxdo_connect_gift_subscriptions`。
- `backend/internal/service/setting_service.go:810`
  - 新增 `GetLinuxDoConnectGiftSubscriptions` 读取方法。
- `backend/internal/service/setting_service.go:846`
  - 初始化默认设置时将 LinuxDo Connect 赠送订阅默认值设为 `[]`。
- `backend/internal/service/setting_service.go:932`
  - 解析系统设置时返回 LinuxDo Connect 赠送订阅。
- `backend/internal/handler/admin/setting_handler.go:74`
  - 管理后台读取设置时返回 LinuxDo Connect 赠送订阅。
- `backend/internal/handler/admin/setting_handler.go:173`
  - 管理后台更新设置请求新增 `linuxdo_connect_gift_subscriptions` 字段。
- `backend/internal/handler/admin/setting_handler.go:256`
  - 更新设置时对 LinuxDo Connect 赠送订阅做归一化处理。
- `backend/internal/handler/admin/setting_handler.go:517`
  - 将 LinuxDo Connect 赠送订阅映射到服务层设置对象。
- `backend/internal/handler/admin/setting_handler.go:657`
  - 更新设置响应返回 LinuxDo Connect 赠送订阅。
- `backend/internal/handler/admin/setting_handler.go:807`
  - diff 逻辑支持识别 `linuxdo_connect_gift_subscriptions` 变更。
- `backend/internal/service/auth_service.go:505`
  - LinuxDo OAuth 首次直登创建用户后发放 LinuxDo Connect 赠送订阅。
- `backend/internal/service/auth_service.go:635`
  - LinuxDo OAuth 邀请码补全注册创建用户后发放 LinuxDo Connect 赠送订阅。
- `backend/internal/service/auth_service.go:754`
  - 抽取 `assignLinuxDoConnectGiftSubscriptions` / `assignConfiguredSubscriptions` 复用发放逻辑。
- `backend/internal/service/setting_service_update_test.go:183`
  - 新增 LinuxDo Connect 赠送订阅设置合法性、默认值与读取归一化测试。
- `backend/internal/service/admin_service_delete_test.go:15`
  - 扩展 `userRepoStub` / `redeemRepoStub`，支持邀请码补全注册链路所需行为。
- `backend/internal/service/auth_service_register_test.go:65`
  - 新增 `refreshTokenCacheStub` 支撑 OAuth token pair 测试。
- `backend/internal/service/auth_service_register_test.go:511`
  - 新增 LinuxDo Connect 赠送订阅首次注册、空配置、不重复发放、邀请码补全注册测试。
- `backend/internal/server/api_contract_test.go:523`
  - API 合约增加 `linuxdo_connect_gift_subscriptions` 空数组字段。
- `frontend/src/api/admin/settings.ts:32`
  - 前端设置类型新增 `linuxdo_connect_gift_subscriptions`。
- `frontend/src/views/admin/SettingsView.vue:1028`
  - 在 LinuxDo Connect 设置区新增赠送订阅编辑 UI。
- `frontend/src/views/admin/SettingsView.vue:2217`
  - 表单默认值增加 `linuxdo_connect_gift_subscriptions: []`。
- `frontend/src/views/admin/SettingsView.vue:2403`
  - 抽取订阅归一化、去重和表单应用逻辑，复用于默认订阅和 LinuxDo Connect 赠送订阅。
- `frontend/src/views/admin/SettingsView.vue:2470`
  - 新增 LinuxDo Connect 赠送订阅增删操作。
- `frontend/src/views/admin/SettingsView.vue:2481`
  - 保存设置时校验 LinuxDo Connect 赠送订阅重复分组并提交到后端。
- `frontend/src/i18n/locales/zh.ts:4395`
  - 新增 LinuxDo Connect 赠送订阅中文文案。
- `frontend/src/i18n/locales/en.ts:4229`
  - 新增 LinuxDo Connect 赠送订阅英文文案。

## Verification

- `gofmt -w backend/internal/service/domain_constants.go backend/internal/service/settings_view.go backend/internal/handler/dto/settings.go backend/internal/service/setting_service.go backend/internal/handler/admin/setting_handler.go backend/internal/service/auth_service.go backend/internal/service/setting_service_update_test.go backend/internal/service/admin_service_delete_test.go backend/internal/service/auth_service_register_test.go backend/internal/server/api_contract_test.go`
- `go test ./internal/service -tags unit`
- `go test ./internal/server -tags unit`
- `npm run typecheck`

## Unified Diff

### Backend

```diff
diff --git a/backend/internal/handler/admin/setting_handler.go b/backend/internal/handler/admin/setting_handler.go
index 06916917..97dd1c9d 100644
--- a/backend/internal/handler/admin/setting_handler.go
+++ b/backend/internal/handler/admin/setting_handler.go
@@ -71,6 +71,13 @@ func (h *SettingHandler) GetSettings(c *gin.Context) {
 			ValidityDays: sub.ValidityDays,
 		})
 	}
+	linuxDoGiftSubscriptions := make([]dto.DefaultSubscriptionSetting, 0, len(settings.LinuxDoConnectGiftSubscriptions))
+	for _, sub := range settings.LinuxDoConnectGiftSubscriptions {
+		linuxDoGiftSubscriptions = append(linuxDoGiftSubscriptions, dto.DefaultSubscriptionSetting{
+			GroupID:      sub.GroupID,
+			ValidityDays: sub.ValidityDays,
+		})
+	}
 
 	response.Success(c, dto.SystemSettings{
 		RegistrationEnabled:                  settings.RegistrationEnabled,
@@ -96,6 +103,7 @@ func (h *SettingHandler) GetSettings(c *gin.Context) {
 		LinuxDoConnectClientID:               settings.LinuxDoConnectClientID,
 		LinuxDoConnectClientSecretConfigured: settings.LinuxDoConnectClientSecretConfigured,
 		LinuxDoConnectRedirectURL:            settings.LinuxDoConnectRedirectURL,
+		LinuxDoConnectGiftSubscriptions:      linuxDoGiftSubscriptions,
 		SiteName:                             settings.SiteName,
 		SiteLogo:                             settings.SiteLogo,
 		SiteSubtitle:                         settings.SiteSubtitle,
@@ -158,10 +166,11 @@ type UpdateSettingsRequest struct {
 	TurnstileSecretKey string `json:"turnstile_secret_key"`
 
 	// LinuxDo Connect OAuth 登录
-	LinuxDoConnectEnabled      bool   `json:"linuxdo_connect_enabled"`
-	LinuxDoConnectClientID     string `json:"linuxdo_connect_client_id"`
-	LinuxDoConnectClientSecret string `json:"linuxdo_connect_client_secret"`
-	LinuxDoConnectRedirectURL  string `json:"linuxdo_connect_redirect_url"`
+	LinuxDoConnectEnabled           bool                             `json:"linuxdo_connect_enabled"`
+	LinuxDoConnectClientID          string                           `json:"linuxdo_connect_client_id"`
+	LinuxDoConnectClientSecret      string                           `json:"linuxdo_connect_client_secret"`
+	LinuxDoConnectRedirectURL       string                           `json:"linuxdo_connect_redirect_url"`
+	LinuxDoConnectGiftSubscriptions []dto.DefaultSubscriptionSetting `json:"linuxdo_connect_gift_subscriptions"`
 
 	// OEM设置
 	SiteName                    string                `json:"site_name"`
@@ -243,7 +252,8 @@ func (h *SettingHandler) UpdateSettings(c *gin.Context) {
 	if req.SMTPPort <= 0 {
 		req.SMTPPort = 587
 	}
-	req.DefaultSubscriptions = normalizeDefaultSubscriptions(req.DefaultSubscriptions)
+	req.DefaultSubscriptions = normalizeSubscriptionSettings(req.DefaultSubscriptions)
+	req.LinuxDoConnectGiftSubscriptions = normalizeSubscriptionSettings(req.LinuxDoConnectGiftSubscriptions)
 
 	// SMTP 配置保护：如果请求中 smtp_host 为空但数据库中已有配置，则保留已有 SMTP 配置
 	// 防止前端加载设置失败时空表单覆盖已保存的 SMTP 配置
@@ -504,6 +514,13 @@ func (h *SettingHandler) UpdateSettings(c *gin.Context) {
 			ValidityDays: sub.ValidityDays,
 		})
 	}
+	linuxDoGiftSubscriptions := make([]service.DefaultSubscriptionSetting, 0, len(req.LinuxDoConnectGiftSubscriptions))
+	for _, sub := range req.LinuxDoConnectGiftSubscriptions {
+		linuxDoGiftSubscriptions = append(linuxDoGiftSubscriptions, service.DefaultSubscriptionSetting{
+			GroupID:      sub.GroupID,
+			ValidityDays: sub.ValidityDays,
+		})
+	}
 
 	// 验证最低版本号格式（空字符串=禁用，或合法 semver）
 	if req.MinClaudeCodeVersion != "" {
@@ -552,6 +569,7 @@ func (h *SettingHandler) UpdateSettings(c *gin.Context) {
 		LinuxDoConnectClientID:           req.LinuxDoConnectClientID,
 		LinuxDoConnectClientSecret:       req.LinuxDoConnectClientSecret,
 		LinuxDoConnectRedirectURL:        req.LinuxDoConnectRedirectURL,
+		LinuxDoConnectGiftSubscriptions:  linuxDoGiftSubscriptions,
 		SiteName:                         req.SiteName,
 		SiteLogo:                         req.SiteLogo,
 		SiteSubtitle:                     req.SiteSubtitle,
@@ -636,6 +654,13 @@ func (h *SettingHandler) UpdateSettings(c *gin.Context) {
 			ValidityDays: sub.ValidityDays,
 		})
 	}
+	updatedLinuxDoGiftSubscriptions := make([]dto.DefaultSubscriptionSetting, 0, len(updatedSettings.LinuxDoConnectGiftSubscriptions))
+	for _, sub := range updatedSettings.LinuxDoConnectGiftSubscriptions {
+		updatedLinuxDoGiftSubscriptions = append(updatedLinuxDoGiftSubscriptions, dto.DefaultSubscriptionSetting{
+			GroupID:      sub.GroupID,
+			ValidityDays: sub.ValidityDays,
+		})
+	}
 
 	response.Success(c, dto.SystemSettings{
 		RegistrationEnabled:                  updatedSettings.RegistrationEnabled,
@@ -661,6 +686,7 @@ func (h *SettingHandler) UpdateSettings(c *gin.Context) {
 		LinuxDoConnectClientID:               updatedSettings.LinuxDoConnectClientID,
 		LinuxDoConnectClientSecretConfigured: updatedSettings.LinuxDoConnectClientSecretConfigured,
 		LinuxDoConnectRedirectURL:            updatedSettings.LinuxDoConnectRedirectURL,
+		LinuxDoConnectGiftSubscriptions:      updatedLinuxDoGiftSubscriptions,
 		SiteName:                             updatedSettings.SiteName,
 		SiteLogo:                             updatedSettings.SiteLogo,
 		SiteSubtitle:                         updatedSettings.SiteSubtitle,
@@ -778,6 +804,9 @@ func diffSettings(before *service.SystemSettings, after *service.SystemSettings,
 	if before.LinuxDoConnectRedirectURL != after.LinuxDoConnectRedirectURL {
 		changed = append(changed, "linuxdo_connect_redirect_url")
 	}
+	if !equalSubscriptionSettings(before.LinuxDoConnectGiftSubscriptions, after.LinuxDoConnectGiftSubscriptions) {
+		changed = append(changed, "linuxdo_connect_gift_subscriptions")
+	}
 	if before.SiteName != after.SiteName {
 		changed = append(changed, "site_name")
 	}
@@ -808,7 +837,7 @@ func diffSettings(before *service.SystemSettings, after *service.SystemSettings,
 	if before.DefaultBalance != after.DefaultBalance {
 		changed = append(changed, "default_balance")
 	}
-	if !equalDefaultSubscriptions(before.DefaultSubscriptions, after.DefaultSubscriptions) {
+	if !equalSubscriptionSettings(before.DefaultSubscriptions, after.DefaultSubscriptions) {
 		changed = append(changed, "default_subscriptions")
 	}
 	if before.EnableModelFallback != after.EnableModelFallback {
@@ -874,7 +903,7 @@ func diffSettings(before *service.SystemSettings, after *service.SystemSettings,
 	return changed
 }
 
-func normalizeDefaultSubscriptions(input []dto.DefaultSubscriptionSetting) []dto.DefaultSubscriptionSetting {
+func normalizeSubscriptionSettings(input []dto.DefaultSubscriptionSetting) []dto.DefaultSubscriptionSetting {
 	if len(input) == 0 {
 		return nil
 	}
@@ -891,6 +920,10 @@ func normalizeDefaultSubscriptions(input []dto.DefaultSubscriptionSetting) []dto
 	return normalized
 }
 
+func normalizeDefaultSubscriptions(input []dto.DefaultSubscriptionSetting) []dto.DefaultSubscriptionSetting {
+	return normalizeSubscriptionSettings(input)
+}
+
 func equalStringSlice(a, b []string) bool {
 	if len(a) != len(b) {
 		return false
@@ -903,7 +936,7 @@ func equalStringSlice(a, b []string) bool {
 	return true
 }
 
-func equalDefaultSubscriptions(a, b []service.DefaultSubscriptionSetting) bool {
+func equalSubscriptionSettings(a, b []service.DefaultSubscriptionSetting) bool {
 	if len(a) != len(b) {
 		return false
 	}
@@ -915,6 +948,10 @@ func equalDefaultSubscriptions(a, b []service.DefaultSubscriptionSetting) bool {
 	return true
 }
 
+func equalDefaultSubscriptions(a, b []service.DefaultSubscriptionSetting) bool {
+	return equalSubscriptionSettings(a, b)
+}
+
 // TestSMTPRequest 测试SMTP连接请求
 type TestSMTPRequest struct {
 	SMTPHost     string `json:"smtp_host"`
diff --git a/backend/internal/handler/dto/settings.go b/backend/internal/handler/dto/settings.go
index acc1129c..e91c35ba 100644
--- a/backend/internal/handler/dto/settings.go
+++ b/backend/internal/handler/dto/settings.go
@@ -46,10 +46,11 @@ type SystemSettings struct {
 	TurnstileSiteKey             string `json:"turnstile_site_key"`
 	TurnstileSecretKeyConfigured bool   `json:"turnstile_secret_key_configured"`
 
-	LinuxDoConnectEnabled                bool   `json:"linuxdo_connect_enabled"`
-	LinuxDoConnectClientID               string `json:"linuxdo_connect_client_id"`
-	LinuxDoConnectClientSecretConfigured bool   `json:"linuxdo_connect_client_secret_configured"`
-	LinuxDoConnectRedirectURL            string `json:"linuxdo_connect_redirect_url"`
+	LinuxDoConnectEnabled                bool                         `json:"linuxdo_connect_enabled"`
+	LinuxDoConnectClientID               string                       `json:"linuxdo_connect_client_id"`
+	LinuxDoConnectClientSecretConfigured bool                         `json:"linuxdo_connect_client_secret_configured"`
+	LinuxDoConnectRedirectURL            string                       `json:"linuxdo_connect_redirect_url"`
+	LinuxDoConnectGiftSubscriptions      []DefaultSubscriptionSetting `json:"linuxdo_connect_gift_subscriptions"`
 
 	SiteName                    string           `json:"site_name"`
 	SiteLogo                    string           `json:"site_logo"`
diff --git a/backend/internal/server/api_contract_test.go b/backend/internal/server/api_contract_test.go
index d412ea34..f3df8d64 100644
--- a/backend/internal/server/api_contract_test.go
+++ b/backend/internal/server/api_contract_test.go
@@ -520,6 +520,7 @@ func TestAPIContracts(t *testing.T) {
 					"default_concurrency": 5,
 					"default_balance": 1.25,
 					"default_subscriptions": [],
+					"linuxdo_connect_gift_subscriptions": [],
 					"enable_model_fallback": false,
 					"fallback_model_anthropic": "claude-3-5-sonnet-20241022",
 					"fallback_model_antigravity": "gemini-2.5-pro",
diff --git a/backend/internal/service/admin_service_delete_test.go b/backend/internal/service/admin_service_delete_test.go
index fbc856cf..b207d89e 100644
--- a/backend/internal/service/admin_service_delete_test.go
+++ b/backend/internal/service/admin_service_delete_test.go
@@ -13,15 +13,19 @@ import (
 )
 
 type userRepoStub struct {
-	user       *User
-	getErr     error
-	createErr  error
-	deleteErr  error
-	exists     bool
-	existsErr  error
-	nextID     int64
-	created    []*User
-	deletedIDs []int64
+	user          *User
+	getErr        error
+	getByEmailErr error
+	createErr     error
+	deleteErr     error
+	updateErr     error
+	exists        bool
+	existsErr     error
+	nextID        int64
+	created       []*User
+	updated       []*User
+	deletedIDs    []int64
+	userByEmail   map[string]*User
 }
 
 func (s *userRepoStub) Create(ctx context.Context, user *User) error {
@@ -32,6 +36,13 @@ func (s *userRepoStub) Create(ctx context.Context, user *User) error {
 		user.ID = s.nextID
 	}
 	s.created = append(s.created, user)
+	if s.userByEmail == nil {
+		s.userByEmail = make(map[string]*User)
+	}
+	s.userByEmail[user.Email] = user
+	if s.user == nil {
+		s.user = user
+	}
 	return nil
 }
 
@@ -46,7 +57,18 @@ func (s *userRepoStub) GetByID(ctx context.Context, id int64) (*User, error) {
 }
 
 func (s *userRepoStub) GetByEmail(ctx context.Context, email string) (*User, error) {
-	panic("unexpected GetByEmail call")
+	if s.getByEmailErr != nil {
+		return nil, s.getByEmailErr
+	}
+	if s.userByEmail != nil {
+		if user, ok := s.userByEmail[email]; ok {
+			return user, nil
+		}
+	}
+	if s.user != nil && s.user.Email == email {
+		return s.user, nil
+	}
+	return nil, ErrUserNotFound
 }
 
 func (s *userRepoStub) GetFirstAdmin(ctx context.Context) (*User, error) {
@@ -54,7 +76,16 @@ func (s *userRepoStub) GetFirstAdmin(ctx context.Context) (*User, error) {
 }
 
 func (s *userRepoStub) Update(ctx context.Context, user *User) error {
-	panic("unexpected Update call")
+	if s.updateErr != nil {
+		return s.updateErr
+	}
+	s.updated = append(s.updated, user)
+	if s.userByEmail == nil {
+		s.userByEmail = make(map[string]*User)
+	}
+	s.userByEmail[user.Email] = user
+	s.user = user
+	return nil
 }
 
 func (s *userRepoStub) Delete(ctx context.Context, id int64) error {
@@ -86,6 +117,14 @@ func (s *userRepoStub) ExistsByEmail(ctx context.Context, email string) (bool, e
 	if s.existsErr != nil {
 		return false, s.existsErr
 	}
+	if s.userByEmail != nil {
+		if _, ok := s.userByEmail[email]; ok {
+			return true, nil
+		}
+	}
+	if s.user != nil && s.user.Email == email {
+		return true, nil
+	}
 	return s.exists, nil
 }
 
@@ -250,6 +289,10 @@ func (s *proxyRepoStub) ListAccountSummariesByProxyID(ctx context.Context, proxy
 type redeemRepoStub struct {
 	deleteErrByID map[int64]error
 	deletedIDs    []int64
+	codesByCode   map[string]*RedeemCode
+	usedCodeIDs   []int64
+	usedUserIDs   []int64
+	useErr        error
 }
 
 func (s *redeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
@@ -265,7 +308,12 @@ func (s *redeemRepoStub) GetByID(ctx context.Context, id int64) (*RedeemCode, er
 }
 
 func (s *redeemRepoStub) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
-	panic("unexpected GetByCode call")
+	if s.codesByCode != nil {
+		if redeemCode, ok := s.codesByCode[code]; ok {
+			return redeemCode, nil
+		}
+	}
+	return nil, ErrRedeemCodeNotFound
 }
 
 func (s *redeemRepoStub) Update(ctx context.Context, code *RedeemCode) error {
@@ -283,7 +331,22 @@ func (s *redeemRepoStub) Delete(ctx context.Context, id int64) error {
 }
 
 func (s *redeemRepoStub) Use(ctx context.Context, id, userID int64) error {
-	panic("unexpected Use call")
+	s.usedCodeIDs = append(s.usedCodeIDs, id)
+	s.usedUserIDs = append(s.usedUserIDs, userID)
+	if s.useErr != nil {
+		return s.useErr
+	}
+	for _, redeemCode := range s.codesByCode {
+		if redeemCode.ID != id {
+			continue
+		}
+		redeemCode.Status = StatusUsed
+		redeemCode.UsedBy = &userID
+		now := time.Now()
+		redeemCode.UsedAt = &now
+		return nil
+	}
+	return ErrRedeemCodeNotFound
 }
 
 func (s *redeemRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
diff --git a/backend/internal/service/auth_service.go b/backend/internal/service/auth_service.go
index 6e524fb9..976cf94f 100644
--- a/backend/internal/service/auth_service.go
+++ b/backend/internal/service/auth_service.go
@@ -502,6 +502,7 @@ func (s *AuthService) LoginOrRegisterOAuth(ctx context.Context, email, username
 			} else {
 				user = newUser
 				s.assignDefaultSubscriptions(ctx, user.ID)
+				s.assignLinuxDoConnectGiftSubscriptions(ctx, user.ID)
 			}
 		} else {
 			logger.LegacyPrintf("service.auth", "[Auth] Database error during oauth login: %v", err)
@@ -631,6 +632,7 @@ func (s *AuthService) LoginOrRegisterOAuthWithTokenPair(ctx context.Context, ema
 					}
 					user = newUser
 					s.assignDefaultSubscriptions(ctx, user.ID)
+					s.assignLinuxDoConnectGiftSubscriptions(ctx, user.ID)
 				}
 			} else {
 				if err := s.userRepo.Create(ctx, newUser); err != nil {
@@ -647,6 +649,7 @@ func (s *AuthService) LoginOrRegisterOAuthWithTokenPair(ctx context.Context, ema
 				} else {
 					user = newUser
 					s.assignDefaultSubscriptions(ctx, user.ID)
+					s.assignLinuxDoConnectGiftSubscriptions(ctx, user.ID)
 					if invitationRedeemCode != nil {
 						if err := s.redeemRepo.Use(ctx, invitationRedeemCode.ID, user.ID); err != nil {
 							return nil, nil, ErrInvitationCodeInvalid
@@ -736,18 +739,43 @@ func (s *AuthService) VerifyPendingOAuthToken(tokenStr string) (email, username
 }
 
 func (s *AuthService) assignDefaultSubscriptions(ctx context.Context, userID int64) {
-	if s.settingService == nil || s.defaultSubAssigner == nil || userID <= 0 {
+	if s.settingService == nil {
+		return
+	}
+	s.assignConfiguredSubscriptions(
+		ctx,
+		userID,
+		s.settingService.GetDefaultSubscriptions(ctx),
+		"auto assigned by default user subscriptions setting",
+		"default",
+	)
+}
+
+func (s *AuthService) assignLinuxDoConnectGiftSubscriptions(ctx context.Context, userID int64) {
+	if s.settingService == nil {
+		return
+	}
+	s.assignConfiguredSubscriptions(
+		ctx,
+		userID,
+		s.settingService.GetLinuxDoConnectGiftSubscriptions(ctx),
+		"auto assigned by linuxdo connect gift subscriptions setting",
+		"linuxdo connect gift",
+	)
+}
+
+func (s *AuthService) assignConfiguredSubscriptions(ctx context.Context, userID int64, items []DefaultSubscriptionSetting, notes, logLabel string) {
+	if s.defaultSubAssigner == nil || userID <= 0 || len(items) == 0 {
 		return
 	}
-	items := s.settingService.GetDefaultSubscriptions(ctx)
 	for _, item := range items {
 		if _, _, err := s.defaultSubAssigner.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
 			UserID:       userID,
 			GroupID:      item.GroupID,
 			ValidityDays: item.ValidityDays,
-			Notes:        "auto assigned by default user subscriptions setting",
+			Notes:        notes,
 		}); err != nil {
-			logger.LegacyPrintf("service.auth", "[Auth] Failed to assign default subscription: user_id=%d group_id=%d err=%v", userID, item.GroupID, err)
+			logger.LegacyPrintf("service.auth", "[Auth] Failed to assign %s subscription: user_id=%d group_id=%d err=%v", logLabel, userID, item.GroupID, err)
 		}
 	}
 }
diff --git a/backend/internal/service/auth_service_register_test.go b/backend/internal/service/auth_service_register_test.go
index 7b50e90d..c608f9cd 100644
--- a/backend/internal/service/auth_service_register_test.go
+++ b/backend/internal/service/auth_service_register_test.go
@@ -62,6 +62,8 @@ type defaultSubscriptionAssignerStub struct {
 	err   error
 }
 
+type refreshTokenCacheStub struct{}
+
 func (s *defaultSubscriptionAssignerStub) AssignOrExtendSubscription(_ context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
 	if input != nil {
 		s.calls = append(s.calls, *input)
@@ -107,11 +109,52 @@ func (s *emailCacheStub) SetPasswordResetEmailCooldown(ctx context.Context, emai
 	return nil
 }
 
+func (s *refreshTokenCacheStub) StoreRefreshToken(ctx context.Context, tokenHash string, data *RefreshTokenData, ttl time.Duration) error {
+	return nil
+}
+
+func (s *refreshTokenCacheStub) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenData, error) {
+	return nil, ErrRefreshTokenNotFound
+}
+
+func (s *refreshTokenCacheStub) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
+	return nil
+}
+
+func (s *refreshTokenCacheStub) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
+	return nil
+}
+
+func (s *refreshTokenCacheStub) DeleteTokenFamily(ctx context.Context, familyID string) error {
+	return nil
+}
+
+func (s *refreshTokenCacheStub) AddToUserTokenSet(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
+	return nil
+}
+
+func (s *refreshTokenCacheStub) AddToFamilyTokenSet(ctx context.Context, familyID string, tokenHash string, ttl time.Duration) error {
+	return nil
+}
+
+func (s *refreshTokenCacheStub) GetUserTokenHashes(ctx context.Context, userID int64) ([]string, error) {
+	return nil, nil
+}
+
+func (s *refreshTokenCacheStub) GetFamilyTokenHashes(ctx context.Context, familyID string) ([]string, error) {
+	return nil, nil
+}
+
+func (s *refreshTokenCacheStub) IsTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
+	return false, nil
+}
+
 func newAuthService(repo *userRepoStub, settings map[string]string, emailCache EmailCache) *AuthService {
 	cfg := &config.Config{
 		JWT: config.JWTConfig{
-			Secret:     "test-secret",
-			ExpireHour: 1,
+			Secret:                 "test-secret",
+			ExpireHour:             1,
+			RefreshTokenExpireDays: 30,
 		},
 		Default: config.DefaultConfig{
 			UserBalance:     3.5,
@@ -464,3 +507,113 @@ func TestAuthService_Register_AssignsDefaultSubscriptions(t *testing.T) {
 	require.Equal(t, int64(12), assigner.calls[1].GroupID)
 	require.Equal(t, 7, assigner.calls[1].ValidityDays)
 }
+
+func TestAuthService_LoginOrRegisterOAuth_AssignsLinuxDoGiftSubscriptionsOnFirstSignup(t *testing.T) {
+	repo := &userRepoStub{nextID: 21}
+	assigner := &defaultSubscriptionAssignerStub{}
+	service := newAuthService(repo, map[string]string{
+		SettingKeyRegistrationEnabled:    "true",
+		SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":31,"validity_days":14},{"group_id":32,"validity_days":30}]`,
+		SettingKeyDefaultSubscriptions:   `[]`,
+		SettingKeyInvitationCodeEnabled:  "false",
+	}, nil)
+	service.defaultSubAssigner = assigner
+
+	token, user, err := service.LoginOrRegisterOAuth(context.Background(), "linuxdo-new@test.com", "linuxdo-user")
+	require.NoError(t, err)
+	require.NotEmpty(t, token)
+	require.NotNil(t, user)
+	require.Equal(t, int64(21), user.ID)
+	require.Len(t, assigner.calls, 2)
+	require.Equal(t, int64(31), assigner.calls[0].GroupID)
+	require.Equal(t, 14, assigner.calls[0].ValidityDays)
+	require.Equal(t, "auto assigned by linuxdo connect gift subscriptions setting", assigner.calls[0].Notes)
+	require.Equal(t, int64(32), assigner.calls[1].GroupID)
+	require.Equal(t, 30, assigner.calls[1].ValidityDays)
+}
+
+func TestAuthService_LoginOrRegisterOAuth_DoesNotAssignLinuxDoGiftSubscriptionsWhenEmpty(t *testing.T) {
+	repo := &userRepoStub{nextID: 22}
+	assigner := &defaultSubscriptionAssignerStub{}
+	service := newAuthService(repo, map[string]string{
+		SettingKeyRegistrationEnabled:    "true",
+		SettingKeyLinuxDoConnectGiftSubs: `[]`,
+	}, nil)
+	service.defaultSubAssigner = assigner
+
+	token, user, err := service.LoginOrRegisterOAuth(context.Background(), "linuxdo-empty@test.com", "linuxdo-empty")
+	require.NoError(t, err)
+	require.NotEmpty(t, token)
+	require.NotNil(t, user)
+	require.Empty(t, assigner.calls)
+}
+
+func TestAuthService_LoginOrRegisterOAuth_DoesNotReassignLinuxDoGiftSubscriptionsForExistingUser(t *testing.T) {
+	existingUser := &User{
+		ID:           23,
+		Email:        "linuxdo-existing@test.com",
+		Username:     "existing-user",
+		Role:         RoleUser,
+		Status:       StatusActive,
+		TokenVersion: 1,
+	}
+	repo := &userRepoStub{
+		userByEmail: map[string]*User{
+			existingUser.Email: existingUser,
+		},
+	}
+	assigner := &defaultSubscriptionAssignerStub{}
+	service := newAuthService(repo, map[string]string{
+		SettingKeyRegistrationEnabled:    "true",
+		SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":41,"validity_days":15}]`,
+	}, nil)
+	service.defaultSubAssigner = assigner
+
+	token, user, err := service.LoginOrRegisterOAuth(context.Background(), existingUser.Email, existingUser.Username)
+	require.NoError(t, err)
+	require.NotEmpty(t, token)
+	require.Same(t, existingUser, user)
+	require.Empty(t, repo.created)
+	require.Empty(t, assigner.calls)
+}
+
+func TestAuthService_LoginOrRegisterOAuthWithTokenPair_AssignsLinuxDoGiftSubscriptionsForInvitationSignup(t *testing.T) {
+	repo := &userRepoStub{nextID: 24}
+	assigner := &defaultSubscriptionAssignerStub{}
+	redeemRepo := &redeemRepoStub{
+		codesByCode: map[string]*RedeemCode{
+			"invite-123": {
+				ID:     9,
+				Code:   "invite-123",
+				Type:   RedeemTypeInvitation,
+				Status: StatusUnused,
+			},
+		},
+	}
+	service := newAuthService(repo, map[string]string{
+		SettingKeyRegistrationEnabled:    "true",
+		SettingKeyInvitationCodeEnabled:  "true",
+		SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":51,"validity_days":45}]`,
+	}, nil)
+	service.defaultSubAssigner = assigner
+	service.refreshTokenCache = &refreshTokenCacheStub{}
+	service.redeemRepo = redeemRepo
+
+	tokenPair, user, err := service.LoginOrRegisterOAuthWithTokenPair(
+		context.Background(),
+		"linuxdo-invite@test.com",
+		"invite-user",
+		"invite-123",
+	)
+	require.NoError(t, err)
+	require.NotNil(t, tokenPair)
+	require.NotEmpty(t, tokenPair.AccessToken)
+	require.NotEmpty(t, tokenPair.RefreshToken)
+	require.NotNil(t, user)
+	require.Equal(t, int64(24), user.ID)
+	require.Len(t, assigner.calls, 1)
+	require.Equal(t, int64(51), assigner.calls[0].GroupID)
+	require.Equal(t, 45, assigner.calls[0].ValidityDays)
+	require.Equal(t, []int64{9}, redeemRepo.usedCodeIDs)
+	require.Equal(t, []int64{24}, redeemRepo.usedUserIDs)
+}
diff --git a/backend/internal/service/domain_constants.go b/backend/internal/service/domain_constants.go
index 52df52d6..6e63b40d 100644
--- a/backend/internal/service/domain_constants.go
+++ b/backend/internal/service/domain_constants.go
@@ -104,6 +104,7 @@ const (
 	SettingKeyLinuxDoConnectClientID     = "linuxdo_connect_client_id"
 	SettingKeyLinuxDoConnectClientSecret = "linuxdo_connect_client_secret"
 	SettingKeyLinuxDoConnectRedirectURL  = "linuxdo_connect_redirect_url"
+	SettingKeyLinuxDoConnectGiftSubs     = "linuxdo_connect_gift_subscriptions"
 
 	// OEM设置
 	SettingKeySiteName                    = "site_name"                     // 网站名称
diff --git a/backend/internal/service/setting_service.go b/backend/internal/service/setting_service.go
index a85efabd..6d82d7e6 100644
--- a/backend/internal/service/setting_service.go
+++ b/backend/internal/service/setting_service.go
@@ -405,7 +405,10 @@ func parseCustomMenuItemURLs(raw string) []string {
 
 // UpdateSettings 更新系统设置
 func (s *SettingService) UpdateSettings(ctx context.Context, settings *SystemSettings) error {
-	if err := s.validateDefaultSubscriptionGroups(ctx, settings.DefaultSubscriptions); err != nil {
+	if err := s.validateSubscriptionGroups(ctx, settings.DefaultSubscriptions); err != nil {
+		return err
+	}
+	if err := s.validateSubscriptionGroups(ctx, settings.LinuxDoConnectGiftSubscriptions); err != nil {
 		return err
 	}
 	normalizedWhitelist, err := NormalizeRegistrationEmailSuffixWhitelist(settings.RegistrationEmailSuffixWhitelist)
@@ -458,6 +461,11 @@ func (s *SettingService) UpdateSettings(ctx context.Context, settings *SystemSet
 	if settings.LinuxDoConnectClientSecret != "" {
 		updates[SettingKeyLinuxDoConnectClientSecret] = settings.LinuxDoConnectClientSecret
 	}
+	linuxDoGiftSubsJSON, err := json.Marshal(settings.LinuxDoConnectGiftSubscriptions)
+	if err != nil {
+		return fmt.Errorf("marshal linuxdo connect gift subscriptions: %w", err)
+	}
+	updates[SettingKeyLinuxDoConnectGiftSubs] = string(linuxDoGiftSubsJSON)
 
 	// OEM设置
 	updates[SettingKeySiteName] = settings.SiteName
@@ -542,7 +550,7 @@ func (s *SettingService) UpdateSettings(ctx context.Context, settings *SystemSet
 	return err
 }
 
-func (s *SettingService) validateDefaultSubscriptionGroups(ctx context.Context, items []DefaultSubscriptionSetting) error {
+func (s *SettingService) validateSubscriptionGroups(ctx context.Context, items []DefaultSubscriptionSetting) error {
 	if len(items) == 0 {
 		return nil
 	}
@@ -581,6 +589,10 @@ func (s *SettingService) validateDefaultSubscriptionGroups(ctx context.Context,
 	return nil
 }
 
+func (s *SettingService) validateDefaultSubscriptionGroups(ctx context.Context, items []DefaultSubscriptionSetting) error {
+	return s.validateSubscriptionGroups(ctx, items)
+}
+
 // IsRegistrationEnabled 检查是否开放注册
 func (s *SettingService) IsRegistrationEnabled(ctx context.Context) bool {
 	value, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationEnabled)
@@ -792,7 +804,16 @@ func (s *SettingService) GetDefaultSubscriptions(ctx context.Context) []DefaultS
 	if err != nil {
 		return nil
 	}
-	return parseDefaultSubscriptions(value)
+	return parseSubscriptionSettings(value)
+}
+
+// GetLinuxDoConnectGiftSubscriptions 获取 LinuxDo Connect 首次登录赠送订阅配置列表。
+func (s *SettingService) GetLinuxDoConnectGiftSubscriptions(ctx context.Context) []DefaultSubscriptionSetting {
+	value, err := s.settingRepo.GetValue(ctx, SettingKeyLinuxDoConnectGiftSubs)
+	if err != nil {
+		return nil
+	}
+	return parseSubscriptionSettings(value)
 }
 
 // InitializeDefaultSettings 初始化默认设置
@@ -822,6 +843,7 @@ func (s *SettingService) InitializeDefaultSettings(ctx context.Context) error {
 		SettingKeyDefaultConcurrency:               strconv.Itoa(s.cfg.Default.UserConcurrency),
 		SettingKeyDefaultBalance:                   strconv.FormatFloat(s.cfg.Default.UserBalance, 'f', 8, 64),
 		SettingKeyDefaultSubscriptions:             "[]",
+		SettingKeyLinuxDoConnectGiftSubs:           "[]",
 		SettingKeySMTPPort:                         "587",
 		SettingKeySMTPUseTLS:                       "false",
 		// Model fallback defaults
@@ -906,7 +928,8 @@ func (s *SettingService) parseSettings(settings map[string]string) *SystemSettin
 	} else {
 		result.DefaultBalance = s.cfg.Default.UserBalance
 	}
-	result.DefaultSubscriptions = parseDefaultSubscriptions(settings[SettingKeyDefaultSubscriptions])
+	result.DefaultSubscriptions = parseSubscriptionSettings(settings[SettingKeyDefaultSubscriptions])
+	result.LinuxDoConnectGiftSubscriptions = parseSubscriptionSettings(settings[SettingKeyLinuxDoConnectGiftSubs])
 
 	// 敏感信息直接返回，方便测试连接时使用
 	result.SMTPPassword = settings[SettingKeySMTPPassword]
@@ -1003,7 +1026,7 @@ func isFalseSettingValue(value string) bool {
 	}
 }
 
-func parseDefaultSubscriptions(raw string) []DefaultSubscriptionSetting {
+func parseSubscriptionSettings(raw string) []DefaultSubscriptionSetting {
 	raw = strings.TrimSpace(raw)
 	if raw == "" {
 		return nil
@@ -1028,6 +1051,10 @@ func parseDefaultSubscriptions(raw string) []DefaultSubscriptionSetting {
 	return normalized
 }
 
+func parseDefaultSubscriptions(raw string) []DefaultSubscriptionSetting {
+	return parseSubscriptionSettings(raw)
+}
+
 // getStringOrDefault 获取字符串值或默认值
 func (s *SettingService) getStringOrDefault(settings map[string]string, key, defaultValue string) string {
 	if value, ok := settings[key]; ok && value != "" {
diff --git a/backend/internal/service/setting_service_update_test.go b/backend/internal/service/setting_service_update_test.go
index 1de08611..392171ec 100644
--- a/backend/internal/service/setting_service_update_test.go
+++ b/backend/internal/service/setting_service_update_test.go
@@ -13,6 +13,7 @@ import (
 )
 
 type settingUpdateRepoStub struct {
+	values  map[string]string
 	updates map[string]string
 }
 
@@ -21,7 +22,12 @@ func (s *settingUpdateRepoStub) Get(ctx context.Context, key string) (*Setting,
 }
 
 func (s *settingUpdateRepoStub) GetValue(ctx context.Context, key string) (string, error) {
-	panic("unexpected GetValue call")
+	if s.values != nil {
+		if v, ok := s.values[key]; ok {
+			return v, nil
+		}
+	}
+	return "", ErrSettingNotFound
 }
 
 func (s *settingUpdateRepoStub) Set(ctx context.Context, key, value string) error {
@@ -34,8 +40,10 @@ func (s *settingUpdateRepoStub) GetMultiple(ctx context.Context, keys []string)
 
 func (s *settingUpdateRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
 	s.updates = make(map[string]string, len(settings))
+	s.values = make(map[string]string, len(settings))
 	for k, v := range settings {
 		s.updates[k] = v
+		s.values[k] = v
 	}
 	return nil
 }
@@ -172,6 +180,54 @@ func TestSettingService_UpdateSettings_DefaultSubscriptions_RejectsDuplicateGrou
 	require.Nil(t, repo.updates)
 }
 
+func TestSettingService_UpdateSettings_LinuxDoConnectGiftSubscriptions_ValidGroup(t *testing.T) {
+	repo := &settingUpdateRepoStub{}
+	groupReader := &defaultSubGroupReaderStub{
+		byID: map[int64]*Group{
+			21: {ID: 21, SubscriptionType: SubscriptionTypeSubscription},
+		},
+	}
+	svc := NewSettingService(repo, &config.Config{})
+	svc.SetDefaultSubscriptionGroupReader(groupReader)
+
+	err := svc.UpdateSettings(context.Background(), &SystemSettings{
+		LinuxDoConnectGiftSubscriptions: []DefaultSubscriptionSetting{
+			{GroupID: 21, ValidityDays: 14},
+		},
+	})
+	require.NoError(t, err)
+
+	raw, ok := repo.updates[SettingKeyLinuxDoConnectGiftSubs]
+	require.True(t, ok)
+
+	var got []DefaultSubscriptionSetting
+	require.NoError(t, json.Unmarshal([]byte(raw), &got))
+	require.Equal(t, []DefaultSubscriptionSetting{
+		{GroupID: 21, ValidityDays: 14},
+	}, got)
+}
+
+func TestSettingService_UpdateSettings_LinuxDoConnectGiftSubscriptions_RejectsDuplicateGroup(t *testing.T) {
+	repo := &settingUpdateRepoStub{}
+	groupReader := &defaultSubGroupReaderStub{
+		byID: map[int64]*Group{
+			21: {ID: 21, SubscriptionType: SubscriptionTypeSubscription},
+		},
+	}
+	svc := NewSettingService(repo, &config.Config{})
+	svc.SetDefaultSubscriptionGroupReader(groupReader)
+
+	err := svc.UpdateSettings(context.Background(), &SystemSettings{
+		LinuxDoConnectGiftSubscriptions: []DefaultSubscriptionSetting{
+			{GroupID: 21, ValidityDays: 14},
+			{GroupID: 21, ValidityDays: 21},
+		},
+	})
+	require.Error(t, err)
+	require.Equal(t, "DEFAULT_SUBSCRIPTION_GROUP_DUPLICATE", infraerrors.Reason(err))
+	require.Nil(t, repo.updates)
+}
+
 func TestSettingService_UpdateSettings_RegistrationEmailSuffixWhitelist_Normalized(t *testing.T) {
 	repo := &settingUpdateRepoStub{}
 	svc := NewSettingService(repo, &config.Config{})
@@ -202,3 +258,31 @@ func TestParseDefaultSubscriptions_NormalizesValues(t *testing.T) {
 		{GroupID: 12, ValidityDays: MaxValidityDays},
 	}, got)
 }
+
+func TestSettingService_GetLinuxDoConnectGiftSubscriptions_NormalizesValues(t *testing.T) {
+	svc := NewSettingService(&settingUpdateRepoStub{
+		values: map[string]string{
+			SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":21,"validity_days":14},{"group_id":0,"validity_days":3},{"group_id":22,"validity_days":99999}]`,
+		},
+	}, &config.Config{})
+
+	got := svc.GetLinuxDoConnectGiftSubscriptions(context.Background())
+	require.Equal(t, []DefaultSubscriptionSetting{
+		{GroupID: 21, ValidityDays: 14},
+		{GroupID: 22, ValidityDays: MaxValidityDays},
+	}, got)
+}
+
+func TestSettingService_InitializeDefaultSettings_SetsLinuxDoGiftSubscriptionsEmptyArray(t *testing.T) {
+	repo := &settingUpdateRepoStub{}
+	svc := NewSettingService(repo, &config.Config{
+		Default: config.DefaultConfig{
+			UserBalance:     1.5,
+			UserConcurrency: 2,
+		},
+	})
+
+	err := svc.InitializeDefaultSettings(context.Background())
+	require.NoError(t, err)
+	require.Equal(t, "[]", repo.updates[SettingKeyLinuxDoConnectGiftSubs])
+}
diff --git a/backend/internal/service/settings_view.go b/backend/internal/service/settings_view.go
index 4b64267f..b609cc3e 100644
--- a/backend/internal/service/settings_view.go
+++ b/backend/internal/service/settings_view.go
@@ -30,6 +30,7 @@ type SystemSettings struct {
 	LinuxDoConnectClientSecret           string
 	LinuxDoConnectClientSecretConfigured bool
 	LinuxDoConnectRedirectURL            string
+	LinuxDoConnectGiftSubscriptions      []DefaultSubscriptionSetting
 
 	SiteName                    string
 	SiteLogo                    string
```

### Frontend

```diff
diff --git a/frontend/src/api/admin/settings.ts b/frontend/src/api/admin/settings.ts
index 8f9284b7..a1910de8 100644
--- a/frontend/src/api/admin/settings.ts
+++ b/frontend/src/api/admin/settings.ts
@@ -29,6 +29,7 @@ export interface SystemSettings {
   default_balance: number
   default_concurrency: number
   default_subscriptions: DefaultSubscriptionSetting[]
+  linuxdo_connect_gift_subscriptions: DefaultSubscriptionSetting[]
   // OEM settings
   site_name: string
   site_logo: string
@@ -103,6 +104,7 @@ export interface UpdateSettingsRequest {
   default_balance?: number
   default_concurrency?: number
   default_subscriptions?: DefaultSubscriptionSetting[]
+  linuxdo_connect_gift_subscriptions?: DefaultSubscriptionSetting[]
   site_name?: string
   site_logo?: string
   site_subtitle?: string
diff --git a/frontend/src/i18n/locales/en.ts b/frontend/src/i18n/locales/en.ts
index 30c87d92..a31e08de 100644
--- a/frontend/src/i18n/locales/en.ts
+++ b/frontend/src/i18n/locales/en.ts
@@ -4225,7 +4225,15 @@ export default {
         redirectUrlHint:
           'Must match the redirect URL configured in Connect.Linux.Do (must be an absolute http(s) URL)',
         quickSetCopy: 'Generate & Copy (current site)',
-        redirectUrlSetAndCopied: 'Redirect URL generated and copied to clipboard'
+        redirectUrlSetAndCopied: 'Redirect URL generated and copied to clipboard',
+        giftSubscriptions: 'Gift Subscriptions',
+        giftSubscriptionsHint:
+          'Granted only when a user creates a local account through LinuxDo Connect for the first time. Leave empty to disable gifting.',
+        addGiftSubscription: 'Add Gift Subscription',
+        giftSubscriptionsEmpty:
+          'No LinuxDo Connect gift subscriptions configured. First-time LinuxDo sign-ins will not receive extra subscriptions.',
+        giftSubscriptionsDuplicate:
+          'Duplicate LinuxDo Connect gift subscription group: {groupId}. Each group can only appear once.'
       },
       defaults: {
         title: 'Default User Settings',
diff --git a/frontend/src/i18n/locales/zh.ts b/frontend/src/i18n/locales/zh.ts
index d7d920ae..a840bee4 100644
--- a/frontend/src/i18n/locales/zh.ts
+++ b/frontend/src/i18n/locales/zh.ts
@@ -4391,7 +4391,12 @@ export default {
         redirectUrlPlaceholder: 'https://your-domain.com/api/v1/auth/oauth/linuxdo/callback',
         redirectUrlHint: '需与 Connect.Linux.Do 中配置的回调地址一致（必须是 http(s) 完整 URL）',
         quickSetCopy: '使用当前站点生成并复制',
-        redirectUrlSetAndCopied: '已使用当前站点生成回调地址并复制到剪贴板'
+        redirectUrlSetAndCopied: '已使用当前站点生成回调地址并复制到剪贴板',
+        giftSubscriptions: '赠送订阅',
+        giftSubscriptionsHint: '仅在用户首次通过 LinuxDo Connect 创建本地账号后发放，可留空表示不赠送',
+        addGiftSubscription: '添加赠送订阅',
+        giftSubscriptionsEmpty: '未配置 LinuxDo Connect 赠送订阅。首次通过 LinuxDo 登录的用户不会自动获得额外订阅。',
+        giftSubscriptionsDuplicate: 'LinuxDo Connect 赠送订阅存在重复分组：{groupId}。每个分组只能出现一次。'
       },
       defaults: {
         title: '用户默认设置',
diff --git a/frontend/src/views/admin/SettingsView.vue b/frontend/src/views/admin/SettingsView.vue
index 1839d03c..822b3d85 100644
--- a/frontend/src/views/admin/SettingsView.vue
+++ b/frontend/src/views/admin/SettingsView.vue
@@ -1020,6 +1020,98 @@
                 </div>
               </div>
             </div>
+
+            <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
+              <div class="mb-3 flex items-center justify-between">
+                <div>
+                  <label class="font-medium text-gray-900 dark:text-white">
+                    {{ t('admin.settings.linuxdo.giftSubscriptions') }}
+                  </label>
+                  <p class="text-sm text-gray-500 dark:text-gray-400">
+                    {{ t('admin.settings.linuxdo.giftSubscriptionsHint') }}
+                  </p>
+                </div>
+                <button
+                  type="button"
+                  class="btn btn-secondary btn-sm"
+                  @click="addLinuxDoConnectGiftSubscription"
+                  :disabled="subscriptionGroups.length === 0"
+                >
+                  {{ t('admin.settings.linuxdo.addGiftSubscription') }}
+                </button>
+              </div>
+
+              <div
+                v-if="form.linuxdo_connect_gift_subscriptions.length === 0"
+                class="rounded border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
+              >
+                {{ t('admin.settings.linuxdo.giftSubscriptionsEmpty') }}
+              </div>
+
+              <div v-else class="space-y-3">
+                <div
+                  v-for="(item, index) in form.linuxdo_connect_gift_subscriptions"
+                  :key="`linuxdo-gift-sub-${index}`"
+                  class="grid grid-cols-1 gap-3 rounded border border-gray-200 p-3 md:grid-cols-[1fr_160px_auto] dark:border-dark-600"
+                >
+                  <div>
+                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
+                      {{ t('admin.settings.defaults.subscriptionGroup') }}
+                    </label>
+                    <Select
+                      v-model="item.group_id"
+                      class="default-sub-group-select"
+                      :options="defaultSubscriptionGroupOptions"
+                      :placeholder="t('admin.settings.defaults.subscriptionGroup')"
+                    >
+                      <template #selected="{ option }">
+                        <GroupBadge
+                          v-if="option"
+                          :name="(option as unknown as DefaultSubscriptionGroupOption).label"
+                          :platform="(option as unknown as DefaultSubscriptionGroupOption).platform"
+                          :subscription-type="(option as unknown as DefaultSubscriptionGroupOption).subscriptionType"
+                          :rate-multiplier="(option as unknown as DefaultSubscriptionGroupOption).rate"
+                        />
+                        <span v-else class="text-gray-400">
+                          {{ t('admin.settings.defaults.subscriptionGroup') }}
+                        </span>
+                      </template>
+                      <template #option="{ option, selected }">
+                        <GroupOptionItem
+                          :name="(option as unknown as DefaultSubscriptionGroupOption).label"
+                          :platform="(option as unknown as DefaultSubscriptionGroupOption).platform"
+                          :subscription-type="(option as unknown as DefaultSubscriptionGroupOption).subscriptionType"
+                          :rate-multiplier="(option as unknown as DefaultSubscriptionGroupOption).rate"
+                          :description="(option as unknown as DefaultSubscriptionGroupOption).description"
+                          :selected="selected"
+                        />
+                      </template>
+                    </Select>
+                  </div>
+                  <div>
+                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
+                      {{ t('admin.settings.defaults.subscriptionValidityDays') }}
+                    </label>
+                    <input
+                      v-model.number="item.validity_days"
+                      type="number"
+                      min="1"
+                      max="36500"
+                      class="input h-[42px]"
+                    />
+                  </div>
+                  <div class="flex items-end">
+                    <button
+                      type="button"
+                      class="btn btn-secondary default-sub-delete-btn w-full text-red-600 hover:text-red-700 dark:text-red-400"
+                      @click="removeLinuxDoConnectGiftSubscription(index)"
+                    >
+                      {{ t('common.delete') }}
+                    </button>
+                  </div>
+                </div>
+              </div>
+            </div>
           </div>
         </div>
         </div><!-- /Tab: Security — Registration, Turnstile, LinuxDo -->
@@ -2122,6 +2214,7 @@ const form = reactive<SettingsForm>({
   linuxdo_connect_client_secret: '',
   linuxdo_connect_client_secret_configured: false,
   linuxdo_connect_redirect_url: '',
+  linuxdo_connect_gift_subscriptions: [],
   // Model fallback
   enable_model_fallback: false,
   fallback_model_anthropic: 'claude-3-5-sonnet-20241022',
@@ -2288,16 +2381,7 @@ async function loadSettings() {
   loadFailed.value = false
   try {
     const settings = await adminAPI.settings.getSettings()
-    Object.assign(form, settings)
-    form.backend_mode_enabled = settings.backend_mode_enabled
-    form.default_subscriptions = Array.isArray(settings.default_subscriptions)
-      ? settings.default_subscriptions
-          .filter((item) => item.group_id > 0 && item.validity_days > 0)
-          .map((item) => ({
-            group_id: item.group_id,
-            validity_days: item.validity_days
-          }))
-      : []
+    applySettingsToForm(settings)
     registrationEmailSuffixWhitelistTags.value = normalizeRegistrationEmailSuffixDomains(
       settings.registration_email_suffix_whitelist
     )
@@ -2316,6 +2400,53 @@ async function loadSettings() {
   }
 }
 
+function normalizeSubscriptionSettings(
+  items: DefaultSubscriptionSetting[] | null | undefined
+): DefaultSubscriptionSetting[] {
+  if (!Array.isArray(items)) return []
+  return items
+    .filter((item) => item.group_id > 0 && item.validity_days > 0)
+    .map((item) => ({
+      group_id: item.group_id,
+      validity_days: Math.min(36500, Math.max(1, Math.floor(item.validity_days)))
+    }))
+}
+
+function findDuplicateSubscription(items: DefaultSubscriptionSetting[]): DefaultSubscriptionSetting | undefined {
+  const seenGroupIDs = new Set<number>()
+  return items.find((item) => {
+    if (seenGroupIDs.has(item.group_id)) {
+      return true
+    }
+    seenGroupIDs.add(item.group_id)
+    return false
+  })
+}
+
+function addSubscriptionItem(target: DefaultSubscriptionSetting[]) {
+  if (subscriptionGroups.value.length === 0) return
+  const existing = new Set(target.map((item) => item.group_id))
+  const candidate = subscriptionGroups.value.find((group) => !existing.has(group.id))
+  if (!candidate) return
+  target.push({
+    group_id: candidate.id,
+    validity_days: 30
+  })
+}
+
+function removeSubscriptionItem(target: DefaultSubscriptionSetting[], index: number) {
+  target.splice(index, 1)
+}
+
+function applySettingsToForm(settings: SystemSettings) {
+  Object.assign(form, settings)
+  form.backend_mode_enabled = settings.backend_mode_enabled
+  form.default_subscriptions = normalizeSubscriptionSettings(settings.default_subscriptions)
+  form.linuxdo_connect_gift_subscriptions = normalizeSubscriptionSettings(
+    settings.linuxdo_connect_gift_subscriptions
+  )
+}
+
 async function loadSubscriptionGroups() {
   try {
     const groups = await adminAPI.groups.getAll()
@@ -2329,38 +2460,30 @@ async function loadSubscriptionGroups() {
 }
 
 function addDefaultSubscription() {
-  if (subscriptionGroups.value.length === 0) return
-  const existing = new Set(form.default_subscriptions.map((item) => item.group_id))
-  const candidate = subscriptionGroups.value.find((group) => !existing.has(group.id))
-  if (!candidate) return
-  form.default_subscriptions.push({
-    group_id: candidate.id,
-    validity_days: 30
-  })
+  addSubscriptionItem(form.default_subscriptions)
 }
 
 function removeDefaultSubscription(index: number) {
-  form.default_subscriptions.splice(index, 1)
+  removeSubscriptionItem(form.default_subscriptions, index)
+}
+
+function addLinuxDoConnectGiftSubscription() {
+  addSubscriptionItem(form.linuxdo_connect_gift_subscriptions)
+}
+
+function removeLinuxDoConnectGiftSubscription(index: number) {
+  removeSubscriptionItem(form.linuxdo_connect_gift_subscriptions, index)
 }
 
 async function saveSettings() {
   saving.value = true
   try {
-    const normalizedDefaultSubscriptions = form.default_subscriptions
-      .filter((item) => item.group_id > 0 && item.validity_days > 0)
-      .map((item: DefaultSubscriptionSetting) => ({
-        group_id: item.group_id,
-        validity_days: Math.min(36500, Math.max(1, Math.floor(item.validity_days)))
-      }))
-
-    const seenGroupIDs = new Set<number>()
-    const duplicateDefaultSubscription = normalizedDefaultSubscriptions.find((item) => {
-      if (seenGroupIDs.has(item.group_id)) {
-        return true
-      }
-      seenGroupIDs.add(item.group_id)
-      return false
-    })
+    const normalizedDefaultSubscriptions = normalizeSubscriptionSettings(form.default_subscriptions)
+    const normalizedLinuxDoConnectGiftSubscriptions = normalizeSubscriptionSettings(
+      form.linuxdo_connect_gift_subscriptions
+    )
+
+    const duplicateDefaultSubscription = findDuplicateSubscription(normalizedDefaultSubscriptions)
     if (duplicateDefaultSubscription) {
       appStore.showError(
         t('admin.settings.defaults.defaultSubscriptionsDuplicate', {
@@ -2370,6 +2493,18 @@ async function saveSettings() {
       return
     }
 
+    const duplicateLinuxDoGiftSubscription = findDuplicateSubscription(
+      normalizedLinuxDoConnectGiftSubscriptions
+    )
+    if (duplicateLinuxDoGiftSubscription) {
+      appStore.showError(
+        t('admin.settings.linuxdo.giftSubscriptionsDuplicate', {
+          groupId: duplicateLinuxDoGiftSubscription.group_id
+        })
+      )
+      return
+    }
+
     // Validate URL fields — novalidate disables browser-native checks, so we validate here
     const isValidHttpUrl = (url: string): boolean => {
       if (!url) return true
@@ -2412,6 +2547,7 @@ async function saveSettings() {
       default_balance: form.default_balance,
       default_concurrency: form.default_concurrency,
       default_subscriptions: normalizedDefaultSubscriptions,
+      linuxdo_connect_gift_subscriptions: normalizedLinuxDoConnectGiftSubscriptions,
       site_name: form.site_name,
       site_logo: form.site_logo,
       site_subtitle: form.site_subtitle,
@@ -2454,7 +2590,7 @@ async function saveSettings() {
       enable_metadata_passthrough: form.enable_metadata_passthrough
     }
     const updated = await adminAPI.settings.updateSettings(payload)
-    Object.assign(form, updated)
+    applySettingsToForm(updated)
     registrationEmailSuffixWhitelistTags.value = normalizeRegistrationEmailSuffixDomains(
       updated.registration_email_suffix_whitelist
     )
```
