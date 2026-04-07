//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type settingRepoStub struct {
	values map[string]string
	err    error
}

func (s *settingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type emailCacheStub struct {
	data *VerificationCodeData
	err  error
}

type defaultSubscriptionAssignerStub struct {
	calls []AssignSubscriptionInput
	err   error
}

type refreshTokenCacheStub struct{}

type linuxDoAutoCheckinRewardRepoStub struct {
	records map[string]CreateLinuxDoAutoCheckinRewardInput
	calls   []CreateLinuxDoAutoCheckinRewardInput
	err     error
}

func (s *linuxDoAutoCheckinRewardRepoStub) Create(_ context.Context, input CreateLinuxDoAutoCheckinRewardInput) error {
	s.calls = append(s.calls, input)
	if s.err != nil {
		return s.err
	}
	if s.records == nil {
		s.records = make(map[string]CreateLinuxDoAutoCheckinRewardInput)
	}
	key := fmt.Sprintf("%d|%s|%s", input.UserID, input.RewardDate.Format(time.DateOnly), input.Source)
	if _, exists := s.records[key]; exists {
		return ErrLinuxDoAutoCheckinRewardAlreadyGranted
	}
	s.records[key] = input
	return nil
}

func (s *defaultSubscriptionAssignerStub) AssignOrExtendSubscription(_ context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	if input != nil {
		s.calls = append(s.calls, *input)
	}
	if s.err != nil {
		return nil, false, s.err
	}
	return &UserSubscription{UserID: input.UserID, GroupID: input.GroupID}, false, nil
}

func (s *emailCacheStub) GetVerificationCode(ctx context.Context, email string) (*VerificationCodeData, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s *emailCacheStub) SetVerificationCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error {
	return nil
}

func (s *emailCacheStub) DeleteVerificationCode(ctx context.Context, email string) error {
	return nil
}

func (s *emailCacheStub) GetPasswordResetToken(ctx context.Context, email string) (*PasswordResetTokenData, error) {
	return nil, nil
}

func (s *emailCacheStub) SetPasswordResetToken(ctx context.Context, email string, data *PasswordResetTokenData, ttl time.Duration) error {
	return nil
}

func (s *emailCacheStub) DeletePasswordResetToken(ctx context.Context, email string) error {
	return nil
}

func (s *emailCacheStub) IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool {
	return false
}

func (s *emailCacheStub) SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) StoreRefreshToken(ctx context.Context, tokenHash string, data *RefreshTokenData, ttl time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenData, error) {
	return nil, ErrRefreshTokenNotFound
}

func (s *refreshTokenCacheStub) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}

func (s *refreshTokenCacheStub) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	return nil
}

func (s *refreshTokenCacheStub) DeleteTokenFamily(ctx context.Context, familyID string) error {
	return nil
}

func (s *refreshTokenCacheStub) AddToUserTokenSet(ctx context.Context, userID int64, tokenHash string, ttl time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) AddToFamilyTokenSet(ctx context.Context, familyID string, tokenHash string, ttl time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) GetUserTokenHashes(ctx context.Context, userID int64) ([]string, error) {
	return nil, nil
}

func (s *refreshTokenCacheStub) GetFamilyTokenHashes(ctx context.Context, familyID string) ([]string, error) {
	return nil, nil
}

func (s *refreshTokenCacheStub) IsTokenInFamily(ctx context.Context, familyID string, tokenHash string) (bool, error) {
	return false, nil
}

func newAuthService(repo *userRepoStub, settings map[string]string, emailCache EmailCache) *AuthService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                 "test-secret",
			ExpireHour:             1,
			RefreshTokenExpireDays: 30,
		},
		Default: config.DefaultConfig{
			UserBalance:     3.5,
			UserConcurrency: 2,
		},
	}

	var settingService *SettingService
	if settings != nil {
		settingService = NewSettingService(&settingRepoStub{values: settings}, cfg)
	}

	var emailService *EmailService
	if emailCache != nil {
		emailService = NewEmailService(&settingRepoStub{values: settings}, emailCache)
	}

	return NewAuthService(
		nil, // entClient
		repo,
		nil, // redeemRepo
		nil, // refreshTokenCache
		cfg,
		settingService,
		emailService,
		nil,
		nil,
		nil, // promoService
		nil, // defaultSubAssigner
	)
}

