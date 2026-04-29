package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	stdhttp "net/http"
	stdcookiejar "net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

var (
	ErrPasswordAuthorizationAddPhone = errors.New("openai password authorization requires add-phone")
	ErrPasswordAuthorizationEmailOTP = errors.New("openai password authorization requires email otp")
)

const (
	passwordAuthorizationIssuer  = "https://auth.openai.com"
	passwordAuthorizationChatGPT = "https://chatgpt.com"
	passwordAuthorizationTimeout = 90 * time.Second
	defaultFreemailBaseURL       = "https://freemail.walker-feng.workers.dev"

	sentinelEndpoint              = "https://sentinel.openai.com/backend-api/sentinel/req"
	sentinelPowMaxTry             = 500000
	sentinelPowErrorPrefix        = "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D"
	sentinelFlowAuthorizeContinue = "authorize_continue"
	sentinelFlowEmailOTPValidate  = "email_otp_validate"
	sentinelFlowPasswordVerify    = "password_verify"
	sentinelFlowRegisterUser      = "username_password_create"
	sentinelFlowCreateAccount     = "oauth_create_account"
	pathAuthorizeContinue         = "/api/accounts/authorize/continue"
	pathRegisterUser              = "/api/accounts/user/register"
	pathRegisterEmailOTPSend      = "/api/accounts/email-otp/send"
	pathPasswordlessSendOTP       = "/api/accounts/passwordless/send-otp"
	pathEmailOTPValidate          = "/api/accounts/email-otp/validate"
	pathPasswordVerify            = "/api/accounts/password/verify"
	pathCreateAccount             = "/api/accounts/create_account"
	defaultPasswordLoginUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.7103.92 Safari/537.36"
	defaultPasswordLoginSecCHUA   = `"Chromium";v="136", "Google Chrome";v="136", "Not.A/Brand";v="99"`
	defaultPasswordTLSProfile     = "chrome_144"
	defaultPasswordRequestTimeout = 60 * time.Second
)

type PasswordAuthorizationInput struct {
	AuthURL        string
	Email          string
	Password       string
	WorkflowKind   string
	ProxyURL       string
	Logger         *slog.Logger
	Logf           func(level, message string)
	StepPrefix     string
	FreeMailConfig *FreeMailOTPConfig
	PhoneConfig    *PhoneOTPProviderConfig
}

type FreeMailOTPConfig struct {
	BaseURL            string
	Username           string
	Password           string
	Domain             string
	MaxAttempts        int
	Interval           time.Duration
	PollIntervalMillis int
	ResendAfter        time.Duration
	ResendAfterSeconds int
}

type passwordAuthorizationPage struct {
	ContinueURL string `json:"continue_url"`
	Page        struct {
		Type string `json:"type"`
	} `json:"page"`
	Data struct {
		Orgs []struct {
			ID       string `json:"id"`
			Projects []struct {
				ID string `json:"id"`
			} `json:"projects"`
		} `json:"orgs"`
	} `json:"data"`
}

type passwordAuthorizationSession struct {
	client     passwordAuthorizationHTTPClient
	jar        http.CookieJar
	deviceID   string
	userAgent  string
	secCHUA    string
	cookiesURL *url.URL
	proxyURL   string
	logf       func(level, message string, attrs ...any)
}

type passwordAuthorizationHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type passwordAuthorizationRedirectPolicySetter interface {
	SetFollowRedirect(followRedirect bool)
	GetFollowRedirect() bool
}

type sentinelChallenge struct {
	Token       string `json:"token"`
	Proofofwork struct {
		Required   bool   `json:"required"`
		Seed       string `json:"seed"`
		Difficulty string `json:"difficulty"`
	} `json:"proofofwork"`
	Turnstile struct {
		DX any `json:"dx"`
	} `json:"turnstile"`
	RequirementsToken string `json:"-"`
}

type authSessionCookie struct {
	Workspaces []struct {
		ID string `json:"id"`
	} `json:"workspaces"`
}

type freemailClient struct {
	baseURL  string
	username string
	password string
	client   *stdhttp.Client
	logf     func(level, message string, attrs ...any)
}

type freemailSessionResponse struct {
	Authenticated bool `json:"authenticated"`
}

type freemailMessage struct {
	ID               string
	Subject          string
	Sender           string
	BodyPreview      string
	Raw              string
	ReceivedDateTime string
	VerificationCode string
}

type freemailCodeCandidate struct {
	code       string
	receivedAt time.Time
	score      int
}

func AcquireAuthorizationCodeWithPassword(ctx context.Context, input PasswordAuthorizationInput) (string, error) {
	authURL := strings.TrimSpace(input.AuthURL)
	email := strings.TrimSpace(input.Email)
	if authURL == "" {
		return "", fmt.Errorf("auth url is required")
	}
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	if input.FreeMailConfig == nil {
		return "", fmt.Errorf("email otp provider is required")
	}

	logger := input.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logf := func(level, message string, attrs ...any) {
		switch strings.ToLower(strings.TrimSpace(level)) {
		case "warn":
			logger.Warn(message, attrs...)
		case "error":
			logger.Error(message, attrs...)
		default:
			logger.Info(message, attrs...)
		}
		if input.Logf != nil {
			input.Logf(level, message+formatPasswordAuthorizationLogAttrs(attrs...))
		}
	}
	stepLogf := func(level, step, message string, attrs ...any) {
		logf(level, formatPasswordAuthorizationStep(input.StepPrefix, step, message), attrs...)
	}

	session, err := newPasswordAuthorizationSession(strings.TrimSpace(input.ProxyURL), logf)
	if err != nil {
		return "", err
	}
	sentinelClient, err := newSentinelClient(strings.TrimSpace(input.ProxyURL), logf)
	if err != nil {
		return "", err
	}

	stepLogf("info", "", "开始 OpenAI 纯 HTTP 登录", "email", email)

	passwordSubmittedAt := time.Now().UTC()
	finalPage, err := advancePasswordAuthorizationFlow(ctx, &passwordAuthorizationFlowContext{
		authURL:             authURL,
		email:               email,
		password:            input.Password,
		passwordSubmittedAt: passwordSubmittedAt,
		workflowKind:        normalizePasswordAuthorizationWorkflowKind(input.WorkflowKind),
		baseStepPrefix:      input.StepPrefix,
		session:             session,
		sentinel:            sentinelClient,
		freeMailConfig:      input.FreeMailConfig,
		phoneConfig:         input.PhoneConfig,
		logf:                logf,
		stepLogf:            stepLogf,
	}, passwordAuthorizationFlowPage{})
	if err != nil {
		return "", err
	}
	code := extractCodeFromURL(finalPage.ContinueURL)
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("authorization code not found in workflow result")
	}
	return code, nil
}

func newPasswordAuthorizationSession(proxyURL string, logf func(level, message string, attrs ...any)) (*passwordAuthorizationSession, error) {
	jar, err := newPasswordAuthorizationCookieJar()
	if err != nil {
		return nil, err
	}
	client, resolvedProfile, err := newPasswordAuthorizationHTTPClient(strings.TrimSpace(proxyURL), jar)
	if err != nil {
		return nil, err
	}
	userAgent, secCHUA := resolvePasswordAuthorizationClientIdentity(resolvedProfile)
	deviceID, err := generateDeviceID()
	if err != nil {
		return nil, err
	}
	baseURL, err := url.Parse(passwordAuthorizationIssuer)
	if err != nil {
		return nil, err
	}
	session := &passwordAuthorizationSession{
		client:     client,
		jar:        jar,
		deviceID:   deviceID,
		userAgent:  userAgent,
		secCHUA:    secCHUA,
		cookiesURL: baseURL,
		proxyURL:   strings.TrimSpace(proxyURL),
		logf:       logf,
	}
	session.seedCookies()
	return session, nil
}

func newPasswordAuthorizationCookieJar() (http.CookieJar, error) {
	return tls_client.NewCookieJar(), nil
}

func resolvePasswordAuthorizationTLSProfile(value string) (string, profiles.ClientProfile, error) {
	profileName := strings.ToLower(strings.TrimSpace(value))
	if profileName == "" {
		profileName = defaultPasswordTLSProfile
	}
	profile, ok := profiles.MappedTLSClients[profileName]
	if !ok {
		return "", profiles.ClientProfile{}, fmt.Errorf("unsupported tls_client_profile: %s", profileName)
	}
	return profileName, profile, nil
}

func newPasswordAuthorizationHTTPClient(proxyURL string, jar http.CookieJar) (passwordAuthorizationHTTPClient, string, error) {
	profileName, profile, err := resolvePasswordAuthorizationTLSProfile(defaultPasswordTLSProfile)
	if err != nil {
		return nil, "", err
	}
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(int(defaultPasswordRequestTimeout / time.Second)),
		tls_client.WithClientProfile(profile),
		tls_client.WithCookieJar(jar),
	}
	if strings.TrimSpace(proxyURL) != "" {
		options = append(options, tls_client.WithProxyUrl(strings.TrimSpace(proxyURL)))
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, "", err
	}
	client.SetFollowRedirect(true)
	return client, profileName, nil
}

func resolvePasswordAuthorizationClientIdentity(profileName string) (string, string) {
	major, ok := passwordAuthorizationChromeMajorFromProfile(profileName)
	if ok {
		return fmt.Sprintf(
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36",
				major,
			),
			fmt.Sprintf(`"Chromium";v="%d", "Google Chrome";v="%d", "Not.A/Brand";v="99"`, major, major)
	}
	return defaultPasswordLoginUserAgent, defaultPasswordLoginSecCHUA
}

func passwordAuthorizationChromeMajorFromProfile(profileName string) (int, bool) {
	profileName = strings.ToLower(strings.TrimSpace(profileName))
	if !strings.HasPrefix(profileName, "chrome_") {
		return 0, false
	}
	version := strings.TrimPrefix(profileName, "chrome_")
	if idx := strings.Index(version, "_"); idx >= 0 {
		version = version[:idx]
	}
	major, err := strconv.Atoi(version)
	if err != nil || major <= 0 {
		return 0, false
	}
	return major, true
}

