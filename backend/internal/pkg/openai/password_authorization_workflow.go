package openai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	passwordAuthorizationPageTypeEmailOTPVerification = "email_otp_verification"
	passwordAuthorizationPageTypePhoneOTPVerification = "phone_otp_verification"
	passwordAuthorizationPageTypeAboutYou             = "about_you"
	passwordAuthorizationMaxTransitions               = 8
)

type passwordAuthorizationFlowPage struct {
	ContinueURL string
	PageType    string
}

type passwordAuthorizationFlowContext struct {
	authURL             string
	email               string
	password            string
	polledEmailOTPCode  string
	passwordSubmittedAt time.Time
	workflowKind        passwordAuthorizationWorkflowKind
	baseStepPrefix      string
	session             *passwordAuthorizationSession
	sentinel            *sentinelClient
	freeMailConfig      *FreeMailOTPConfig
	phoneConfig         *PhoneOTPProviderConfig
	logf                func(level, message string, attrs ...any)
	stepLogf            func(level, step, message string, attrs ...any)
}

type passwordAuthorizationWorkflowKind string

type passwordAuthorizationPageKind string

const (
	passwordAuthorizationWorkflowLogin    passwordAuthorizationWorkflowKind = "login"
	passwordAuthorizationWorkflowRegister passwordAuthorizationWorkflowKind = "register"

	passwordAuthorizationPageKindUnknown      passwordAuthorizationPageKind = ""
	passwordAuthorizationPageKindEmailOTP     passwordAuthorizationPageKind = "email_otp"
	passwordAuthorizationPageKindPhoneOTP     passwordAuthorizationPageKind = "phone_otp"
	passwordAuthorizationPageKindAboutYou     passwordAuthorizationPageKind = "about_you"
	passwordAuthorizationPageKindReadyForCode passwordAuthorizationPageKind = "ready_for_code"
)

type passwordAuthorizationFlowStepKey string

const (
	passwordAuthorizationFlowStepAuthorize      passwordAuthorizationFlowStepKey = "authorize"
	passwordAuthorizationFlowStepAuthorizeEmail passwordAuthorizationFlowStepKey = "authorize_email"
	passwordAuthorizationFlowStepWarmupRegisterSession passwordAuthorizationFlowStepKey = "warmup_register_session"
	passwordAuthorizationFlowStepSendEmailOTP   passwordAuthorizationFlowStepKey = "send_email_otp"
	passwordAuthorizationFlowStepRegisterUser   passwordAuthorizationFlowStepKey = "register_user"
	passwordAuthorizationFlowStepRegisterSendEmailOTP passwordAuthorizationFlowStepKey = "register_send_email_otp"
	passwordAuthorizationFlowStepPollEmailOTP   passwordAuthorizationFlowStepKey = "poll_email_otp"
	passwordAuthorizationFlowStepVerifyEmailOTP passwordAuthorizationFlowStepKey = "verify_email_otp"
	passwordAuthorizationFlowStepFillProfile    passwordAuthorizationFlowStepKey = "fill_profile"
	passwordAuthorizationFlowStepVerifyPhoneOTP passwordAuthorizationFlowStepKey = "verify_phone_otp"
	passwordAuthorizationFlowStepExtractCode    passwordAuthorizationFlowStepKey = "extract_code"
)

type passwordAuthorizationFlowStepResult struct {
	Page     passwordAuthorizationFlowPage
	NextStep passwordAuthorizationFlowStepKey
}

type passwordAuthorizationFlowStepDefinition struct {
	key             passwordAuthorizationFlowStepKey
	defaultNextStep passwordAuthorizationFlowStepKey
	execute         func(context.Context, *passwordAuthorizationFlowContext, passwordAuthorizationFlowPage) (passwordAuthorizationFlowStepResult, error)
}

func newPasswordAuthorizationFlowPage(continueURL, pageType string) passwordAuthorizationFlowPage {
	return passwordAuthorizationFlowPage{
		ContinueURL: normalizeURL(passwordAuthorizationIssuer, continueURL),
		PageType:    normalizePasswordAuthorizationPageType(pageType),
	}
}

func normalizePasswordAuthorizationPageType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func passwordAuthorizationNextStepForPage(page passwordAuthorizationFlowPage) passwordAuthorizationFlowStepKey {
	switch {
	case isEmailOTPPage(page.ContinueURL, page.PageType):
		return passwordAuthorizationFlowStepVerifyEmailOTP
	case isAboutYouPage(page.ContinueURL, page.PageType):
		return passwordAuthorizationFlowStepFillProfile
	case isPhoneOTPPage(page.ContinueURL, page.PageType):
		return passwordAuthorizationFlowStepVerifyPhoneOTP
	default:
		return ""
	}
}