func TestAuthService_Register_Disabled(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "false",
	}, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrRegDisabled)
}

func TestAuthService_Register_DisabledByDefault(t *testing.T) {
	// 当 settings 为 nil（设置项不存在）时，注册应该默认关闭
	repo := &userRepoStub{}
	service := newAuthService(repo, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrRegDisabled)
}

func TestAuthService_Register_EmailVerifyEnabledButServiceNotConfigured(t *testing.T) {
	repo := &userRepoStub{}
	// 邮件验证开启但 emailCache 为 nil（emailService 未配置）
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyEmailVerifyEnabled:  "true",
	}, nil)

	// 应返回服务不可用错误，而不是允许绕过验证
	_, _, err := service.RegisterWithVerification(context.Background(), "user@test.com", "password", "any-code", "", "")
	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestAuthService_Register_EmailVerifyRequired(t *testing.T) {
	repo := &userRepoStub{}
	cache := &emailCacheStub{} // 配置 emailService
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyEmailVerifyEnabled:  "true",
	}, cache)

	_, _, err := service.RegisterWithVerification(context.Background(), "user@test.com", "password", "", "", "")
	require.ErrorIs(t, err, ErrEmailVerifyRequired)
}

func TestAuthService_Register_EmailVerifyInvalid(t *testing.T) {
	repo := &userRepoStub{}
	cache := &emailCacheStub{
		data: &VerificationCodeData{Code: "expected", Attempts: 0},
	}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyEmailVerifyEnabled:  "true",
	}, cache)

	_, _, err := service.RegisterWithVerification(context.Background(), "user@test.com", "password", "wrong", "", "")
	require.ErrorIs(t, err, ErrInvalidVerifyCode)
	require.ErrorContains(t, err, "verify code")
}

func TestAuthService_Register_EmailExists(t *testing.T) {
	repo := &userRepoStub{exists: true}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrEmailExists)
}

func TestAuthService_Register_CheckEmailError(t *testing.T) {
	repo := &userRepoStub{existsErr: errors.New("db down")}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestAuthService_Register_ReservedEmail(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := service.Register(context.Background(), "linuxdo-123@linuxdo-connect.invalid", "password")
	require.ErrorIs(t, err, ErrEmailReserved)
}

func TestAuthService_Register_EmailSuffixNotAllowed(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["@example.com","@company.com"]`,
	}, nil)

	_, _, err := service.Register(context.Background(), "user@other.com", "password")
	require.ErrorIs(t, err, ErrEmailSuffixNotAllowed)
	appErr := infraerrors.FromError(err)
	require.Contains(t, appErr.Message, "@example.com")
	require.Contains(t, appErr.Message, "@company.com")
	require.Equal(t, "EMAIL_SUFFIX_NOT_ALLOWED", appErr.Reason)
	require.Equal(t, "2", appErr.Metadata["allowed_suffix_count"])
	require.Equal(t, "@example.com,@company.com", appErr.Metadata["allowed_suffixes"])
}

func TestAuthService_Register_EmailSuffixAllowed(t *testing.T) {
	repo := &userRepoStub{nextID: 8}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["example.com"]`,
	}, nil)

	_, user, err := service.Register(context.Background(), "user@example.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, int64(8), user.ID)
}