func (s *passwordAuthorizationSession) seedCookies() {
	if s == nil || s.jar == nil || s.cookiesURL == nil {
		return
	}
	for _, domain := range []string{s.cookiesURL.Hostname(), "." + s.cookiesURL.Hostname()} {
		s.jar.SetCookies(s.cookiesURL, []*http.Cookie{
			{Name: "oai-did", Value: s.deviceID, Domain: domain, Path: "/"},
		})
	}
}

func (s *passwordAuthorizationSession) cookies() []*http.Cookie {
	if s == nil || s.jar == nil || s.cookiesURL == nil {
		return nil
	}
	return s.jar.Cookies(s.cookiesURL)
}

func (s *passwordAuthorizationSession) do(req *http.Request, follow bool) (*http.Response, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("flow session is not configured")
	}
	if setter, ok := s.client.(passwordAuthorizationRedirectPolicySetter); ok {
		previous := setter.GetFollowRedirect()
		if previous != follow {
			setter.SetFollowRedirect(follow)
			defer setter.SetFollowRedirect(previous)
		}
		return s.client.Do(req)
	}
	if follow {
		return s.client.Do(req)
	}
	return s.client.Do(req)
}

func (s *passwordAuthorizationSession) resetHTTPState() error {
	if s == nil {
		return fmt.Errorf("flow session is not configured")
	}
	jar, err := newPasswordAuthorizationCookieJar()
	if err != nil {
		return err
	}
	client, resolvedProfile, err := newPasswordAuthorizationHTTPClient(s.proxyURL, jar)
	if err != nil {
		return err
	}
	s.jar = jar
	s.client = client
	s.userAgent, s.secCHUA = resolvePasswordAuthorizationClientIdentity(resolvedProfile)
	if s.cookiesURL != nil {
		s.seedCookies()
	}
	return nil
}

func (s *passwordAuthorizationSession) commonHeaders(referer string) http.Header {
	headers := http.Header{}
	headers.Set("accept", "application/json")
	headers.Set("accept-language", "en-US,en;q=0.9")
	headers.Set("content-type", "application/json")
	headers.Set("origin", passwordAuthorizationIssuer)
	headers.Set("user-agent", s.userAgent)
	headers.Set("sec-ch-ua", s.secCHUA)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-platform", `"Windows"`)
	headers.Set("sec-fetch-dest", "empty")
	headers.Set("sec-fetch-mode", "cors")
	headers.Set("sec-fetch-site", "same-origin")
	headers.Set("oai-device-id", s.deviceID)
	if strings.TrimSpace(referer) != "" {
		headers.Set("referer", referer)
	}
	return headers
}

func (s *passwordAuthorizationSession) navigateHeaders(referer string) http.Header {
	headers := http.Header{}
	headers.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	headers.Set("accept-language", "en-US,en;q=0.9")
	headers.Set("user-agent", s.userAgent)
	headers.Set("sec-ch-ua", s.secCHUA)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-platform", `"Windows"`)
	headers.Set("sec-fetch-dest", "document")
	headers.Set("sec-fetch-mode", "navigate")
	headers.Set("sec-fetch-site", "same-origin")
	headers.Set("sec-fetch-user", "?1")
	headers.Set("upgrade-insecure-requests", "1")
	if strings.TrimSpace(referer) != "" {
		headers.Set("referer", referer)
	}
	return headers
}

type sentinelClient struct {
	client passwordAuthorizationHTTPClient
	logf   func(level, message string, attrs ...any)
}

func newSentinelClient(proxyURL string, logf func(level, message string, attrs ...any)) (*sentinelClient, error) {
	_, profile, err := resolvePasswordAuthorizationTLSProfile(defaultPasswordTLSProfile)
	if err != nil {
		return nil, err
	}
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(15),
		tls_client.WithClientProfile(profile),
	}
	if strings.TrimSpace(proxyURL) != "" {
		options = append(options, tls_client.WithProxyUrl(strings.TrimSpace(proxyURL)))
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}
	client.SetFollowRedirect(true)
	return &sentinelClient{client: client, logf: logf}, nil
}

func (c *sentinelClient) token(ctx context.Context, session *passwordAuthorizationSession, flow string) (string, error) {
	requirementsToken := generateRequirementsToken(session.deviceID, session.userAgent)
	body := map[string]any{
		"p":    requirementsToken,
		"id":   session.deviceID,
		"flow": flow,
	}
	emitPasswordAuthorizationLog(c.logf, "info", "准备请求 Sentinel challenge",
		"method", "POST",
		"api", "/backend-api/sentinel/req",
		"url", sentinelEndpoint,
		"payload", sanitizePasswordAuthorizationPayload(body),
	)
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sentinelEndpoint, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "text/plain;charset=UTF-8")
	req.Header.Set("referer", "https://sentinel.openai.com/backend-api/sentinel/frame.html")
	req.Header.Set("origin", "https://sentinel.openai.com")
	req.Header.Set("user-agent", session.userAgent)
	req.Header.Set("sec-ch-ua", session.secCHUA)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	resp, err := c.client.Do(req)
	if err != nil {
		emitPasswordAuthorizationLog(c.logf, "error", "Sentinel challenge 请求失败",
			"method", "POST",
			"api", "/backend-api/sentinel/req",
			"url", sentinelEndpoint,
			"error", err.Error(),
		)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		emitPasswordAuthorizationLog(c.logf, "error", "Sentinel challenge 返回异常状态",
			"method", "POST",
			"api", "/backend-api/sentinel/req",
			"url", sentinelEndpoint,
			"status", resp.StatusCode,
		)
		return "", fmt.Errorf("sentinel challenge failed: status=%d", resp.StatusCode)
	}
	var challenge sentinelChallenge
	if err := json.NewDecoder(resp.Body).Decode(&challenge); err != nil {
		return "", err
	}
	challenge.RequirementsToken = requirementsToken
	if strings.TrimSpace(challenge.Token) == "" {
		return "", fmt.Errorf("sentinel challenge token is empty")
	}
	if challenge.Proofofwork.Required && strings.TrimSpace(challenge.Proofofwork.Seed) == "" {
		return "", fmt.Errorf("sentinel challenge seed is empty")
	}
	pToken := generateProofToken(session.deviceID, session.userAgent, challenge.Proofofwork.Seed, challenge.Proofofwork.Difficulty, challenge.Proofofwork.Required)
	tToken := generateTurnstileToken(session.deviceID, session.userAgent, challenge.Turnstile.DX, requirementsToken)
	payload := map[string]any{
		"p":    pToken,
		"t":    tToken,
		"c":    challenge.Token,
		"id":   session.deviceID,
		"flow": flow,
	}
	encoded, _ := json.Marshal(payload)
	emitPasswordAuthorizationLog(c.logf, "info", "Sentinel token 已生成",
		"method", "POST",
		"api", "/backend-api/sentinel/req",
		"url", sentinelEndpoint,
		"status", resp.StatusCode,
		"response", sanitizePasswordAuthorizationPayload(challenge),
		"result", sanitizePasswordAuthorizationPayload(payload),
	)
	return string(encoded), nil
}

func startAuthorization(ctx context.Context, session *passwordAuthorizationSession, authURL string) error {
	const maxAttempts = 4
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			if err := waitAuthorizeRetry(ctx, attempt); err != nil {
				return err
			}
			if err := session.resetHTTPState(); err != nil {
				return err
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
		if err != nil {
			return err
		}
		req.Header = session.navigateHeaders("")
		emitPasswordAuthorizationLog(session.logf, "info", "准备请求 OAuth authorize",
			"method", "GET",
			"api", "/oauth/authorize",
			"url", authURL,
			"attempt", attempt,
			"max_attempts", maxAttempts,
		)
		resp, err := session.do(req, true)
		if err != nil {
			lastErr = err
			emitPasswordAuthorizationLog(session.logf, "error", "调用接口 /oauth/authorize 失败",
				"method", "GET",
				"api", "/oauth/authorize",
				"url", authURL,
				"attempt", attempt,
				"error", err.Error(),
			)
			if isRetryableConnectionError(err) && attempt < maxAttempts {
				continue
			}
			if attempt < maxAttempts {
				continue
			}
			return err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < maxAttempts {
				continue
			}
			return readErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("oauth authorize failed: status=%d diagnostics=%s body=%s", resp.StatusCode, blockedResponseDiagnostics(resp, body), trimBody(body))
			emitPasswordAuthorizationLog(session.logf, "error", "接口 /oauth/authorize 调用失败",
				"method", "GET",
				"api", "/oauth/authorize",
				"url", authURL,
				"status", resp.StatusCode,
				"final_url", finalResponseURL(authURL, resp),
				"diagnostics", blockedResponseDiagnostics(resp, body),
				"body", trimBody(body),
				"attempt", attempt,
			)
			if isChallengeBlocked(resp, body) && attempt < maxAttempts {
				continue
			}
			if attempt < maxAttempts {
				continue
			}
			return lastErr
		}
		if cookieValue(session, "login_session") == "" {
			lastErr = fmt.Errorf("oauth authorize did not create login_session cookie")
			emitPasswordAuthorizationLog(session.logf, "error", "接口 /oauth/authorize 调用后未生成 login_session",
				"method", "GET",
				"api", "/oauth/authorize",
				"url", authURL,
				"final_url", finalResponseURL(authURL, resp),
				"attempt", attempt,
			)
			if attempt < maxAttempts {
				continue
			}
			return lastErr
		}
		emitPasswordAuthorizationLog(session.logf, "info", "接口 /oauth/authorize 调用完成",
			"method", "GET",
			"api", "/oauth/authorize",
			"url", authURL,
			"status", resp.StatusCode,
			"final_url", finalResponseURL(authURL, resp),
			"has_login_session", true,
			"attempt", attempt,
		)
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("oauth authorize failed after retries")
	}
	return lastErr
}