func passwordAuthorizationFallbackNextStepForPage(page passwordAuthorizationFlowPage) passwordAuthorizationFlowStepKey {
	if strings.TrimSpace(page.ContinueURL) != "" {
		return passwordAuthorizationFlowStepExtractCode
	}
	return ""
}

func passwordAuthorizationWorkflowAfterAuthorize(kind passwordAuthorizationWorkflowKind) passwordAuthorizationFlowStepKey {
	switch kind {
	case passwordAuthorizationWorkflowRegister:
		return passwordAuthorizationFlowStepWarmupRegisterSession
	case passwordAuthorizationWorkflowLogin:
		return passwordAuthorizationFlowStepSendEmailOTP
	default:
		return passwordAuthorizationFlowStepSendEmailOTP
	}
}

func passwordAuthorizationWorkflowStartStep(kind passwordAuthorizationWorkflowKind) passwordAuthorizationFlowStepKey {
	return passwordAuthorizationFlowStepAuthorize
}

func normalizePasswordAuthorizationWorkflowKind(value string) passwordAuthorizationWorkflowKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(passwordAuthorizationWorkflowRegister):
		return passwordAuthorizationWorkflowRegister
	default:
		return passwordAuthorizationWorkflowLogin
	}
}

func classifyPasswordAuthorizationPage(page passwordAuthorizationFlowPage) passwordAuthorizationPageKind {
	switch {
	case isEmailOTPPage(page.ContinueURL, page.PageType):
		return passwordAuthorizationPageKindEmailOTP
	case isAboutYouPage(page.ContinueURL, page.PageType):
		return passwordAuthorizationPageKindAboutYou
	case isPhoneOTPPage(page.ContinueURL, page.PageType):
		return passwordAuthorizationPageKindPhoneOTP
	case strings.TrimSpace(page.ContinueURL) != "":
		return passwordAuthorizationPageKindReadyForCode
	default:
		return passwordAuthorizationPageKindUnknown
	}
}

func detectPasswordAuthorizationWorkflow(page passwordAuthorizationFlowPage) passwordAuthorizationWorkflowKind {
	normalized := strings.ToLower(strings.TrimSpace(page.ContinueURL))
	if strings.Contains(normalized, "create-account") || strings.Contains(normalized, "register") {
		return passwordAuthorizationWorkflowRegister
	}
	return passwordAuthorizationWorkflowLogin
}

func nextPasswordAuthorizationFlowStep(
	current passwordAuthorizationFlowPage,
	result passwordAuthorizationFlowStepResult,
	currentStep passwordAuthorizationFlowStepDefinition,
) passwordAuthorizationFlowStepKey {
	if result.NextStep != "" {
		return result.NextStep
	}
	if next := passwordAuthorizationNextStepForPage(result.Page); next != "" {
		return next
	}
	if currentStep.defaultNextStep != "" {
		return currentStep.defaultNextStep
	}
	if next := passwordAuthorizationFallbackNextStepForPage(result.Page); next != "" {
		return next
	}
	if next := passwordAuthorizationNextStepForPage(current); next != "" {
		return next
	}
	if next := passwordAuthorizationFallbackNextStepForPage(current); next != "" {
		return next
	}
	return ""
}

func advancePasswordAuthorizationFlow(
	ctx context.Context,
	flow *passwordAuthorizationFlowContext,
	initialPage passwordAuthorizationFlowPage,
) (passwordAuthorizationFlowPage, error) {
	current := newPasswordAuthorizationFlowPage(initialPage.ContinueURL, initialPage.PageType)
	stepKey := passwordAuthorizationWorkflowStartStep(flow.workflowKind)
	if stepKey == "" {
		stepKey = passwordAuthorizationNextStepForPage(current)
	}
	if stepKey == "" {
		return current, nil
	}

	operations := passwordAuthorizationOperationPool()
	for transition := 0; transition < passwordAuthorizationMaxTransitions; transition++ {
		if stepKey == "" || stepKey == passwordAuthorizationFlowStepExtractCode {
			return current, nil
		}
		step, ok := operations[stepKey]
		if !ok {
			return current, fmt.Errorf("password authorization flow step %q is not defined", stepKey)
		}
		result, err := step.execute(ctx, flow, current)
		if err != nil {
			return current, err
		}
		if strings.TrimSpace(result.Page.ContinueURL) != "" || strings.TrimSpace(result.Page.PageType) != "" {
			current = newPasswordAuthorizationFlowPage(result.Page.ContinueURL, result.Page.PageType)
		}
		if step.key == passwordAuthorizationFlowStepAuthorize {
			stepKey = passwordAuthorizationWorkflowAfterAuthorize(flow.workflowKind)
			continue
		}
		stepKey = nextPasswordAuthorizationFlowStep(current, result, step)
	}
	return current, fmt.Errorf("password authorization flow exceeded max transitions")
}