func TestAuthService_SendVerifyCode_EmailSuffixNotAllowed(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["@example.com","@company.com"]`,
	}, nil)

	err := service.SendVerifyCode(context.Background(), "user@other.com")
	require.ErrorIs(t, err, ErrEmailSuffixNotAllowed)
	appErr := infraerrors.FromError(err)
	require.Contains(t, appErr.Message, "@example.com")
	require.Contains(t, appErr.Message, "@company.com")
	require.Equal(t, "2", appErr.Metadata["allowed_suffix_count"])
}

func TestAuthService_Register_CreateError(t *testing.T) {
	repo := &userRepoStub{createErr: errors.New("create failed")}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestAuthService_Register_CreateEmailExistsRace(t *testing.T) {
	// 模拟竞态条件：ExistsByEmail 返回 false，但 Create 时因唯一约束失败
	repo := &userRepoStub{createErr: ErrEmailExists}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrEmailExists)
}

func TestAuthService_Register_Success(t *testing.T) {
	repo := &userRepoStub{nextID: 5}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	token, user, err := service.Register(context.Background(), "user@test.com", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.Equal(t, int64(5), user.ID)
	require.Equal(t, "user@test.com", user.Email)
	require.Equal(t, RoleUser, user.Role)
	require.Equal(t, StatusActive, user.Status)
	require.Equal(t, 3.5, user.Balance)
	require.Equal(t, 2, user.Concurrency)
	require.Len(t, repo.created, 1)
	require.True(t, user.CheckPassword("password"))
}

func TestAuthService_ValidateToken_ExpiredReturnsClaimsWithError(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, nil, nil)

	// 创建用户并生成 token
	user := &User{
		ID:           1,
		Email:        "test@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}
	token, err := service.GenerateToken(user)
	require.NoError(t, err)

	// 验证有效 token
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.Equal(t, int64(1), claims.UserID)

	// 模拟过期 token（通过创建一个过期很久的 token）
	service.cfg.JWT.ExpireHour = -1 // 设置为负数使 token 立即过期
	expiredToken, err := service.GenerateToken(user)
	require.NoError(t, err)
	service.cfg.JWT.ExpireHour = 1 // 恢复

	// 验证过期 token 应返回 claims 和 ErrTokenExpired
	claims, err = service.ValidateToken(expiredToken)
	require.ErrorIs(t, err, ErrTokenExpired)
	require.NotNil(t, claims, "claims should not be nil when token is expired")
	require.Equal(t, int64(1), claims.UserID)
	require.Equal(t, "test@test.com", claims.Email)
}

func TestAuthService_RefreshToken_ExpiredTokenNoPanic(t *testing.T) {
	user := &User{
		ID:           1,
		Email:        "test@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}
	repo := &userRepoStub{user: user}
	service := newAuthService(repo, nil, nil)

	// 创建过期 token
	service.cfg.JWT.ExpireHour = -1
	expiredToken, err := service.GenerateToken(user)
	require.NoError(t, err)
	service.cfg.JWT.ExpireHour = 1

	// RefreshToken 使用过期 token 不应 panic
	require.NotPanics(t, func() {
		newToken, err := service.RefreshToken(context.Background(), expiredToken)
		require.NoError(t, err)
		require.NotEmpty(t, newToken)
	})
}

func TestAuthService_GetAccessTokenExpiresIn_FallbackToExpireHour(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 0

	require.Equal(t, 24*3600, service.GetAccessTokenExpiresIn())
}

func TestAuthService_GetAccessTokenExpiresIn_MinutesHasPriority(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 90

	require.Equal(t, 90*60, service.GetAccessTokenExpiresIn())
}

func TestAuthService_GenerateToken_UsesExpireHourWhenMinutesZero(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 0

	user := &User{
		ID:           1,
		Email:        "test@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}

	token, err := service.GenerateToken(user)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)

	require.WithinDuration(t, claims.IssuedAt.Time.Add(24*time.Hour), claims.ExpiresAt.Time, 2*time.Second)
}

func TestAuthService_GenerateToken_UsesMinutesWhenConfigured(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 90

	user := &User{
		ID:           2,
		Email:        "test2@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}

	token, err := service.GenerateToken(user)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)

	require.WithinDuration(t, claims.IssuedAt.Time.Add(90*time.Minute), claims.ExpiresAt.Time, 2*time.Second)
}

func TestAuthService_Register_AssignsDefaultSubscriptions(t *testing.T) {
	repo := &userRepoStub{nextID: 42}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyDefaultSubscriptions: `[{"group_id":11,"validity_days":30},{"group_id":12,"validity_days":7}]`,
	}, nil)
	service.defaultSubAssigner = assigner

	_, user, err := service.Register(context.Background(), "default-sub@test.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, assigner.calls, 2)
	require.Equal(t, int64(42), assigner.calls[0].UserID)
	require.Equal(t, int64(11), assigner.calls[0].GroupID)
	require.Equal(t, 30, assigner.calls[0].ValidityDays)
	require.Equal(t, int64(12), assigner.calls[1].GroupID)
	require.Equal(t, 7, assigner.calls[1].ValidityDays)
}

func TestAuthService_LoginOrRegisterOAuth_AssignsLinuxDoGiftSubscriptionsOnFirstSignup(t *testing.T) {
	repo := &userRepoStub{nextID: 21}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:    "true",
		SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":31,"validity_days":14},{"group_id":32,"validity_days":30}]`,
		SettingKeyDefaultSubscriptions:   `[]`,
		SettingKeyInvitationCodeEnabled:  "false",
	}, nil)
	service.defaultSubAssigner = assigner

	token, user, _, err := service.LoginOrRegisterOAuth(context.Background(), "linuxdo-new@test.com", "linuxdo-user")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.Equal(t, int64(21), user.ID)
	require.Len(t, assigner.calls, 2)
	require.Equal(t, int64(31), assigner.calls[0].GroupID)
	require.Equal(t, 14, assigner.calls[0].ValidityDays)
	require.Equal(t, "auto assigned by linuxdo connect gift subscriptions setting", assigner.calls[0].Notes)
	require.Equal(t, int64(32), assigner.calls[1].GroupID)
	require.Equal(t, 30, assigner.calls[1].ValidityDays)
}