func warmupChatGPTSession(ctx context.Context, session *passwordAuthorizationSession) error {
	if session == nil {
		return fmt.Errorf("flow session is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, passwordAuthorizationChatGPT, nil)
	if err != nil {
		return err
	}
	req.Header = session.navigateHeaders("")
	emitPasswordAuthorizationLog(session.logf, "info", "准备访问 ChatGPT 首页以预热注册会话",
		"method", "GET",
		"url", passwordAuthorizationChatGPT,
	)
	resp, err := session.do(req, true)
	if err != nil {
		emitPasswordAuthorizationLog(session.logf, "error", "访问 ChatGPT 首页失败",
			"method", "GET",
			"url", passwordAuthorizationChatGPT,
			"error", err.Error(),
		)
		return err
	}
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("chatgpt warmup failed: status=%d body=%s", resp.StatusCode, trimBody(body))
	}
	emitPasswordAuthorizationLog(session.logf, "info", "ChatGPT 首页访问完成，已预热注册会话",
		"method", "GET",
		"url", passwordAuthorizationChatGPT,
		"status", resp.StatusCode,
		"final_url", finalResponseURL(passwordAuthorizationChatGPT, resp),
	)
	return nil
}

func postAuthorizeContinue(ctx context.Context, session *passwordAuthorizationSession, sentinel *sentinelClient, email string) (string, string, error) {
	token, err := sentinel.token(ctx, session, sentinelFlowAuthorizeContinue)
	if err != nil {
		return "", "", err
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/log-in")
	headers.Set("openai-sentinel-token", token)
	body := map[string]any{
		"username": map[string]any{
			"kind":  "email",
			"value": email,
		},
	}
	emitPasswordAuthorizationLog(session.logf, "info", "准备提交 authorize continue",
		"method", "POST",
		"api", pathAuthorizeContinue,
		"referer", passwordAuthorizationIssuer+"/log-in",
		"cookie_login_session", cookiePresence(cookieValue(session, "login_session")),
		"cookie_oai_client_auth_session", cookiePresence(cookieValue(session, "oai-client-auth-session")),
		"sentinel_token", tokenPresence(token),
		"payload", sanitizePasswordAuthorizationPayload(body),
	)
	payload, err := postJSON(ctx, session, passwordAuthorizationIssuer+pathAuthorizeContinue, headers, body, true)
	if err != nil {
		return "", "", err
	}
	continueURL := payload.ContinueURL
	pageType := payload.Page.Type
	if strings.TrimSpace(continueURL) == "" {
		continueURL = passwordAuthorizationIssuer + "/log-in/password"
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/authorize/continue 调用完成",
		"method", "POST",
		"api", pathAuthorizeContinue,
		"continue_url", continueURL,
		"page_type", pageType,
	)
	return continueURL, pageType, nil
}

func postPasswordVerify(ctx context.Context, session *passwordAuthorizationSession, sentinel *sentinelClient, password string) (string, string, error) {
	token, err := sentinel.token(ctx, session, sentinelFlowPasswordVerify)
	if err != nil {
		return "", "", err
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/log-in/password")
	headers.Set("openai-sentinel-token", token)
	requestBody := map[string]any{"password": password}
	emitPasswordAuthorizationLog(session.logf, "info", "准备提交 password verify",
		"method", "POST",
		"api", pathPasswordVerify,
		"referer", passwordAuthorizationIssuer+"/log-in/password",
		"cookie_login_session", cookiePresence(cookieValue(session, "login_session")),
		"cookie_oai_client_auth_session", cookiePresence(cookieValue(session, "oai-client-auth-session")),
		"sentinel_token", tokenPresence(token),
		"payload", sanitizePasswordAuthorizationPayload(requestBody),
	)
	payload, err := postJSON(ctx, session, passwordAuthorizationIssuer+pathPasswordVerify, headers, requestBody, true)
	if err != nil {
		return "", "", err
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/password/verify 调用完成",
		"method", "POST",
		"api", pathPasswordVerify,
		"page_type", payload.Page.Type,
		"continue_url", payload.ContinueURL,
	)
	return payload.ContinueURL, payload.Page.Type, nil
}

func postRegisterUser(ctx context.Context, session *passwordAuthorizationSession, sentinel *sentinelClient, email, password string) (string, string, error) {
	token, err := sentinel.token(ctx, session, sentinelFlowRegisterUser)
	if err != nil {
		return "", "", err
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/create-account/password")
	headers.Set("openai-sentinel-token", token)
	requestBody := map[string]any{
		"username": strings.TrimSpace(email),
		"password": password,
	}
	emitPasswordAuthorizationLog(session.logf, "info", "准备提交账号注册请求",
		"method", "POST",
		"api", pathRegisterUser,
		"referer", passwordAuthorizationIssuer+"/create-account/password",
		"sentinel_token", tokenPresence(token),
		"payload", sanitizePasswordAuthorizationPayload(requestBody),
	)
	payload, err := postJSON(ctx, session, passwordAuthorizationIssuer+pathRegisterUser, headers, requestBody, true)
	if err != nil {
		return "", "", err
	}
	continueURL := payload.ContinueURL
	pageType := payload.Page.Type
	if strings.TrimSpace(continueURL) == "" {
		continueURL = passwordAuthorizationIssuer + "/email-verification"
	}
	if strings.TrimSpace(pageType) == "" {
		pageType = passwordAuthorizationPageTypeEmailOTPVerification
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/user/register 调用完成",
		"method", "POST",
		"api", pathRegisterUser,
		"continue_url", continueURL,
		"page_type", pageType,
	)
	return continueURL, pageType, nil
}

func sendRegisterEmailOTP(ctx context.Context, session *passwordAuthorizationSession) (string, string, error) {
	emitPasswordAuthorizationLog(session.logf, "info", "准备发送注册邮箱验证码",
		"method", "GET",
		"api", pathRegisterEmailOTPSend,
		"referer", passwordAuthorizationIssuer+"/create-account/password",
	)
	resp, raw, err := navigate(ctx, session, passwordAuthorizationIssuer+pathRegisterEmailOTPSend, passwordAuthorizationIssuer+"/create-account/password", true)
	if err != nil {
		return "", "", err
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	continueURL := passwordAuthorizationIssuer + "/email-verification"
	pageType := passwordAuthorizationPageTypeEmailOTPVerification
	if resp != nil && isRedirectStatus(resp.StatusCode) {
		location := resolveLocation(passwordAuthorizationIssuer+pathRegisterEmailOTPSend, resp.Header.Get("Location"))
		if strings.TrimSpace(location) != "" {
			continueURL = location
		}
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/email-otp/send 调用完成",
		"method", "GET",
		"api", pathRegisterEmailOTPSend,
		"status", func() int {
			if resp == nil {
				return 0
			}
			return resp.StatusCode
		}(),
		"continue_url", continueURL,
		"page_type", pageType,
		"response", trimBody(raw),
	)
	return continueURL, pageType, nil
}

func postPasswordlessSendOTP(ctx context.Context, session *passwordAuthorizationSession) (string, string, error) {
	emitPasswordAuthorizationLog(session.logf, "info", "准备调用 passwordless send-otp",
		"method", "POST",
		"api", pathPasswordlessSendOTP,
		"referer", passwordAuthorizationIssuer+"/log-in/password",
		"payload", "null",
	)
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+pathPasswordlessSendOTP, session.commonHeaders(passwordAuthorizationIssuer+"/log-in/password"), nil, true)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if isRedirectStatus(resp.StatusCode) {
		redirectURL := resolveLocation(passwordAuthorizationIssuer+pathPasswordlessSendOTP, resp.Header.Get("Location"))
		emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/passwordless/send-otp 返回重定向",
			"method", "POST",
			"api", pathPasswordlessSendOTP,
			"status", resp.StatusCode,
			"location", redirectURL,
			"response", trimBody(raw),
		)
		return redirectURL, "email_otp_verification", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		emitPasswordAuthorizationLog(session.logf, "error", "接口 /api/accounts/passwordless/send-otp 调用失败",
			"method", "POST",
			"api", pathPasswordlessSendOTP,
			"status", resp.StatusCode,
			"error_code", extractAPIErrorCode(raw),
			"error_message", extractAPIErrorMessage(raw),
			"response", trimBody(raw),
		)
		return "", "", fmt.Errorf("passwordless send otp failed: status=%d body=%s", resp.StatusCode, trimBody(raw))
	}
	continueURL := passwordAuthorizationIssuer + "/email-verification"
	pageType := "email_otp_verification"
	if len(raw) > 0 {
		var payload passwordAuthorizationPage
		if err := json.Unmarshal(raw, &payload); err == nil {
			if strings.TrimSpace(payload.ContinueURL) != "" {
				continueURL = payload.ContinueURL
			}
			if strings.TrimSpace(payload.Page.Type) != "" {
				pageType = payload.Page.Type
			}
		}
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/passwordless/send-otp 调用完成",
		"method", "POST",
		"api", pathPasswordlessSendOTP,
		"status", resp.StatusCode,
		"continue_url", continueURL,
		"page_type", pageType,
		"response", trimBody(raw),
	)
	return continueURL, pageType, nil
}

func waitAndValidateEmailOTP(
	ctx context.Context,
	session *passwordAuthorizationSession,
	sentinel *sentinelClient,
	config *FreeMailOTPConfig,
	targetEmail string,
	since time.Time,
	logf func(level, message string, attrs ...any),
	stepPrefix string,
) (string, string, error) {
	if config == nil {
		return "", "", ErrPasswordAuthorizationEmailOTP
	}
	// This is now a thin composition layer:
	// 1. waitForEmailOTPCode handles polling / timeout / resend scheduling
	// 2. postValidateEmailOTP submits the resolved code
	stepLogf := func(level, step, message string, attrs ...any) {
		if logf == nil {
			return
		}
		logf(level, formatPasswordAuthorizationStep(stepPrefix, step, message), attrs...)
	}
	currentSince := since
	if currentSince.IsZero() {
		currentSince = time.Now().UTC()
	}
	code, _, err := waitForEmailOTPCode(
		ctx,
		config,
		targetEmail,
		currentSince,
		logf,
		stepPrefix,
		func(ctx context.Context) error {
			return triggerLoginEmailOTPResend(ctx, session)
		},
	)
	if err != nil {
		return "", "", err
	}
	stepLogf("info", "2", "已获取邮箱 OTP，开始提交验证码", "email", targetEmail)
	continueURL, pageType, validateErr := postValidateEmailOTP(ctx, session, sentinel, code)
	if validateErr != nil {
		stepLogf("warn", "2", "提交邮箱 OTP 失败: "+validateErr.Error(), "email", targetEmail)
		return "", "", validateErr
	}
	stepLogf("info", "2", "提交邮箱 OTP 成功，"+describeAuthPage(continueURL, pageType), "email", targetEmail)
	return continueURL, pageType, nil
}

func waitForEmailOTPCode(
	ctx context.Context,
	config *FreeMailOTPConfig,
	targetEmail string,
	since time.Time,
	logf func(level, message string, attrs ...any),
	stepPrefix string,
	onTimeoutResend func(context.Context) error,
) (string, time.Time, error) {
	if config == nil {
		return "", time.Time{}, ErrPasswordAuthorizationEmailOTP
	}
	stepLogf := func(level, step, message string, attrs ...any) {
		if logf == nil {
			return
		}
		logf(level, formatPasswordAuthorizationStep(stepPrefix, step, message), attrs...)
	}
	currentSince := since
	if currentSince.IsZero() {
		currentSince = time.Now().UTC()
	}
	interval := 3 * time.Second
	if config != nil && config.Interval > 0 {
		interval = config.Interval
	} else if config != nil && config.PollIntervalMillis > 0 {
		interval = time.Duration(config.PollIntervalMillis) * time.Millisecond
	}
	if interval <= 0 {
		interval = 3 * time.Second
	}

	maxAttempts := 5
	if config != nil && config.MaxAttempts > 0 {
		maxAttempts = config.MaxAttempts
	}

	resendAfter := 15 * time.Second
	if config != nil && config.ResendAfter > 0 {
		resendAfter = config.ResendAfter
	} else if config != nil && config.ResendAfterSeconds > 0 {
		resendAfter = time.Duration(config.ResendAfterSeconds) * time.Second
	}
	maxResendRounds := 3
	for round := 1; round <= maxResendRounds; round++ {
		startedAt := time.Now()
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			stepLogf("info", "1", fmt.Sprintf("开始轮询邮箱 OTP，第 %d 次", attempt), "email", targetEmail)
			code, receivedAt, err := pollFreemailVerificationCode(ctx, config, targetEmail, currentSince, func(level, message string, attrs ...any) {
				stepLogf(level, "1", message, attrs...)
			})
			if err == nil && strings.TrimSpace(code) != "" {
				stepLogf("info", "1", "已轮询获取到邮箱 OTP", "email", targetEmail)
				return code, receivedAt, nil
			}
			if err != nil && errors.Is(err, ctx.Err()) {
				return "", time.Time{}, err
			}
			if attempt < maxAttempts && time.Since(startedAt) < resendAfter {
				if err != nil {
					stepLogf("warn", "1", "当前轮次未获取到邮箱 OTP，继续轮询: "+err.Error(), "email", targetEmail)
				} else {
					stepLogf("info", "1", "当前轮次未获取到邮箱 OTP，继续轮询", "email", targetEmail)
				}
				timer := time.NewTimer(interval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return "", time.Time{}, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			if time.Since(startedAt) >= resendAfter {
				stepLogf("warn", "2", "轮询邮箱 OTP 超时，尝试重发验证码", "email", targetEmail)
				currentSince = time.Now().UTC()
				if onTimeoutResend != nil {
					if resendErr := onTimeoutResend(ctx); resendErr != nil {
						return "", time.Time{}, resendErr
					}
					stepLogf("info", "2", "邮箱 OTP 重发请求已提交", "email", targetEmail)
				}
				if !receivedAt.IsZero() && receivedAt.After(currentSince) {
					currentSince = receivedAt
				}
				break
			}
			return "", time.Time{}, ErrPasswordAuthorizationEmailOTP
		}
	}
	return "", time.Time{}, ErrPasswordAuthorizationEmailOTP
}

func formatPasswordAuthorizationStep(stepPrefix, step, message string) string {
	trimmedPrefix := strings.TrimSpace(stepPrefix)
	trimmedStep := strings.TrimSpace(step)
	switch {
	case trimmedPrefix == "" && trimmedStep == "":
		return message
	case trimmedPrefix == "":
		return fmt.Sprintf("步骤 %s：%s", trimmedStep, message)
	case trimmedStep == "":
		return fmt.Sprintf("%s：%s", trimmedPrefix, message)
	default:
		return fmt.Sprintf("%s.%s：%s", trimmedPrefix, trimmedStep, message)
	}
}

func extendPasswordAuthorizationStepPrefix(stepPrefix, step string) string {
	trimmedPrefix := strings.TrimSpace(stepPrefix)
	trimmedStep := strings.TrimSpace(step)
	switch {
	case trimmedPrefix == "" && trimmedStep == "":
		return ""
	case trimmedPrefix == "":
		return "步骤 " + trimmedStep
	case trimmedStep == "":
		return trimmedPrefix
	default:
		return trimmedPrefix + "." + trimmedStep
	}
}

func describeAuthPage(continueURL, pageType string) string {
	parts := make([]string, 0, 2)
	if trimmedPageType := strings.TrimSpace(pageType); trimmedPageType != "" {
		parts = append(parts, "page="+trimmedPageType)
	}
	if summarizedURL := summarizeURLForLog(continueURL); summarizedURL != "" {
		parts = append(parts, "continue="+summarizedURL)
	}
	if len(parts) == 0 {
		return "未返回页面跳转信息"
	}
	return strings.Join(parts, "，")
}

func summarizeURLForLog(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		if idx := strings.Index(trimmed, "?"); idx >= 0 {
			return trimmed[:idx] + "?..."
		}
		return trimmed
	}
	summary := parsed.Path
	if summary == "" {
		summary = "/"
	}
	if parsed.Host != "" {
		summary = parsed.Host + summary
	}
	if parsed.RawQuery != "" {
		queryKeys := make([]string, 0, len(parsed.Query()))
		for key := range parsed.Query() {
			queryKeys = append(queryKeys, key)
		}
		sort.Strings(queryKeys)
		summary += "?keys=" + strings.Join(queryKeys, ",")
	}
	return summary
}

func triggerEmailOTPResend(ctx context.Context, session *passwordAuthorizationSession) error {
	emitPasswordAuthorizationLog(session.logf, "info", "准备请求邮箱 OTP 重发",
		"method", "POST",
		"api", "/api/accounts/email-otp/resend",
		"referer", passwordAuthorizationIssuer+"/email-verification",
	)
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/email-verification")
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+"/api/accounts/email-otp/resend", headers, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("email otp resend failed: status=%d body=%s", resp.StatusCode, trimBody(raw))
	}
	if resp != nil {
		emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/email-otp/resend 调用完成",
			"method", "POST",
			"api", "/api/accounts/email-otp/resend",
			"status", resp.StatusCode,
			"final_url", finalResponseURL(passwordAuthorizationIssuer+"/api/accounts/email-otp/resend", resp),
			"response", trimBody(raw),
		)
	}
	return nil
}

func triggerLoginEmailOTPResend(ctx context.Context, session *passwordAuthorizationSession) error {
	return triggerEmailOTPResend(ctx, session)
}

func triggerRegisterEmailOTPResend(ctx context.Context, session *passwordAuthorizationSession) error {
	return triggerEmailOTPResend(ctx, session)
}

func postValidateEmailOTP(ctx context.Context, session *passwordAuthorizationSession, sentinel *sentinelClient, code string) (string, string, error) {
	token, err := sentinel.token(ctx, session, sentinelFlowEmailOTPValidate)
	if err != nil {
		return "", "", err
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/email-verification")
	headers.Set("openai-sentinel-token", token)
	requestBody := map[string]any{
		"code": strings.TrimSpace(code),
	}
	emitPasswordAuthorizationLog(session.logf, "info", "准备提交邮箱 OTP 验证",
		"method", "POST",
		"api", pathEmailOTPValidate,
		"referer", passwordAuthorizationIssuer+"/email-verification",
		"sentinel_token", tokenPresence(token),
		"payload", sanitizePasswordAuthorizationPayload(requestBody),
	)
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+pathEmailOTPValidate, headers, requestBody, true)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if isRedirectStatus(resp.StatusCode) {
		emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/email-otp/validate 返回重定向",
			"method", "POST",
			"api", pathEmailOTPValidate,
			"status", resp.StatusCode,
			"location", resolveLocation(passwordAuthorizationIssuer+pathEmailOTPValidate, resp.Header.Get("Location")),
			"response", trimBody(raw),
		)
		return resolveLocation(passwordAuthorizationIssuer+pathEmailOTPValidate, resp.Header.Get("Location")), "", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		emitPasswordAuthorizationLog(session.logf, "error", "接口 /api/accounts/email-otp/validate 调用失败",
			"method", "POST",
			"api", pathEmailOTPValidate,
			"status", resp.StatusCode,
			"error_code", extractAPIErrorCode(raw),
			"error_message", extractAPIErrorMessage(raw),
			"response", trimBody(raw),
		)
		return "", "", fmt.Errorf("email otp validate failed: status=%d body=%s", resp.StatusCode, trimBody(raw))
	}
	if len(raw) == 0 {
		return "", "", nil
	}
	var payload passwordAuthorizationPage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", err
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/email-otp/validate 调用完成",
		"method", "POST",
		"api", pathEmailOTPValidate,
		"status", resp.StatusCode,
		"page_type", payload.Page.Type,
		"continue_url", payload.ContinueURL,
		"response", trimBody(raw),
	)
	return payload.ContinueURL, payload.Page.Type, nil
}

