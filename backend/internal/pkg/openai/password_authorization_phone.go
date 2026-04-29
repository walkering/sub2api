package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
)

const (
	pathAddPhoneSend             = "/api/accounts/add-phone/send"
	pathPhoneOTPResend           = "/api/accounts/phone-otp/resend"
	pathPhoneOTPValidate         = "/api/accounts/phone-otp/validate"
	defaultHeroSMSBaseURL        = "https://hero-sms.com/stubs/handler_api.php"
	defaultPhonePollInterval     = 5 * time.Second
	defaultPhoneResendAfter      = 30 * time.Second
	defaultPhoneVerificationWait = 3 * time.Minute
	defaultPhoneLeaseMinutes     = 20
	maxPhoneRotationAttempts     = 3
	defaultPhoneMaxBindsPerLease = 3
)

type PhoneOTPProviderConfig struct {
	Provider           string
	BaseURL            string
	APIKey             string
	ServiceCode        string
	Country            string
	Operator           string
	MaxPrice           *float64
	FixedPrice         bool
	LeaseMinutes       int
	PollInterval       time.Duration
	PollIntervalMillis int
	ResendAfter        time.Duration
	ResendAfterSeconds int
}

type passwordAuthorizationAPIError struct {
	Status  int
	Code    string
	Message string
	Body    string
}

func (e *passwordAuthorizationAPIError) Error() string {
	if e == nil {
		return ""
	}
	detail := strings.TrimSpace(e.Message)
	if detail == "" {
		detail = strings.TrimSpace(e.Code)
	}
	if detail == "" {
		detail = strings.TrimSpace(e.Body)
	}
	if detail == "" {
		return fmt.Sprintf("api request failed: status=%d", e.Status)
	}
	return fmt.Sprintf("api request failed: status=%d code=%s message=%s", e.Status, strings.TrimSpace(e.Code), detail)
}

type heroSMSLease struct {
	ProviderLeaseID  string
	PhoneNumber      string
	CountryPhoneCode string
	ExpiresAt        time.Time
}

type pooledHeroSMSLease struct {
	Lease      *heroSMSLease
	BindCount  int
	InUse      bool
	LastUsedAt time.Time
}

type heroSMSMessage struct {
	ID         string
	Code       string
	RawText    string
	ReceivedAt time.Time
}

type heroSMSClient struct {
	session *passwordAuthorizationSession
	config  *PhoneOTPProviderConfig
}

var heroSMSLeaseRegistry = struct {
	mu     sync.Mutex
	leases map[string][]*pooledHeroSMSLease
}{
	leases: make(map[string][]*pooledHeroSMSLease),
}

func normalizePhoneOTPProviderConfig(raw *PhoneOTPProviderConfig) *PhoneOTPProviderConfig {
	if raw == nil {
		return nil
	}
	cfg := *raw
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	if cfg.Provider == "" {
		cfg.Provider = "hero-sms"
	}
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultHeroSMSBaseURL
	}
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.ServiceCode = strings.TrimSpace(cfg.ServiceCode)
	cfg.Country = strings.TrimSpace(cfg.Country)
	cfg.Operator = strings.TrimSpace(cfg.Operator)
	if cfg.LeaseMinutes <= 0 {
		cfg.LeaseMinutes = defaultPhoneLeaseMinutes
	}
	if cfg.PollIntervalMillis > 0 && cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Duration(cfg.PollIntervalMillis) * time.Millisecond
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPhonePollInterval
	}
	if cfg.ResendAfterSeconds > 0 && cfg.ResendAfter <= 0 {
		cfg.ResendAfter = time.Duration(cfg.ResendAfterSeconds) * time.Second
	}
	if cfg.ResendAfter <= 0 {
		cfg.ResendAfter = defaultPhoneResendAfter
	}
	return &cfg
}