func TestAuthService_LoginOrRegisterOAuth_DoesNotAssignLinuxDoGiftSubscriptionsWhenEmpty(t *testing.T) {
	repo := &userRepoStub{nextID: 22}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:    "true",
		SettingKeyLinuxDoConnectGiftSubs: `[]`,
	}, nil)
	service.defaultSubAssigner = assigner

	token, user, _, err := service.LoginOrRegisterOAuth(context.Background(), "linuxdo-empty@test.com", "linuxdo-empty")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.Empty(t, assigner.calls)
}

func TestAuthService_LoginOrRegisterOAuth_DoesNotReassignLinuxDoGiftSubscriptionsForExistingUser(t *testing.T) {
	existingUser := &User{
		ID:           23,
		Email:        "linuxdo-existing@test.com",
		Username:     "existing-user",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}
	repo := &userRepoStub{
		userByEmail: map[string]*User{
			existingUser.Email: existingUser,
		},
	}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:    "true",
		SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":41,"validity_days":15}]`,
	}, nil)
	service.defaultSubAssigner = assigner

	token, user, _, err := service.LoginOrRegisterOAuth(context.Background(), existingUser.Email, existingUser.Username)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Same(t, existingUser, user)
	require.Empty(t, repo.created)
	require.Empty(t, assigner.calls)
}