func createAccountProfile(ctx context.Context, session *passwordAuthorizationSession, sentinel *sentinelClient) (string, string, error) {
	token, err := sentinel.token(ctx, session, sentinelFlowCreateAccount)
	if err != nil {
		return "", "", err
	}
	headers := session.commonHeaders(passwordAuthorizationIssuer + "/about-you")
	headers.Set("openai-sentinel-token", token)
	first, last, birthdate := randomProfileIdentity()
	requestBody := map[string]any{
		"name":      first + " " + last,
		"birthdate": birthdate,
	}
	emitPasswordAuthorizationLog(session.logf, "info", "准备调用接口 /api/accounts/create_account",
		"method", "POST",
		"api", pathCreateAccount,
		"payload", sanitizePasswordAuthorizationPayload(requestBody),
		"sentinel_token", tokenPresence(token),
	)
	resp, raw, err := postRawJSON(ctx, session, passwordAuthorizationIssuer+pathCreateAccount, headers, requestBody, false)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if isRedirectStatus(resp.StatusCode) {
		emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/create_account 返回重定向",
			"method", "POST",
			"api", pathCreateAccount,
			"status", resp.StatusCode,
			"location", resolveLocation(passwordAuthorizationIssuer+pathCreateAccount, resp.Header.Get("Location")),
			"response", trimBody(raw),
		)
		return resolveLocation(passwordAuthorizationIssuer+pathCreateAccount, resp.Header.Get("Location")), "", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		emitPasswordAuthorizationLog(session.logf, "error", "接口 /api/accounts/create_account 调用失败",
			"method", "POST",
			"api", pathCreateAccount,
			"status", resp.StatusCode,
			"error_code", extractAPIErrorCode(raw),
			"error_message", extractAPIErrorMessage(raw),
			"response", trimBody(raw),
		)
		return "", "", fmt.Errorf("create account failed: status=%d body=%s", resp.StatusCode, trimBody(raw))
	}
	if len(raw) == 0 {
		return "", "", nil
	}
	var payload passwordAuthorizationPage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", err
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口 /api/accounts/create_account 调用完成",
		"method", "POST",
		"api", pathCreateAccount,
		"status", resp.StatusCode,
		"page_type", payload.Page.Type,
		"continue_url", payload.ContinueURL,
		"response", trimBody(raw),
	)
	return payload.ContinueURL, payload.Page.Type, nil
}