func heroSMSPoolKey(cfg *PhoneOTPProviderConfig) string {
	if cfg == nil {
		return ""
	}
	parts := []string{
		strings.ToLower(strings.TrimSpace(cfg.Provider)),
		strings.TrimSpace(cfg.BaseURL),
		strings.TrimSpace(cfg.APIKey),
		strings.TrimSpace(cfg.ServiceCode),
		strings.TrimSpace(cfg.Country),
		strings.TrimSpace(cfg.Operator),
		strconv.FormatBool(cfg.FixedPrice),
	}
	if cfg.MaxPrice != nil {
		parts = append(parts, strconv.FormatFloat(*cfg.MaxPrice, 'f', -1, 64))
	}
	return strings.Join(parts, "|")
}

func newPasswordAuthorizationAPIError(status int, body []byte) error {
	return &passwordAuthorizationAPIError{
		Status:  status,
		Code:    extractAPIErrorCode(body),
		Message: extractAPIErrorMessage(body),
		Body:    trimBody(body),
	}
}

func isFraudGuardPhoneError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *passwordAuthorizationAPIError
	if errors.As(err, &apiErr) {
		return strings.EqualFold(strings.TrimSpace(apiErr.Code), "fraud_guard") ||
			strings.Contains(strings.ToLower(strings.TrimSpace(apiErr.Message)), "suspicious behavior")
	}
	return strings.Contains(strings.ToLower(err.Error()), "fraud_guard")
}

func isRetryablePhoneOTPValidationError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "invalid code") ||
		strings.Contains(text, "incorrect code") ||
		strings.Contains(text, "verification code")
}

func shouldRetirePhoneLease(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return isFraudGuardPhoneError(err) ||
		strings.Contains(text, "phone_number_in_use") ||
		strings.Contains(text, "phone number is already in use") ||
		strings.Contains(text, "phone_max_usage_exceeded") ||
		strings.Contains(text, "max usage exceeded") ||
		strings.Contains(text, "resend limit") ||
		strings.Contains(text, "too many requests")
}

func handleAddPhoneVerification(
	ctx context.Context,
	session *passwordAuthorizationSession,
	config *PhoneOTPProviderConfig,
	continueURL string,
	pageType string,
	logf func(level, message string, attrs ...any),
	stepPrefix string,
) (string, string, error) {
	cfg := normalizePhoneOTPProviderConfig(config)
	if cfg == nil || cfg.Provider != "hero-sms" || cfg.APIKey == "" || cfg.ServiceCode == "" || cfg.Country == "" {
		return "", "", ErrPasswordAuthorizationAddPhone
	}
	stepLogf := func(level, step, message string, attrs ...any) {
		if logf == nil {
			return
		}
		logf(level, formatPasswordAuthorizationStep(stepPrefix, step, message), attrs...)
	}
	client := &heroSMSClient{session: session, config: cfg}
	var lastErr error

	for attempt := 1; attempt <= maxPhoneRotationAttempts; attempt++ {
		leaseHandle, reused, err := client.checkoutLease(ctx)
		if err != nil {
			return "", "", err
		}
		lease := leaseHandle.Lease
		if reused {
			stepLogf("info", "1", fmt.Sprintf("复用已有手机号，第 %d/%d 次尝试", attempt, maxPhoneRotationAttempts), "phone", maskPhoneNumberForLog(lease.PhoneNumber), "bind_count", leaseHandle.BindCount)
		} else {
			stepLogf("info", "1", fmt.Sprintf("开始向 HeroSMS 申请手机号，第 %d/%d 次", attempt, maxPhoneRotationAttempts))
			stepLogf("info", "1", "申请手机号成功", "phone", maskPhoneNumberForLog(lease.PhoneNumber))
		}

		nextContinueURL, nextPageType, err := postAddPhoneSend(ctx, session, lease.PhoneNumber)
		if err != nil {
			lastErr = err
			client.retireLease(ctx, leaseHandle)
			if shouldRetirePhoneLease(err) && attempt < maxPhoneRotationAttempts {
				stepLogf("warn", "2", "当前号码已不可继续使用，已废弃并重新获取新号码", "phone", maskPhoneNumberForLog(lease.PhoneNumber))
				continue
			}
			client.releaseLease(leaseHandle)
			return "", "", err
		}
		if strings.TrimSpace(nextContinueURL) == "" {
			nextContinueURL = passwordAuthorizationIssuer + "/phone-verification"
		}
		if strings.TrimSpace(nextPageType) == "" {
			nextPageType = "phone_otp_verification"
		}
		stepLogf("info", "2", "手机号提交成功，"+describeAuthPage(nextContinueURL, nextPageType), "phone", maskPhoneNumberForLog(lease.PhoneNumber))

		nextContinueURL, nextPageType, err = waitAndValidatePhoneOTP(ctx, session, client, lease, logf, extendPasswordAuthorizationStepPrefix(stepPrefix, "3"))
		if err != nil {
			lastErr = err
			client.retireLease(ctx, leaseHandle)
			if shouldRetirePhoneLease(err) && attempt < maxPhoneRotationAttempts {
				stepLogf("warn", "3", "短信流程判定当前号码不可继续使用，已废弃并重新获取", "phone", maskPhoneNumberForLog(lease.PhoneNumber))
				continue
			}
			client.releaseLease(leaseHandle)
			return "", "", err
		}

		if retired, bindCount, finishErr := client.markLeaseSuccess(ctx, leaseHandle); finishErr != nil {
			stepLogf("warn", "4", "手机号验证成功，但更新租约状态失败: "+finishErr.Error(), "phone", maskPhoneNumberForLog(lease.PhoneNumber))
		} else if retired {
			stepLogf("info", "4", "当前手机号已达到复用上限，已完成激活并退出池", "phone", maskPhoneNumberForLog(lease.PhoneNumber), "bind_count", bindCount)
		} else {
			stepLogf("info", "4", "当前手机号保留复用", "phone", maskPhoneNumberForLog(lease.PhoneNumber), "bind_count", bindCount)
		}
		return nextContinueURL, nextPageType, nil
	}

	if lastErr != nil {
		return "", "", lastErr
	}
	return "", "", ErrPasswordAuthorizationAddPhone
}