func TestAuthService_LoginOrRegisterOAuthWithTokenPair_AssignsLinuxDoGiftSubscriptionsForInvitationSignup(t *testing.T) {
	repo := &userRepoStub{nextID: 24}
	assigner := &defaultSubscriptionAssignerStub{}
	redeemRepo := &redeemRepoStub{
		codesByCode: map[string]*RedeemCode{
			"invite-123": {
				ID:     9,
				Code:   "invite-123",
				Type:   RedeemTypeInvitation,
				Status: StatusUnused,
			},
		},
	}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:    "true",
		SettingKeyInvitationCodeEnabled:  "true",
		SettingKeyLinuxDoConnectGiftSubs: `[{"group_id":51,"validity_days":45}]`,
	}, nil)
	service.defaultSubAssigner = assigner
	service.refreshTokenCache = &refreshTokenCacheStub{}
	service.redeemRepo = redeemRepo

	tokenPair, user, _, err := service.LoginOrRegisterOAuthWithTokenPair(
		context.Background(),
		"linuxdo-invite@test.com",
		"invite-user",
		"invite-123",
	)
	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.NotEmpty(t, tokenPair.AccessToken)
	require.NotEmpty(t, tokenPair.RefreshToken)
	require.NotNil(t, user)
	require.Equal(t, int64(24), user.ID)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(51), assigner.calls[0].GroupID)
	require.Equal(t, 45, assigner.calls[0].ValidityDays)
	require.Equal(t, []int64{9}, redeemRepo.usedCodeIDs)
	require.Equal(t, []int64{24}, redeemRepo.usedUserIDs)
}

func TestAuthService_LoginOrRegisterOAuth_DoesNotAwardLinuxDoAutoCheckinBonusWhenDisabled(t *testing.T) {
	repo := &userRepoStub{nextID: 25}
	rewardRepo := &linuxDoAutoCheckinRewardRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyLinuxDoAutoCheckinBonus: "false",
	}, nil)
	service.SetLinuxDoAutoCheckinRewardRepository(rewardRepo)
	service.SetAutoCheckinRandomIntn(func(int) int { return 4 })
	service.SetAutoCheckinNow(func() time.Time {
		return time.Date(2026, 4, 7, 9, 30, 0, 0, time.Local)
	})

	token, user, autoCheckinResult, err := service.LoginOrRegisterOAuth(context.Background(), "linuxdo-disabled@test.com", "linuxdo-disabled")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.False(t, autoCheckinResult.Awarded)
	require.Zero(t, autoCheckinResult.BonusAmount)
	require.Empty(t, rewardRepo.calls)
	require.Empty(t, repo.balanceUpdates)
}

func TestAuthService_LoginOrRegisterOAuth_AwardsLinuxDoAutoCheckinBonusOnFirstLogin(t *testing.T) {
	repo := &userRepoStub{nextID: 26}
	rewardRepo := &linuxDoAutoCheckinRewardRepoStub{}
	fixedTime := time.Date(2026, 4, 7, 9, 30, 0, 0, time.Local)
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyLinuxDoAutoCheckinBonus: "true",
	}, nil)
	service.SetLinuxDoAutoCheckinRewardRepository(rewardRepo)
	service.SetAutoCheckinRandomIntn(func(int) int { return 2 })
	service.SetAutoCheckinNow(func() time.Time { return fixedTime })

	token, user, autoCheckinResult, err := service.LoginOrRegisterOAuth(context.Background(), "linuxdo-bonus@test.com", "linuxdo-bonus")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.True(t, autoCheckinResult.Awarded)
	require.Equal(t, 3, autoCheckinResult.BonusAmount)
	require.Len(t, rewardRepo.calls, 1)
	require.Equal(t, CreateLinuxDoAutoCheckinRewardInput{
		UserID:      26,
		RewardDate:  time.Date(2026, 4, 7, 0, 0, 0, 0, fixedTime.Location()),
		Source:      LinuxDoAutoCheckinRewardSourceOAuthLogin,
		BonusAmount: 3,
	}, rewardRepo.calls[0])
	require.Equal(t, []balanceUpdateCall{{id: 26, amount: 3}}, repo.balanceUpdates)
	require.Equal(t, 6.5, user.Balance)
}