func extractAuthorizationCode(ctx context.Context, session *passwordAuthorizationSession, continueURL string) (string, error) {
	consentURL := normalizeURL(passwordAuthorizationIssuer, continueURL)
	if containsAddPhoneMarker(consentURL) {
		return "", ErrPasswordAuthorizationAddPhone
	}
	if code := extractCodeFromURL(consentURL); code != "" {
		return code, nil
	}

	resp, body, err := navigate(ctx, session, consentURL, "", false)
	if err != nil {
		return "", err
	}
	finalURL := consentURL
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		finalURL = strings.TrimSpace(resp.Request.URL.String())
	}
	if containsAddPhoneMarker(consentURL, finalURL, string(body)) {
		return "", ErrPasswordAuthorizationAddPhone
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if isRedirectStatus(resp.StatusCode) {
		location := resolveLocation(consentURL, resp.Header.Get("Location"))
		if containsAddPhoneMarker(location) {
			return "", ErrPasswordAuthorizationAddPhone
		}
		if code := extractCodeFromURL(location); code != "" {
			return code, nil
		}
		if code := followAndExtractCode(ctx, session, location, 10); code != "" {
			return code, nil
		}
	}

	formAction, workspaceID := extractConsentForm(string(body), consentURL)
	if containsAddPhoneMarker(formAction) {
		return "", ErrPasswordAuthorizationAddPhone
	}
	if workspaceID != "" {
		headers := session.commonHeaders(consentURL)
		headers.Set("content-type", "application/x-www-form-urlencoded")
		resp, err := postForm(ctx, session, formAction, headers, url.Values{"workspace_id": {workspaceID}}, false)
		if err == nil {
			defer resp.Body.Close()
			if isRedirectStatus(resp.StatusCode) {
				location := resolveLocation(formAction, resp.Header.Get("Location"))
				if code := extractCodeFromURL(location); code != "" {
					return code, nil
				}
				if code := followAndExtractCode(ctx, session, location, 10); code != "" {
					return code, nil
				}
			}
		}
	}

	sessionData := decodeAuthSessionCookie(session)
	if len(sessionData.Workspaces) > 0 {
		workspaceID := sessionData.Workspaces[0].ID
		headers := session.commonHeaders(consentURL)
		payload, err := postJSON(ctx, session, passwordAuthorizationIssuer+"/api/accounts/workspace/select", headers, map[string]any{
			"workspace_id": workspaceID,
		}, false)
		if err == nil {
			if containsAddPhoneMarker(payload.ContinueURL, payload.Page.Type) {
				return "", ErrPasswordAuthorizationAddPhone
			}
			if code := extractCodeFromURL(payload.ContinueURL); code != "" {
				return code, nil
			}
			if len(payload.Data.Orgs) > 0 {
				orgID := payload.Data.Orgs[0].ID
				body := map[string]any{"org_id": orgID}
				if len(payload.Data.Orgs[0].Projects) > 0 && payload.Data.Orgs[0].Projects[0].ID != "" {
					body["project_id"] = payload.Data.Orgs[0].Projects[0].ID
				}
				orgPayload, orgErr := postJSON(ctx, session, passwordAuthorizationIssuer+"/api/accounts/organization/select", session.commonHeaders(normalizeURL(passwordAuthorizationIssuer, payload.ContinueURL)), body, false)
				if orgErr == nil {
					if containsAddPhoneMarker(orgPayload.ContinueURL, orgPayload.Page.Type) {
						return "", ErrPasswordAuthorizationAddPhone
					}
					if code := extractCodeFromURL(orgPayload.ContinueURL); code != "" {
						return code, nil
					}
					if code := followAndExtractCode(ctx, session, normalizeURL(passwordAuthorizationIssuer, orgPayload.ContinueURL), 10); code != "" {
						return code, nil
					}
				}
			} else if payload.ContinueURL != "" {
				if code := followAndExtractCode(ctx, session, normalizeURL(passwordAuthorizationIssuer, payload.ContinueURL), 10); code != "" {
					return code, nil
				}
			}
		}
	}

	if code := followAndExtractCode(ctx, session, consentURL, 10); code != "" {
		return code, nil
	}
	return "", fmt.Errorf("authorization code not found")
}

func navigate(ctx context.Context, session *passwordAuthorizationSession, targetURL, referer string, follow bool) (*http.Response, []byte, error) {
	const maxAttempts = 3
	var lastErr error
	var lastResp *http.Response
	var lastBody []byte

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			if err := waitRetry(ctx, attempt); err != nil {
				return nil, nil, err
			}
		}
		emitPasswordAuthorizationLog(session.logf, "info", "准备页面导航",
			"method", "GET",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"referer", referer,
			"follow", follow,
			"attempt", attempt,
			"max_attempts", maxAttempts,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return nil, nil, err
		}
		req.Header = session.navigateHeaders(referer)
		resp, err := session.do(req, follow)
		if err != nil {
			lastErr = err
			emitPasswordAuthorizationLog(session.logf, "error", "页面导航请求失败",
				"method", "GET",
				"api", extractAPIPath(targetURL),
				"url", targetURL,
				"referer", referer,
				"follow", follow,
				"attempt", attempt,
				"error", err.Error(),
			)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		if resp.StatusCode >= 200 && resp.StatusCode < 400 && shouldValidateJSONResponse(targetURL, resp, body) && !isValidJSONResponse(body) {
			resp.Body.Close()
			lastErr = fmt.Errorf("invalid json response")
			emitPasswordAuthorizationLog(session.logf, "warn", "页面导航响应不是有效 JSON，准备重试",
				"method", "GET",
				"api", extractAPIPath(targetURL),
				"url", targetURL,
				"status", resp.StatusCode,
				"final_url", finalResponseURL(targetURL, resp),
				"response", trimBody(body),
				"attempt", attempt,
			)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			lastResp = resp
			lastBody = body
			emitPasswordAuthorizationLog(session.logf, "error", "页面导航失败",
				"method", "GET",
				"api", extractAPIPath(targetURL),
				"url", targetURL,
				"referer", referer,
				"follow", follow,
				"status", resp.StatusCode,
				"final_url", finalResponseURL(targetURL, resp),
				"response", trimBody(body),
				"attempt", attempt,
			)
			break
		}
		emitPasswordAuthorizationLog(session.logf, "info", "页面导航完成",
			"method", "GET",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"referer", referer,
			"follow", follow,
			"status", resp.StatusCode,
			"final_url", finalResponseURL(targetURL, resp),
			"response", trimBody(body),
			"attempt", attempt,
		)
		return resp, body, nil
	}

	if lastErr != nil {
		return nil, nil, lastErr
	}
	return lastResp, lastBody, nil
}

func postJSON(ctx context.Context, session *passwordAuthorizationSession, targetURL string, headers http.Header, body any, follow bool) (passwordAuthorizationPage, error) {
	resp, raw, err := postRawJSON(ctx, session, targetURL, headers, body, follow)
	if err != nil {
		return passwordAuthorizationPage{}, err
	}
	defer resp.Body.Close()
	if isRedirectStatus(resp.StatusCode) {
		redirectURL := resolveLocation(targetURL, resp.Header.Get("Location"))
		emitPasswordAuthorizationLog(session.logf, "info", "接口请求返回重定向",
			"method", "POST",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"status", resp.StatusCode,
			"location", redirectURL,
			"follow", follow,
			"response", trimBody(raw),
		)
		return passwordAuthorizationPage{ContinueURL: redirectURL}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		emitPasswordAuthorizationLog(session.logf, "error", "接口请求失败：返回非成功状态码",
			"method", "POST",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"status", resp.StatusCode,
			"follow", follow,
			"error_code", extractAPIErrorCode(raw),
			"error_message", extractAPIErrorMessage(raw),
			"response", trimBody(raw),
		)
		return passwordAuthorizationPage{}, fmt.Errorf("post json failed: status=%d body=%s", resp.StatusCode, trimBody(raw))
	}
	if len(raw) == 0 {
		return passwordAuthorizationPage{}, nil
	}
	var payload passwordAuthorizationPage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return passwordAuthorizationPage{}, err
	}
	emitPasswordAuthorizationLog(session.logf, "info", "接口请求完成",
		"method", "POST",
		"api", extractAPIPath(targetURL),
		"url", targetURL,
		"status", resp.StatusCode,
		"follow", follow,
		"page_type", payload.Page.Type,
		"continue_url", payload.ContinueURL,
		"response", trimBody(raw),
	)
	return payload, nil
}