func waitAndValidatePhoneOTP(
	ctx context.Context,
	session *passwordAuthorizationSession,
	client *heroSMSClient,
	lease *heroSMSLease,
	logf func(level, message string, attrs ...any),
	stepPrefix string,
) (string, string, error) {
	stepLogf := func(level, step, message string, attrs ...any) {
		if logf == nil {
			return
		}
		logf(level, formatPasswordAuthorizationStep(stepPrefix, step, message), attrs...)
	}
	deadline := time.Now().Add(defaultPhoneVerificationWait)
	if !lease.ExpiresAt.IsZero() && lease.ExpiresAt.Before(deadline) {
		deadline = lease.ExpiresAt
	}
	startedAt := time.Now()
	resendRequested := false
	rejectedCodes := map[string]struct{}{}

	for {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		if time.Now().After(deadline) {
			return "", "", fmt.Errorf("wait phone otp timed out")
		}

		if _, err := client.activateForPolling(ctx, lease); err != nil {
			stepLogf("warn", "1", "轮询前激活号码失败: "+err.Error(), "phone", maskPhoneNumberForLog(lease.PhoneNumber))
		}

		messages, err := client.pollMessages(ctx, lease)
		if err != nil {
			stepLogf("warn", "1", "轮询短信失败: "+err.Error(), "phone", maskPhoneNumberForLog(lease.PhoneNumber))
		} else {
			for index := len(messages) - 1; index >= 0; index-- {
				message := messages[index]
				code := strings.TrimSpace(message.Code)
				if code == "" {
					continue
				}
				if _, exists := rejectedCodes[code]; exists {
					continue
				}
				stepLogf("info", "2", "已获取手机号验证码，开始提交", "phone", maskPhoneNumberForLog(lease.PhoneNumber))
				continueURL, pageType, validateErr := postValidatePhoneOTP(ctx, session, code)
				if validateErr == nil {
					return continueURL, pageType, nil
				}
				if isFraudGuardPhoneError(validateErr) {
					return "", "", validateErr
				}
				if isRetryablePhoneOTPValidationError(validateErr) {
					rejectedCodes[code] = struct{}{}
					stepLogf("warn", "2", "手机号验证码被拒绝，继续等待下一条短信: "+validateErr.Error(), "phone", maskPhoneNumberForLog(lease.PhoneNumber))
					break
				}
				return "", "", validateErr
			}
		}

		if !resendRequested && time.Since(startedAt) >= client.config.ResendAfter {
			stepLogf("info", "3", "等待短信超时，尝试请求重发验证码", "phone", maskPhoneNumberForLog(lease.PhoneNumber))
			if err := postPhoneOTPResend(ctx, session); err != nil {
				return "", "", err
			}
			if _, err := client.requestAnotherSMS(ctx, lease); err != nil {
				stepLogf("warn", "3", "HeroSMS 重发短信请求失败，继续等待页面侧短信: "+err.Error(), "phone", maskPhoneNumberForLog(lease.PhoneNumber))
			}
			resendRequested = true
			stepLogf("info", "3", "短信重发请求已提交", "phone", maskPhoneNumberForLog(lease.PhoneNumber))
		}

		wait := client.config.PollInterval
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		if wait <= 0 {
			return "", "", fmt.Errorf("wait phone otp timed out")
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", "", ctx.Err()
		case <-timer.C:
		}
	}
}