func TestAuthService_LoginOrRegisterOAuth_DoesNotAwardLinuxDoAutoCheckinBonusTwiceInSameDay(t *testing.T) {
	existingUser := &User{
		ID:           27,
		Email:        "linuxdo-repeat@test.com",
		Username:     "linuxdo-repeat",
		Role:         RoleUser,
		Status:       StatusActive,
		Balance:      2,
		TokenVersion: 1,
	}
	repo := &userRepoStub{
		user: existingUser,
		userByEmail: map[string]*User{
			existingUser.Email: existingUser,
		},
	}
	rewardRepo := &linuxDoAutoCheckinRewardRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyLinuxDoAutoCheckinBonus: "true",
	}, nil)
	service.SetLinuxDoAutoCheckinRewardRepository(rewardRepo)
	service.SetAutoCheckinRandomIntn(func(int) int { return 1 })
	service.SetAutoCheckinNow(func() time.Time {
		return time.Date(2026, 4, 7, 10, 0, 0, 0, time.Local)
	})

	_, firstUser, firstResult, err := service.LoginOrRegisterOAuth(context.Background(), existingUser.Email, existingUser.Username)
	require.NoError(t, err)
	require.True(t, firstResult.Awarded)
	require.Equal(t, 2, firstResult.BonusAmount)
	require.Same(t, existingUser, firstUser)

	_, secondUser, secondResult, err := service.LoginOrRegisterOAuth(context.Background(), existingUser.Email, existingUser.Username)
	require.NoError(t, err)
	require.False(t, secondResult.Awarded)
	require.Zero(t, secondResult.BonusAmount)
	require.Same(t, existingUser, secondUser)
	require.Len(t, rewardRepo.calls, 2)
	require.Equal(t, []balanceUpdateCall{{id: 27, amount: 2}}, repo.balanceUpdates)
	require.Equal(t, 4.0, existingUser.Balance)
}

func TestAuthService_LoginOrRegisterOAuthWithTokenPair_AwardsLinuxDoAutoCheckinBonusForInvitationSignup(t *testing.T) {
	repo := &userRepoStub{nextID: 28}
	rewardRepo := &linuxDoAutoCheckinRewardRepoStub{}
	redeemRepo := &redeemRepoStub{
		codesByCode: map[string]*RedeemCode{
			"invite-auto-checkin": {
				ID:     10,
				Code:   "invite-auto-checkin",
				Type:   RedeemTypeInvitation,
				Status: StatusUnused,
			},
		},
	}
	fixedTime := time.Date(2026, 4, 7, 11, 0, 0, 0, time.Local)
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:  "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyLinuxDoAutoCheckinBonus: "true",
	}, nil)
	service.refreshTokenCache = &refreshTokenCacheStub{}
	service.redeemRepo = redeemRepo
	service.SetLinuxDoAutoCheckinRewardRepository(rewardRepo)
	service.SetAutoCheckinRandomIntn(func(int) int { return 0 })
	service.SetAutoCheckinNow(func() time.Time { return fixedTime })

	tokenPair, user, autoCheckinResult, err := service.LoginOrRegisterOAuthWithTokenPair(
		context.Background(),
		"linuxdo-bonus-invite@test.com",
		"invite-auto-checkin",
		"invite-auto-checkin",
	)
	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.NotEmpty(t, tokenPair.AccessToken)
	require.NotEmpty(t, tokenPair.RefreshToken)
	require.NotNil(t, user)
	require.True(t, autoCheckinResult.Awarded)
	require.Equal(t, 1, autoCheckinResult.BonusAmount)
	require.Equal(t, []int64{10}, redeemRepo.usedCodeIDs)
	require.Equal(t, []int64{28}, redeemRepo.usedUserIDs)
	require.Equal(t, []balanceUpdateCall{{id: 28, amount: 1}}, repo.balanceUpdates)
	require.Len(t, rewardRepo.calls, 1)
}