func postRawJSON(ctx context.Context, session *passwordAuthorizationSession, targetURL string, headers http.Header, body any, follow bool) (*http.Response, []byte, error) {
	const maxAttempts = 3
	var lastErr error
	var lastResp *http.Response
	var lastPayload []byte

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			if err := waitRetry(ctx, attempt); err != nil {
				return nil, nil, err
			}
		}
		raw, _ := json.Marshal(body)
		emitPasswordAuthorizationLog(session.logf, "info", "准备发起接口请求",
			"method", "POST",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"follow", follow,
			"payload", sanitizePasswordAuthorizationPayload(body),
			"attempt", attempt,
			"max_attempts", maxAttempts,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(raw))
		if err != nil {
			return nil, nil, err
		}
		req.Header = cloneHeader(headers)
		resp, err := session.do(req, follow)
		if err != nil {
			lastErr = err
			emitPasswordAuthorizationLog(session.logf, "error", "接口请求发送失败",
				"method", "POST",
				"api", extractAPIPath(targetURL),
				"url", targetURL,
				"follow", follow,
				"attempt", attempt,
				"error", err.Error(),
			)
			continue
		}
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body = io.NopCloser(bytes.NewReader(payload))
		if resp.StatusCode >= 200 && resp.StatusCode < 400 && shouldValidateJSONResponse(targetURL, resp, payload) && !isValidJSONResponse(payload) {
			resp.Body.Close()
			lastErr = fmt.Errorf("invalid json response")
			emitPasswordAuthorizationLog(session.logf, "warn", "接口响应内容不是有效 JSON，准备重试",
				"method", "POST",
				"api", extractAPIPath(targetURL),
				"url", targetURL,
				"status", resp.StatusCode,
				"follow", follow,
				"final_url", finalResponseURL(targetURL, resp),
				"response", trimBody(payload),
				"attempt", attempt,
			)
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			emitPasswordAuthorizationLog(session.logf, "info", "接口请求完成",
				"method", "POST",
				"api", extractAPIPath(targetURL),
				"url", targetURL,
				"status", resp.StatusCode,
				"follow", follow,
				"final_url", finalResponseURL(targetURL, resp),
				"response", trimBody(payload),
				"attempt", attempt,
			)
			return resp, payload, nil
		}
		lastResp = resp
		lastPayload = payload
		emitPasswordAuthorizationLog(session.logf, "warn", "接口请求返回异常状态",
			"method", "POST",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"status", resp.StatusCode,
			"follow", follow,
			"final_url", finalResponseURL(targetURL, resp),
			"response", trimBody(payload),
			"attempt", attempt,
		)
		break
	}

	if lastErr != nil {
		return nil, nil, lastErr
	}
	return lastResp, lastPayload, nil
}

func postForm(ctx context.Context, session *passwordAuthorizationSession, targetURL string, headers http.Header, form url.Values, follow bool) (*http.Response, error) {
	emitPasswordAuthorizationLog(session.logf, "info", "准备提交表单请求",
		"method", "POST",
		"api", extractAPIPath(targetURL),
		"url", targetURL,
		"follow", follow,
		"payload", sanitizePasswordAuthorizationPayload(form),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header = cloneHeader(headers)
	resp, err := session.do(req, follow)
	if err != nil {
		emitPasswordAuthorizationLog(session.logf, "error", "表单接口请求失败",
			"method", "POST",
			"api", extractAPIPath(targetURL),
			"url", targetURL,
			"follow", follow,
			"error", err.Error(),
		)
		return nil, err
	}
	emitPasswordAuthorizationLog(session.logf, "info", "表单接口请求完成",
		"method", "POST",
		"api", extractAPIPath(targetURL),
		"url", targetURL,
		"follow", follow,
		"status", resp.StatusCode,
		"final_url", finalResponseURL(targetURL, resp),
	)
	return resp, nil
}

func followAndExtractCode(ctx context.Context, session *passwordAuthorizationSession, startURL string, maxDepth int) string {
	current := normalizeURL(passwordAuthorizationIssuer, startURL)
	for i := 0; i < maxDepth && strings.TrimSpace(current) != ""; i++ {
		if code := extractCodeFromURL(current); code != "" {
			return code
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return ""
		}
		req.Header = session.navigateHeaders(current)
		resp, err := session.do(req, false)
		if err != nil {
			return ""
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if isRedirectStatus(resp.StatusCode) {
			next := resolveLocation(current, resp.Header.Get("Location"))
			if code := extractCodeFromURL(next); code != "" {
				return code
			}
			current = next
			continue
		}
		if resp.Request != nil && resp.Request.URL != nil {
			if code := extractCodeFromURL(resp.Request.URL.String()); code != "" {
				return code
			}
		}
		if code := extractCodeFromHTML(string(body)); code != "" {
			return code
		}
		break
	}
	return ""
}

func cookieValue(session *passwordAuthorizationSession, name string) string {
	for _, cookie := range session.cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

func decodeAuthSessionCookie(session *passwordAuthorizationSession) authSessionCookie {
	value := cookieValue(session, "oai-client-auth-session")
	if value == "" {
		return authSessionCookie{}
	}
	firstPart := value
	if idx := strings.Index(firstPart, "."); idx >= 0 {
		firstPart = firstPart[:idx]
	}
	decoded, err := base64.RawURLEncoding.DecodeString(firstPart)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(firstPart)
		if err != nil {
			return authSessionCookie{}
		}
	}
	var payload authSessionCookie
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return authSessionCookie{}
	}
	return payload
}

func cloneHeader(header http.Header) http.Header {
	if header == nil {
		return http.Header{}
	}
	cloned := http.Header{}
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func pollFreemailVerificationCode(
	ctx context.Context,
	config *FreeMailOTPConfig,
	targetEmail string,
	since time.Time,
	logf func(level, message string, attrs ...any),
) (string, time.Time, error) {
	client, err := newFreemailClient(config, logf)
	if err != nil {
		return "", time.Time{}, err
	}
	if err := client.ensureSession(ctx); err != nil {
		return "", time.Time{}, err
	}

	mailbox := normalizeFreemailAddress(targetEmail)
	messages, listErr := client.listMessages(ctx, mailbox)
	if listErr == nil {
		code, receivedAt := pickFreemailVerificationCode(messages, since)
		if strings.TrimSpace(code) != "" {
			logf("info", "FreeMail 命中邮箱 OTP", "email", mailbox)
			return code, receivedAt, nil
		}
		lastErr := fmt.Errorf("暂未在 FreeMail 中找到匹配验证码")
		logf("info", lastErr.Error(), "email", mailbox)
		return "", time.Time{}, lastErr
	}
	logf("warn", "FreeMail 轮询失败: "+listErr.Error(), "email", mailbox)
	return "", time.Time{}, listErr
}

func newFreemailClient(config *FreeMailOTPConfig, logf func(level, message string, attrs ...any)) (*freemailClient, error) {
	if config == nil {
		return nil, fmt.Errorf("freemail config is required")
	}
	baseURL := normalizeFreemailBaseURL(config.BaseURL)
	if baseURL == "" {
		baseURL = defaultFreemailBaseURL
	}
	if strings.TrimSpace(config.Username) == "" || strings.TrimSpace(config.Password) == "" {
		return nil, fmt.Errorf("freemail username or password is empty")
	}
	jar, err := stdcookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &freemailClient{
		baseURL:  baseURL,
		username: strings.TrimSpace(config.Username),
		password: config.Password,
		client: &stdhttp.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
		logf: logf,
	}, nil
}

func (c *freemailClient) ensureSession(ctx context.Context) error {
	session, err := c.requestJSON(ctx, stdhttp.MethodGet, "/api/session", nil, nil)
	if err == nil && isFreemailAuthenticated(session) {
		return nil
	}
	if _, loginErr := c.requestJSON(ctx, stdhttp.MethodPost, "/api/login", map[string]any{
		"username": c.username,
		"password": c.password,
	}, nil); loginErr != nil {
		return loginErr
	}
	session, err = c.requestJSON(ctx, stdhttp.MethodGet, "/api/session", nil, nil)
	if err != nil {
		return err
	}
	if !isFreemailAuthenticated(session) {
		return fmt.Errorf("freemail login succeeded but session not established")
	}
	return nil
}

func (c *freemailClient) listMessages(ctx context.Context, mailbox string) ([]freemailMessage, error) {
	payload, err := c.requestJSON(ctx, stdhttp.MethodGet, "/api/emails", nil, map[string]string{
		"mailbox": mailbox,
	})
	if err != nil {
		return nil, err
	}
	rows := extractFreemailRows(payload)
	messages := make([]freemailMessage, 0, len(rows))
	for _, row := range rows {
		message := normalizeFreemailMessage(row, mailbox)
		if message.ID == "" && message.Subject == "" && message.Sender == "" && message.BodyPreview == "" && message.VerificationCode == "" {
			continue
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (c *freemailClient) requestJSON(ctx context.Context, method, path string, body any, searchParams map[string]string) (any, error) {
	targetURL := joinFreemailURL(c.baseURL, path)
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	for key, value := range searchParams {
		if strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()

	var reader io.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return nil, marshalErr
		}
		reader = bytes.NewReader(raw)
	}
	req, err := stdhttp.NewRequestWithContext(ctx, method, parsed.String(), reader)
	if err != nil {
		return nil, err
	}
	emitPasswordAuthorizationLog(c.logf, "info", "准备请求 FreeMail 接口",
		"method", method,
		"api", path,
		"url", parsed.String(),
		"payload", sanitizePasswordAuthorizationPayload(body),
		"query", sanitizePasswordAuthorizationPayload(searchParams),
	)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		emitPasswordAuthorizationLog(c.logf, "error", "FreeMail 请求失败",
			"method", method,
			"api", path,
			"url", parsed.String(),
			"error", err.Error(),
		)
		return nil, fmt.Errorf("freemail request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("freemail returned invalid json: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		emitPasswordAuthorizationLog(c.logf, "error", "FreeMail 请求返回异常状态",
			"method", method,
			"api", path,
			"url", parsed.String(),
			"status", resp.StatusCode,
			"response", trimBody(raw),
		)
		return nil, fmt.Errorf("freemail request failed: %s", extractFreemailError(payload, raw, resp.StatusCode))
	}
	emitPasswordAuthorizationLog(c.logf, "info", "FreeMail 请求完成",
		"method", method,
		"api", path,
		"url", parsed.String(),
		"status", resp.StatusCode,
		"response", trimBody(raw),
	)
	return payload, nil
}

func isFreemailAuthenticated(payload any) bool {
	if data, ok := payload.(map[string]any); ok {
		auth, _ := data["authenticated"].(bool)
		return auth
	}
	return false
}

func extractFreemailRows(payload any) []map[string]any {
	switch typed := payload.(type) {
	case []any:
		return anySliceToMapRows(typed)
	case map[string]any:
		if rows, ok := typed["data"].([]any); ok {
			return anySliceToMapRows(rows)
		}
	}
	return nil
}

func anySliceToMapRows(rows []any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if mapped, ok := row.(map[string]any); ok {
			out = append(out, mapped)
		}
	}
	return out
}

func extractFreemailError(payload any, raw []byte, status int) string {
	if data, ok := payload.(map[string]any); ok {
		for _, key := range []string{"error", "message"} {
			if value := strings.TrimSpace(fmt.Sprint(data[key])); value != "" && value != "<nil>" {
				return value
			}
		}
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("HTTP %d", status)
}

func normalizeFreemailMessage(row map[string]any, mailbox string) freemailMessage {
	subject := strings.TrimSpace(stringValue(row["subject"]))
	sender := strings.TrimSpace(stringValue(row["sender"]))
	preview := firstNonEmptyString(
		stringValue(row["preview"]),
		stringValue(row["body"]),
		stringValue(row["text"]),
	)
	received := normalizeFreemailReceivedDateTime(firstNonEmptyString(
		stringValue(row["received_at"]),
		stringValue(row["receivedDateTime"]),
	))
	verificationCode := firstNonEmptyString(
		stringValue(row["verification_code"]),
		stringValue(row["verificationCode"]),
	)
	return freemailMessage{
		ID:               firstNonEmptyString(stringValue(row["id"])),
		Subject:          subject,
		Sender:           sender,
		BodyPreview:      preview,
		Raw:              preview,
		ReceivedDateTime: received,
		VerificationCode: verificationCode,
	}
}

func normalizeFreemailReceivedDateTime(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return ""
	}
	if matches := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2}(?::\d{2}(?:\.\d{1,3})?)?)$`).FindStringSubmatch(normalized); len(matches) == 3 {
		if timestamp, err := time.Parse(time.RFC3339, matches[1]+"T"+matches[2]+"Z"); err == nil {
			return timestamp.UTC().Format(time.RFC3339)
		}
	}
	isoCandidate := normalized
	if !strings.Contains(isoCandidate, "T") && strings.Contains(isoCandidate, " ") {
		isoCandidate = strings.Replace(isoCandidate, " ", "T", 1)
	}
	if timestamp, err := time.Parse(time.RFC3339, isoCandidate); err == nil {
		return timestamp.UTC().Format(time.RFC3339)
	}
	if timestamp, err := time.Parse(time.RFC3339Nano, isoCandidate); err == nil {
		return timestamp.UTC().Format(time.RFC3339)
	}
	return normalized
}

func pickFreemailVerificationCode(messages []freemailMessage, since time.Time) (string, time.Time) {
	if len(messages) == 0 {
		return "", time.Time{}
	}
	var strict []freemailCodeCandidate
	var fallback []freemailCodeCandidate
	for _, message := range messages {
		code := strings.TrimSpace(message.VerificationCode)
		if code == "" {
			code = extractOTPCode(firstNonEmptyString(message.Subject, message.BodyPreview, message.Raw))
		}
		if code == "" {
			continue
		}
		receivedAt := parseFreemailMessageTime(message.ReceivedDateTime)
		score := 0
		loweredSubject := strings.ToLower(message.Subject)
		loweredSender := strings.ToLower(message.Sender)
		for _, token := range []string{"openai", "chatgpt", "noreply", "verify", "auth"} {
			if strings.Contains(loweredSender, token) {
				score += 2
			}
			if strings.Contains(loweredSubject, token) {
				score++
			}
		}
		c := freemailCodeCandidate{code: code, receivedAt: receivedAt, score: score}
		fallback = append(fallback, c)
		if since.IsZero() || receivedAt.IsZero() || !receivedAt.Before(since) {
			strict = append(strict, c)
		}
	}
	best := pickBestFreemailCandidate(strict)
	if best.code != "" {
		return best.code, best.receivedAt
	}
	best = pickBestFreemailCandidate(fallback)
	return best.code, best.receivedAt
}

func pickBestFreemailCandidate(candidates []freemailCodeCandidate) freemailCodeCandidate {
	best := freemailCodeCandidate{}
	for _, candidate := range candidates {
		if best.code == "" ||
			candidate.score > best.score ||
			(candidate.score == best.score && candidate.receivedAt.After(best.receivedAt)) {
			best = candidate
		}
	}
	return best
}

func parseFreemailMessageTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func extractOTPCode(text string) string {
	source := strings.TrimSpace(text)
	if source == "" {
		return ""
	}
	if match := regexp.MustCompile(`(?i)(?:code|otp|验证码)[^0-9]{0,12}([0-9]{4,8})`).FindStringSubmatch(source); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	if match := regexp.MustCompile(`\b([0-9]{6})\b`).FindStringSubmatch(source); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func normalizeFreemailBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	candidate := value
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z\d+\-.]*://`).MatchString(candidate) {
		candidate = "https://" + candidate
	}
	parsed, err := url.Parse(candidate)
	if err != nil {
		return ""
	}
	parsed.Fragment = ""
	parsed.RawQuery = ""
	pathname := parsed.Path
	if pathname == "/" {
		pathname = ""
	} else {
		pathname = strings.TrimRight(pathname, "/")
	}
	return parsed.Scheme + "://" + parsed.Host + pathname
}