func postAddPhoneSend(ctx context.Context, session *passwordAuthorizationSession, phoneNumber string) (string, string, error) {
	requestBody := map[string]any{
		"phone_number": strings.TrimSpace(phoneNumber),
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/add-phone")
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+pathAddPhoneSend, headers, requestBody, true)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if isRedirectStatus(resp.StatusCode) {
		return resolveLocation(passwordAuthorizationIssuer+pathAddPhoneSend, resp.Header.Get("Location")), "", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", newPasswordAuthorizationAPIError(resp.StatusCode, raw)
	}
	if len(raw) == 0 {
		return passwordAuthorizationIssuer + "/phone-verification", "phone_otp_verification", nil
	}
	var payload passwordAuthorizationPage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", err
	}
	return payload.ContinueURL, payload.Page.Type, nil
}

func postPhoneOTPResend(ctx context.Context, session *passwordAuthorizationSession) error {
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/phone-verification")
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+pathPhoneOTPResend, headers, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newPasswordAuthorizationAPIError(resp.StatusCode, raw)
	}
	return nil
}

func postValidatePhoneOTP(ctx context.Context, session *passwordAuthorizationSession, code string) (string, string, error) {
	requestBody := map[string]any{
		"code": strings.TrimSpace(code),
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/phone-verification")
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+pathPhoneOTPValidate, headers, requestBody, true)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if isRedirectStatus(resp.StatusCode) {
		return resolveLocation(passwordAuthorizationIssuer+pathPhoneOTPValidate, resp.Header.Get("Location")), "", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", newPasswordAuthorizationAPIError(resp.StatusCode, raw)
	}
	if len(raw) == 0 {
		return "", "", nil
	}
	var payload passwordAuthorizationPage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", err
	}
	return payload.ContinueURL, payload.Page.Type, nil
}

func (c *heroSMSClient) acquireNumber(ctx context.Context) (*heroSMSLease, error) {
	if c == nil || c.session == nil || c.config == nil {
		return nil, fmt.Errorf("hero sms client is not configured")
	}
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("action", "getNumberV2")
	params.Set("service", c.config.ServiceCode)
	params.Set("country", c.config.Country)
	if c.config.Operator != "" {
		params.Set("operator", c.config.Operator)
	}
	if c.config.MaxPrice != nil {
		params.Set("maxPrice", strconv.FormatFloat(*c.config.MaxPrice, 'f', -1, 64))
		if c.config.FixedPrice {
			params.Set("fixedPrice", "true")
		}
	}
	payload, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}
	body, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(normalizeHeroSMSMessage(payload))
	}
	providerLeaseID := firstNonEmptyString(
		stringFromAny(body["activationId"]),
		stringFromAny(body["id"]),
	)
	phoneNumber := firstNonEmptyString(
		stringFromAny(body["phoneNumber"]),
		stringFromAny(body["phone"]),
	)
	if providerLeaseID == "" || phoneNumber == "" {
		return nil, fmt.Errorf("hero sms response missing activationId or phoneNumber")
	}
	expiresAt := time.Now().Add(time.Duration(c.config.LeaseMinutes) * time.Minute)
	if ts := heroSMSTimestamp(firstNonEmptyString(stringFromAny(body["activationEndTime"]), stringFromAny(body["expiresAt"]))); !ts.IsZero() {
		expiresAt = ts
	}
	return &heroSMSLease{
		ProviderLeaseID:  providerLeaseID,
		PhoneNumber:      phoneNumber,
		CountryPhoneCode: strings.TrimSpace(stringFromAny(body["countryPhoneCode"])),
		ExpiresAt:        expiresAt,
	}, nil
}