func joinFreemailURL(baseURL, path string) string {
	normalizedBaseURL := normalizeFreemailBaseURL(baseURL)
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultFreemailBaseURL
	}
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return normalizedBaseURL
	}
	if strings.HasPrefix(normalizedPath, "/") {
		return normalizedBaseURL + normalizedPath
	}
	return normalizedBaseURL + "/" + normalizedPath
}

func normalizeFreemailAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized != "" {
			return normalized
		}
	}
	return ""
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	normalized := strings.TrimSpace(fmt.Sprint(value))
	if normalized == "<nil>" {
		return ""
	}
	return normalized
}

func isEmailOTPPage(continueURL, pageType string) bool {
	if normalizePasswordAuthorizationPageType(pageType) == passwordAuthorizationPageTypeEmailOTPVerification {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(continueURL)), "email-verification")
}

func isPhoneOTPPage(continueURL, pageType string) bool {
	normalizedType := normalizePasswordAuthorizationPageType(pageType)
	if normalizedType == passwordAuthorizationPageTypePhoneOTPVerification {
		return true
	}
	return containsAddPhoneMarker(continueURL, pageType) ||
		strings.Contains(strings.ToLower(strings.TrimSpace(continueURL)), "phone-verification")
}

func isAboutYouPage(continueURL, pageType string) bool {
	if normalizePasswordAuthorizationPageType(pageType) == passwordAuthorizationPageTypeAboutYou {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(continueURL)), "about-you")
}

func normalizeURL(baseURL, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(value, "/")
}

func resolveLocation(baseURL, location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return location
	}
	parsedLocation, err := url.Parse(location)
	if err != nil {
		return location
	}
	return parsedBase.ResolveReference(parsedLocation).String()
}

func extractCodeFromURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Query().Get("code"))
}

var (
	workspaceIDPattern = regexp.MustCompile(`(?i)<input[^>]+name=["']workspace_id["'][^>]+value=["']([^"']+)["']`)
	formActionPattern  = regexp.MustCompile(`(?i)<form[^>]+action=["']([^"']+)["']`)
	codeHTMLPattern    = regexp.MustCompile(`(?i)[?&]code=([^&"'\s<]+)`)
)

func extractConsentForm(html, defaultURL string) (string, string) {
	action := defaultURL
	if matches := formActionPattern.FindStringSubmatch(html); len(matches) > 1 {
		action = resolveLocation(defaultURL, matches[1])
	}
	workspaceID := ""
	if matches := workspaceIDPattern.FindStringSubmatch(html); len(matches) > 1 {
		workspaceID = matches[1]
	}
	return action, workspaceID
}

func extractCodeFromHTML(html string) string {
	matches := codeHTMLPattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func trimBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 240 {
		return text[:240]
	}
	return text
}

func containsAddPhoneMarker(values ...string) bool {
	for _, value := range values {
		text := strings.ToLower(strings.TrimSpace(value))
		if text == "" {
			continue
		}
		if strings.Contains(text, "add-phone") || strings.Contains(text, "add_phone") {
			return true
		}
	}
	return false
}

func generateDeviceID() (string, error) {
	raw, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16]), nil
}

func generateRequirementsToken(deviceID, userAgent string) string {
	config := buildSentinelConfig(deviceID, userAgent)
	config[3] = 1
	config[9] = rand.Intn(45) + 5
	return "gAAAAAC" + encodeJSON(config)
}

func generateProofToken(deviceID, userAgent, seed, difficulty string, required bool) string {
	if !required || strings.TrimSpace(seed) == "" {
		return generateRequirementsToken(deviceID, userAgent)
	}
	start := time.Now()
	config := buildSentinelConfig(deviceID, userAgent)
	if strings.TrimSpace(difficulty) == "" {
		difficulty = "0"
	}
	for i := 0; i < sentinelPowMaxTry; i++ {
		config[3] = i
		config[9] = int(time.Since(start).Milliseconds())
		data := encodeJSON(config)
		hash := fnv1a32Hex(seed + data)
		if len(hash) >= len(difficulty) && hash[:len(difficulty)] <= difficulty {
			return "gAAAAAB" + data + "~S"
		}
	}
	return "gAAAAAB" + sentinelPowErrorPrefix + encodeJSON(nil)
}

func generateTurnstileToken(deviceID, userAgent string, dx any, requirementsToken string) string {
	if strings.TrimSpace(requirementsToken) == "" || isEmptyValue(dx) {
		return ""
	}
	payload := map[string]any{
		"dx":                 dx,
		"requirements_token": requirementsToken,
		"device_id":          deviceID,
		"sid":                fallbackString(deviceID, fmt.Sprintf("sid-%d", time.Now().UnixNano())),
		"timestamp":          time.Now().UTC().UnixMilli(),
		"kind":               "turnstile_local",
	}
	return "gAAAAAT" + encodeJSON(payload)
}

func buildSentinelConfig(deviceID, userAgent string) []any {
	now := time.Now()
	perfNow := rand.Float64()*49000 + 1000
	timeOrigin := float64(now.UnixMilli()) - perfNow
	tzName, tzOffset := zoneLabel(now)
	dateStr := now.Format("Mon Jan 02 2006 15:04:05") + " " + tzOffset + " (" + tzName + ")"
	return []any{
		pickInt(2667, 2745, 2880, 3000, 2560, 2200, 2160),
		dateStr,
		4607680178,
		rand.Float64(),
		fallbackString(userAgent, defaultPasswordLoginUserAgent),
		pickString(
			"https://sentinel.openai.com/sentinel/20260219f9f6/sdk.js",
			"https://sentinel.openai.com/backend-api/sentinel/sdk.js",
		),
		nil,
		pickString("en-US", "zh-CN", "en"),
		"en-US",
		"en-US,en",
		rand.Float64(),
		pickString(
			"windowControlsOverlay−[object WindowControlsOverlay]",
			"scheduling−[object Scheduling]",
			"pdfViewerEnabled−true",
			"hardwareConcurrency−16",
			"deviceMemory−8",
			"maxTouchPoints−0",
			"cookieEnabled−true",
			"vendor−Google Inc.",
			"language−en-US",
			"onLine−true",
			"webdriver−false",
		),
		pickString("location", "implementation", "URL", "documentURI", "compatMode"),
		pickString("__oai_so_bm", "__oai_logHTML", "__NEXT_DATA__", "__next_f", "__oai_SSR_TTI", "__oai_SSR_HTML", "__reactEvents", "__RUNTIME_CONFIG__"),
		perfNow,
		fallbackString(deviceID, fmt.Sprintf("sid-%d", time.Now().UnixNano())),
		"",
		pickInt(4, 8, 12, 16),
		timeOrigin,
		0, 0, 0, 0, 0, 0,
	}
}

func zoneLabel(now time.Time) (string, string) {
	_, offsetSeconds := now.Zone()
	offsetHours := offsetSeconds / 3600
	offsetMinutes := (absInt(offsetSeconds) % 3600) / 60
	tzMap := map[int]string{
		8:  "中国标准时间",
		0:  "Coordinated Universal Time",
		-5: "Eastern Standard Time",
		-8: "Pacific Standard Time",
		9:  "日本標準時",
		1:  "Central European Standard Time",
	}
	label := tzMap[offsetHours]
	if label == "" {
		label = "Coordinated Universal Time"
	}
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
	}
	return label, fmt.Sprintf("GMT%s%02d%02d", sign, absInt(offsetHours), offsetMinutes)
}

func encodeJSON(value any) string {
	raw, _ := json.Marshal(value)
	return base64.StdEncoding.EncodeToString(raw)
}

func fnv1a32Hex(text string) string {
	var h uint32 = 2166136261
	for _, ch := range text {
		h ^= uint32(ch)
		h *= 16777619
	}
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return fmt.Sprintf("%08x", h)
}

func randomProfileIdentity() (string, string, string) {
	firstNames := []string{"James", "Mary", "John", "Linda", "Robert", "Sarah", "David", "Emma"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Wilson", "Taylor", "Anderson"}
	first := firstNames[rand.Intn(len(firstNames))]
	last := lastNames[rand.Intn(len(lastNames))]
	year := rand.Intn(8) + 2000
	month := rand.Intn(12) + 1
	day := rand.Intn(28) + 1
	return first, last, fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func fallbackString(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func pickString(values ...string) string {
	if len(values) == 0 {
		return ""
	}
	return values[rand.Intn(len(values))]
}

func pickInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	return values[rand.Intn(len(values))]
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []byte:
		return len(typed) == 0
	default:
		return false
	}
}

func isRedirectStatus(status int) bool {
	return status == http.StatusMovedPermanently ||
		status == http.StatusFound ||
		status == http.StatusSeeOther ||
		status == http.StatusTemporaryRedirect ||
		status == http.StatusPermanentRedirect
}

func waitAuthorizeRetry(ctx context.Context, attempt int) error {
	if attempt <= 1 {
		return nil
	}
	baseDelay := time.Duration(1<<(attempt-2)) * time.Second
	jitter := time.Duration(rand.Intn(400)) * time.Millisecond
	timer := time.NewTimer(baseDelay + jitter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func waitRetry(ctx context.Context, attempt int) error {
	if attempt <= 1 {
		return nil
	}
	baseDelay := time.Duration(1<<(attempt-2)) * time.Second
	jitter := time.Duration(rand.Intn(400)) * time.Millisecond
	timer := time.NewTimer(baseDelay + jitter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"eof",
		"connection reset",
		"connection refused",
		"timeout",
		"tls handshake",
		"network is unreachable",
		"no such host",
	}
	for _, retryableError := range retryableErrors {
		if strings.Contains(errorStr, retryableError) {
			return true
		}
	}
	return false
}

func finalResponseURL(targetURL string, resp *http.Response) string {
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String()
	}
	return targetURL
}

func shouldValidateJSONResponse(targetURL string, resp *http.Response, body []byte) bool {
	if len(bytes.TrimSpace(body)) == 0 {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/json") {
		return true
	}
	return strings.Contains(extractAPIPath(finalResponseURL(targetURL, resp)), "/api/")
}

func isValidJSONResponse(responseBody []byte) bool {
	if len(responseBody) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(string(responseBody))
	if len(trimmed) == 0 {
		return false
	}
	firstChar := trimmed[0]
	if firstChar != '{' && firstChar != '[' {
		return false
	}
	var jsonData interface{}
	return json.Unmarshal(responseBody, &jsonData) == nil
}

func extractAPIPath(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	path := strings.TrimSpace(parsed.Path)
	if path == "" {
		path = "/"
	}
	if strings.TrimSpace(parsed.RawQuery) == "" {
		return path
	}
	return path + "?" + parsed.RawQuery
}

func isChallengeBlocked(resp *http.Response, body []byte) bool {
	if resp == nil {
		return false
	}
	text := strings.ToLower(string(body))
	server := strings.ToLower(strings.TrimSpace(resp.Header.Get("Server")))
	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if strings.Contains(text, "just a moment") ||
			strings.Contains(text, "cf-browser-verification") ||
			strings.Contains(text, "attention required") ||
			strings.Contains(server, "cloudflare") ||
			resp.Header.Get("Cf-Ray") != "" {
			return true
		}
	}
	return strings.Contains(contentType, "text/html") &&
		(strings.Contains(text, "captcha") || strings.Contains(text, "verify you are human"))
}

func blockedResponseDiagnostics(resp *http.Response, body []byte) string {
	if resp == nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("server=%s", emptyFallback(resp.Header.Get("Server"), "-")),
		fmt.Sprintf("content_type=%s", emptyFallback(resp.Header.Get("Content-Type"), "-")),
		fmt.Sprintf("cf_ray=%s", emptyFallback(resp.Header.Get("Cf-Ray"), "-")),
	}
	if len(body) > 0 {
		parts = append(parts, fmt.Sprintf("body_len=%d", len(body)))
	}
	return strings.Join(parts, " ")
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func tokenPresence(value string) string {
	if strings.TrimSpace(value) == "" {
		return "缺失"
	}
	return "存在"
}

func cookiePresence(value string) string {
	return tokenPresence(value)
}

func passwordAuthorizationLog(level, message string, attrs ...any) {
	// package-level fallback kept intentionally silent; request paths should use per-session/per-client sinks.
	_ = level
	_ = message
	_ = attrs
}

func emitPasswordAuthorizationLog(logf func(level, message string, attrs ...any), level, message string, attrs ...any) {
	if logf == nil {
		passwordAuthorizationLog(level, message, attrs...)
		return
	}
	logf(level, message, attrs...)
}

func formatPasswordAuthorizationLogAttrs(attrs ...any) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(attrs)/2)
	for i := 0; i < len(attrs); i += 2 {
		key := fmt.Sprint(attrs[i])
		var value any
		if i+1 < len(attrs) {
			value = attrs[i+1]
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, sanitizePasswordAuthorizationValue(value)))
	}
	if len(parts) == 0 {
		return ""
	}
	return " | " + strings.Join(parts, " | ")
}

func sanitizePasswordAuthorizationPayload(value any) string {
	if value == nil {
		return ""
	}
	sanitized := sanitizePasswordAuthorizationAny(value)
	raw, err := json.Marshal(sanitized)
	if err == nil {
		return string(raw)
	}
	return sanitizePasswordAuthorizationValue(value)
}

func sanitizePasswordAuthorizationAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = sanitizePasswordAuthorizationField(key, item)
		}
		return out
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = sanitizePasswordAuthorizationField(key, item)
		}
		return out
	case url.Values:
		out := make(map[string]any, len(typed))
		for key, values := range typed {
			if len(values) == 1 {
				out[key] = sanitizePasswordAuthorizationField(key, values[0])
			} else {
				items := make([]any, 0, len(values))
				for _, item := range values {
					items = append(items, sanitizePasswordAuthorizationField(key, item))
				}
				out[key] = items
			}
		}
		return out
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizePasswordAuthorizationAny(item))
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizePasswordAuthorizationAny(item))
		}
		return out
	default:
		return sanitizePasswordAuthorizationValue(value)
	}
}

func sanitizePasswordAuthorizationField(key string, value any) any {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	switch {
	case strings.Contains(lowerKey, "password"),
		strings.Contains(lowerKey, "token"),
		strings.Contains(lowerKey, "secret"),
		strings.Contains(lowerKey, "authorization"),
		strings.Contains(lowerKey, "cookie"),
		strings.Contains(lowerKey, "sentinel"),
		strings.Contains(lowerKey, "code"),
		strings.Contains(lowerKey, "verifier"):
		return maskPasswordAuthorizationSecret(value)
	default:
		return sanitizePasswordAuthorizationAny(value)
	}
}

func sanitizePasswordAuthorizationValue(value any) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	if len(text) > 480 {
		return text[:480]
	}
	return text
}

func maskPasswordAuthorizationSecret(value any) string {
	text := sanitizePasswordAuthorizationValue(value)
	if text == "" {
		return ""
	}
	if len(text) <= 4 {
		return "***"
	}
	return text[:2] + "***" + text[len(text)-2:]
}

func extractAPIErrorCode(body []byte) string {
	var payload struct {
		Code  string `json:"code"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if code := strings.TrimSpace(payload.Code); code != "" {
		return code
	}
	return strings.TrimSpace(payload.Error.Code)
}

func extractAPIErrorMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if message := strings.TrimSpace(payload.Message); message != "" {
		return message
	}
	return strings.TrimSpace(payload.Error.Message)
}