func (c *heroSMSClient) pollMessages(ctx context.Context, lease *heroSMSLease) ([]heroSMSMessage, error) {
	if lease == nil || lease.ProviderLeaseID == "" {
		return nil, nil
	}
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("action", "getStatusV2")
	params.Set("id", lease.ProviderLeaseID)
	payload, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}
	if text, ok := payload.(string); ok {
		switch strings.TrimSpace(text) {
		case "", "STATUS_WAIT_CODE", "STATUS_WAIT_RETRY", "STATUS_CANCEL":
			return nil, nil
		default:
			return nil, fmt.Errorf(normalizeHeroSMSMessage(text))
		}
	}
	body, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid hero sms status response")
	}
	messages := append(parseHeroSMSMessageEntries(body["sms"], lease.ProviderLeaseID, "sms"), parseHeroSMSMessageEntries(body["call"], lease.ProviderLeaseID, "call")...)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ReceivedAt.Before(messages[j].ReceivedAt)
	})
	return messages, nil
}

func (c *heroSMSClient) requestAnotherSMS(ctx context.Context, lease *heroSMSLease) (any, error) {
	if lease == nil || lease.ProviderLeaseID == "" {
		return nil, nil
	}
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("action", "setStatus")
	params.Set("id", lease.ProviderLeaseID)
	params.Set("status", "3")
	return c.request(ctx, params)
}

func (c *heroSMSClient) activateForPolling(ctx context.Context, lease *heroSMSLease) (any, error) {
	if lease == nil || lease.ProviderLeaseID == "" {
		return nil, nil
	}
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("action", "setStatus")
	params.Set("id", lease.ProviderLeaseID)
	params.Set("status", "3")
	payload, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}
	if text, ok := payload.(string); ok {
		switch strings.TrimSpace(text) {
		case "", "ACCESS_RETRY_GET", "ACCESS_READY", "ACCESS_ACTIVATION":
			return payload, nil
		default:
			return nil, fmt.Errorf(normalizeHeroSMSMessage(text))
		}
	}
	return payload, nil
}

func (c *heroSMSClient) maxBindsPerLease() int {
	return defaultPhoneMaxBindsPerLease
}

func (c *heroSMSClient) checkoutLease(ctx context.Context) (*pooledHeroSMSLease, bool, error) {
	if reused := c.checkoutReusableLease(); reused != nil {
		return reused, true, nil
	}
	lease, err := c.acquireNumber(ctx)
	if err != nil {
		return nil, false, err
	}
	handle := &pooledHeroSMSLease{
		Lease:      lease,
		BindCount:  0,
		InUse:      true,
		LastUsedAt: time.Now(),
	}
	c.registerLease(handle)
	return handle, false, nil
}

func (c *heroSMSClient) checkoutReusableLease() *pooledHeroSMSLease {
	poolKey := heroSMSPoolKey(c.config)
	if poolKey == "" {
		return nil
	}
	now := time.Now()
	heroSMSLeaseRegistry.mu.Lock()
	defer heroSMSLeaseRegistry.mu.Unlock()

	leases := heroSMSLeaseRegistry.leases[poolKey]
	filtered := leases[:0]
	var selected *pooledHeroSMSLease
	for _, item := range leases {
		if item == nil || item.Lease == nil || item.Lease.ProviderLeaseID == "" || item.Lease.PhoneNumber == "" {
			continue
		}
		if !item.Lease.ExpiresAt.IsZero() && !item.Lease.ExpiresAt.After(now) {
			continue
		}
		if item.BindCount >= c.maxBindsPerLease() {
			continue
		}
		if selected == nil && !item.InUse {
			item.InUse = true
			item.LastUsedAt = now
			selected = item
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		delete(heroSMSLeaseRegistry.leases, poolKey)
	} else {
		heroSMSLeaseRegistry.leases[poolKey] = filtered
	}
	return selected
}

func (c *heroSMSClient) registerLease(handle *pooledHeroSMSLease) {
	if handle == nil || handle.Lease == nil {
		return
	}
	poolKey := heroSMSPoolKey(c.config)
	if poolKey == "" {
		return
	}
	heroSMSLeaseRegistry.mu.Lock()
	defer heroSMSLeaseRegistry.mu.Unlock()
	heroSMSLeaseRegistry.leases[poolKey] = append(heroSMSLeaseRegistry.leases[poolKey], handle)
}

func (c *heroSMSClient) releaseLease(handle *pooledHeroSMSLease) {
	if handle == nil {
		return
	}
	heroSMSLeaseRegistry.mu.Lock()
	defer heroSMSLeaseRegistry.mu.Unlock()
	handle.InUse = false
	handle.LastUsedAt = time.Now()
}

func (c *heroSMSClient) removeLease(handle *pooledHeroSMSLease) {
	if handle == nil {
		return
	}
	poolKey := heroSMSPoolKey(c.config)
	if poolKey == "" {
		return
	}
	heroSMSLeaseRegistry.mu.Lock()
	defer heroSMSLeaseRegistry.mu.Unlock()
	leases := heroSMSLeaseRegistry.leases[poolKey]
	filtered := leases[:0]
	for _, item := range leases {
		if item != handle {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		delete(heroSMSLeaseRegistry.leases, poolKey)
	} else {
		heroSMSLeaseRegistry.leases[poolKey] = filtered
	}
}

func (c *heroSMSClient) retireLease(ctx context.Context, handle *pooledHeroSMSLease) {
	if handle == nil || handle.Lease == nil {
		return
	}
	if handle.BindCount > 0 {
		_ = c.finishActivation(ctx, handle.Lease)
	} else {
		_ = c.cancelActivation(ctx, handle.Lease)
	}
	c.removeLease(handle)
}

func (c *heroSMSClient) markLeaseSuccess(ctx context.Context, handle *pooledHeroSMSLease) (bool, int, error) {
	if handle == nil || handle.Lease == nil {
		return false, 0, fmt.Errorf("lease handle is not configured")
	}
	handle.BindCount++
	handle.LastUsedAt = time.Now()
	handle.InUse = false
	if handle.BindCount >= c.maxBindsPerLease() {
		if err := c.finishActivation(ctx, handle.Lease); err != nil {
			return true, handle.BindCount, err
		}
		c.removeLease(handle)
		return true, handle.BindCount, nil
	}
	return false, handle.BindCount, nil
}

func (c *heroSMSClient) finishActivation(ctx context.Context, lease *heroSMSLease) error {
	return c.finishWithAction(ctx, lease, "finishActivation")
}

func (c *heroSMSClient) cancelActivation(ctx context.Context, lease *heroSMSLease) error {
	return c.finishWithAction(ctx, lease, "cancelActivation")
}

func (c *heroSMSClient) finishWithAction(ctx context.Context, lease *heroSMSLease, action string) error {
	if lease == nil || lease.ProviderLeaseID == "" {
		return nil
	}
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("action", action)
	params.Set("id", lease.ProviderLeaseID)
	payload, err := c.request(ctx, params)
	if err != nil {
		return err
	}
	if text, ok := payload.(string); ok {
		normalized := strings.TrimSpace(text)
		if normalized == "" || normalized == "ACCESS_ACTIVATION" || normalized == "ACCESS_CANCEL" || normalized == "STATUS_CANCEL" {
			return nil
		}
		return fmt.Errorf(normalizeHeroSMSMessage(normalized))
	}
	return nil
}

func (c *heroSMSClient) request(ctx context.Context, params url.Values) (any, error) {
	targetURL := strings.TrimRight(c.config.BaseURL, "/")
	if targetURL == "" {
		targetURL = defaultHeroSMSBaseURL
	}
	if encoded := params.Encode(); encoded != "" {
		targetURL += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header = http.Header{}
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("user-agent", c.session.userAgent)
	resp, err := c.session.do(req, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	payload := parseHeroSMSPayload(raw)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(normalizeHeroSMSMessage(payload))
	}
	return payload, nil
}

func parseHeroSMSPayload(raw []byte) any {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err == nil {
		return payload
	}
	return text
}

func parseHeroSMSMessageEntries(raw any, leaseID string, channel string) []heroSMSMessage {
	if raw == nil {
		return nil
	}
	entries := make([]any, 0, 1)
	switch typed := raw.(type) {
	case []any:
		entries = typed
	default:
		entries = append(entries, typed)
	}
	result := make([]heroSMSMessage, 0, len(entries))
	for index, entry := range entries {
		switch value := entry.(type) {
		case string:
			text := strings.TrimSpace(value)
			if text == "" {
				continue
			}
			result = append(result, heroSMSMessage{
				ID:         fmt.Sprintf("%s-%s-%d", leaseID, channel, index+1),
				Code:       extractVerificationCode(text),
				RawText:    text,
				ReceivedAt: time.Now().Add(time.Duration(index) * time.Millisecond),
			})
		case map[string]any:
			rawText := firstNonEmptyString(
				stringFromAny(value["text"]),
				stringFromAny(value["message"]),
				stringFromAny(value["content"]),
				stringFromAny(value["body"]),
				stringFromAny(value["sms"]),
				stringFromAny(value["fullText"]),
				stringFromAny(value["code"]),
			)
			result = append(result, heroSMSMessage{
				ID: firstNonEmptyString(
					stringFromAny(value["id"]),
					stringFromAny(value["smsId"]),
					stringFromAny(value["messageId"]),
					stringFromAny(value["msgId"]),
					fmt.Sprintf("%s-%s-%d", leaseID, channel, index+1),
				),
				Code: firstNonEmptyString(
					stringFromAny(value["code"]),
					stringFromAny(value["otpCode"]),
					extractVerificationCode(rawText),
				),
				RawText: rawText,
				ReceivedAt: firstNonZeroTime(
					heroSMSTimestamp(firstNonEmptyString(
						stringFromAny(value["dateTime"]),
						stringFromAny(value["time"]),
						stringFromAny(value["createdAt"]),
						stringFromAny(value["timestamp"]),
					)),
					time.Now().Add(time.Duration(index)*time.Millisecond),
				),
			})
		}
	}
	return result
}

func normalizeHeroSMSMessage(payload any) string {
	switch value := payload.(type) {
	case string:
		switch strings.TrimSpace(value) {
		case "NO_NUMBERS":
			return "HeroSMS 当前没有可用号码。"
		case "BAD_KEY":
			return "HeroSMS API Key 无效。"
		case "NO_BALANCE":
			return "HeroSMS 余额不足。"
		case "NO_ACTIVATION":
			return "HeroSMS 激活记录不存在。"
		case "STATUS_CANCEL":
			return "HeroSMS 激活已取消。"
		default:
			return strings.TrimSpace(value)
		}
	case map[string]any:
		return firstNonEmptyString(
			stringFromAny(value["message"]),
			stringFromAny(value["error"]),
			stringFromAny(value["detail"]),
			stringFromAny(value["code"]),
		)
	default:
		return fmt.Sprintf("%v", payload)
	}
}

func extractVerificationCode(text string) string {
	if match := regexp.MustCompile(`(?i)(?:code|otp|验证码)[^0-9]{0,12}([0-9]{4,8})`).FindStringSubmatch(text); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`\b([0-9]{4,8})\b`).FindStringSubmatch(text); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func heroSMSTimestamp(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", trimmed); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02T15:04:05", trimmed); err == nil {
		return parsed
	}
	return time.Time{}
}

func maskPhoneNumberForLog(phoneNumber string) string {
	value := strings.TrimSpace(phoneNumber)
	if value == "" || len(value) <= 5 {
		return value
	}
	return value[:3] + "***" + value[len(value)-2:]
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}
