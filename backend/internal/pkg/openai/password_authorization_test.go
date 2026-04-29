package openai

import (
	"testing"
	"time"
)

func TestFormatPasswordAuthorizationStep(t *testing.T) {
	t.Run("formats top level step prefix", func(t *testing.T) {
		got := formatPasswordAuthorizationStep("步骤 4", "2", "提交登录邮箱")
		want := "步骤 4.2：提交登录邮箱"
		if got != want {
			t.Fatalf("formatPasswordAuthorizationStep() = %q, want %q", got, want)
		}
	})

	t.Run("uses prefix without suffix", func(t *testing.T) {
		got := formatPasswordAuthorizationStep("步骤 4", "", "开始 OpenAI 纯 HTTP 登录")
		want := "步骤 4：开始 OpenAI 纯 HTTP 登录"
		if got != want {
			t.Fatalf("formatPasswordAuthorizationStep() = %q, want %q", got, want)
		}
	})
}

func TestExtendPasswordAuthorizationStepPrefix(t *testing.T) {
	got := extendPasswordAuthorizationStepPrefix("步骤 4", "4")
	want := "步骤 4.4"
	if got != want {
		t.Fatalf("extendPasswordAuthorizationStepPrefix() = %q, want %q", got, want)
	}
}

func TestDescribeAuthPage(t *testing.T) {
	t.Run("includes page type and sanitized url", func(t *testing.T) {
		got := describeAuthPage("https://auth.openai.com/u/mfa?state=abc&code=secret", "email_otp")
		want := "page=email_otp，continue=auth.openai.com/u/mfa?keys=code,state"
		if got != want {
			t.Fatalf("describeAuthPage() = %q, want %q", got, want)
		}
	})

	t.Run("falls back when page metadata missing", func(t *testing.T) {
		got := describeAuthPage("", "")
		want := "未返回页面跳转信息"
		if got != want {
			t.Fatalf("describeAuthPage() = %q, want %q", got, want)
		}
	})
}

func TestSummarizeURLForLog(t *testing.T) {
	t.Run("hides query values and keeps keys", func(t *testing.T) {
		got := summarizeURLForLog("/authorize?client_id=test&redirect_uri=https://example.com/callback")
		want := "/authorize?keys=client_id,redirect_uri"
		if got != want {
			t.Fatalf("summarizeURLForLog() = %q, want %q", got, want)
		}
	})

	t.Run("keeps plain paths", func(t *testing.T) {
		got := summarizeURLForLog("/u/consent")
		want := "/u/consent"
		if got != want {
			t.Fatalf("summarizeURLForLog() = %q, want %q", got, want)
		}
	})
}

func TestNormalizePasswordAuthorizationPageType(t *testing.T) {
	got := normalizePasswordAuthorizationPageType(" Phone-OTP_Verification ")
	want := "phone_otp_verification"
	if got != want {
		t.Fatalf("normalizePasswordAuthorizationPageType() = %q, want %q", got, want)
	}
}

func TestPasswordAuthorizationPageClassification(t *testing.T) {
	t.Run("prefers explicit page type for phone otp", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("https://auth.openai.com/phone-verification", "phone_otp_verification")
		if got := classifyPasswordAuthorizationPage(page); got != passwordAuthorizationPageKindPhoneOTP {
			t.Fatalf("classifyPasswordAuthorizationPage() = %q, want %q", got, passwordAuthorizationPageKindPhoneOTP)
		}
	})

	t.Run("detects about-you by page type", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("/next", "about_you")
		if got := classifyPasswordAuthorizationPage(page); got != passwordAuthorizationPageKindAboutYou {
			t.Fatalf("classifyPasswordAuthorizationPage() = %q, want %q", got, passwordAuthorizationPageKindAboutYou)
		}
	})

	t.Run("detects email verification by explicit page type", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("https://auth.openai.com/email-verification", "email_otp_verification")
		if got := classifyPasswordAuthorizationPage(page); got != passwordAuthorizationPageKindEmailOTP {
			t.Fatalf("classifyPasswordAuthorizationPage() = %q, want %q", got, passwordAuthorizationPageKindEmailOTP)
		}
	})

	t.Run("falls back to ready for code when continue url exists", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("https://auth.openai.com/authorize/resume?state=abc", "")
		if got := classifyPasswordAuthorizationPage(page); got != passwordAuthorizationPageKindReadyForCode {
			t.Fatalf("classifyPasswordAuthorizationPage() = %q, want %q", got, passwordAuthorizationPageKindReadyForCode)
		}
	})
}

func TestNextPasswordAuthorizationFlowStep(t *testing.T) {
	t.Run("prefers next step resolved from page type", func(t *testing.T) {
		currentStep := passwordAuthorizationFlowStepDefinition{
			key:             passwordAuthorizationFlowStepVerifyEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepFillProfile,
		}
		result := passwordAuthorizationFlowStepResult{
			Page: newPasswordAuthorizationFlowPage("https://auth.openai.com/phone-verification", "phone_otp_verification"),
		}
		got := nextPasswordAuthorizationFlowStep(passwordAuthorizationFlowPage{}, result, currentStep)
		want := passwordAuthorizationFlowStepVerifyPhoneOTP
		if got != want {
			t.Fatalf("nextPasswordAuthorizationFlowStep() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to default next step when page type is unknown", func(t *testing.T) {
		currentStep := passwordAuthorizationFlowStepDefinition{
			key:             passwordAuthorizationFlowStepVerifyEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepFillProfile,
		}
		result := passwordAuthorizationFlowStepResult{
			Page: newPasswordAuthorizationFlowPage("https://auth.openai.com/unknown-next", "mystery_page"),
		}
		got := nextPasswordAuthorizationFlowStep(passwordAuthorizationFlowPage{}, result, currentStep)
		want := passwordAuthorizationFlowStepFillProfile
		if got != want {
			t.Fatalf("nextPasswordAuthorizationFlowStep() = %q, want %q", got, want)
		}
	})

	t.Run("register defaults to register email otp send before verification", func(t *testing.T) {
		currentStep := passwordAuthorizationFlowStepDefinition{
			key:             passwordAuthorizationFlowStepRegisterUser,
			defaultNextStep: passwordAuthorizationFlowStepRegisterSendEmailOTP,
		}
		result := passwordAuthorizationFlowStepResult{
			Page: passwordAuthorizationFlowPage{},
		}
		got := nextPasswordAuthorizationFlowStep(passwordAuthorizationFlowPage{}, result, currentStep)
		want := passwordAuthorizationFlowStepRegisterSendEmailOTP
		if got != want {
			t.Fatalf("nextPasswordAuthorizationFlowStep() = %q, want %q", got, want)
		}
	})

	t.Run("send email otp defaults to polling email otp", func(t *testing.T) {
		currentStep := passwordAuthorizationFlowStepDefinition{
			key:             passwordAuthorizationFlowStepSendEmailOTP,
			defaultNextStep: passwordAuthorizationFlowStepPollEmailOTP,
		}
		result := passwordAuthorizationFlowStepResult{
			Page: passwordAuthorizationFlowPage{},
		}
		got := nextPasswordAuthorizationFlowStep(passwordAuthorizationFlowPage{}, result, currentStep)
		want := passwordAuthorizationFlowStepPollEmailOTP
		if got != want {
			t.Fatalf("nextPasswordAuthorizationFlowStep() = %q, want %q", got, want)
		}
	})
}

func TestPasswordAuthorizationWorkflowDetection(t *testing.T) {
	t.Run("detects register workflow from create-account url", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("https://auth.openai.com/create-account/password", "")
		if got := detectPasswordAuthorizationWorkflow(page); got != passwordAuthorizationWorkflowRegister {
			t.Fatalf("detectPasswordAuthorizationWorkflow() = %q, want %q", got, passwordAuthorizationWorkflowRegister)
		}
	})

	t.Run("defaults to login workflow", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("https://auth.openai.com/log-in/password", "")
		if got := detectPasswordAuthorizationWorkflow(page); got != passwordAuthorizationWorkflowLogin {
			t.Fatalf("detectPasswordAuthorizationWorkflow() = %q, want %q", got, passwordAuthorizationWorkflowLogin)
		}
	})

	t.Run("detects register workflow from authorize continue result", func(t *testing.T) {
		page := newPasswordAuthorizationFlowPage("https://auth.openai.com/create-account/password?from=authorize_continue", "")
		if got := detectPasswordAuthorizationWorkflow(page); got != passwordAuthorizationWorkflowRegister {
			t.Fatalf("detectPasswordAuthorizationWorkflow() = %q, want %q", got, passwordAuthorizationWorkflowRegister)
		}
	})
}

func TestPasswordAuthorizationWorkflowStartStep(t *testing.T) {
	if got := passwordAuthorizationWorkflowStartStep(passwordAuthorizationWorkflowLogin); got != passwordAuthorizationFlowStepAuthorize {
		t.Fatalf("passwordAuthorizationWorkflowStartStep(login) = %q, want %q", got, passwordAuthorizationFlowStepAuthorize)
	}
	if got := passwordAuthorizationWorkflowStartStep(passwordAuthorizationWorkflowRegister); got != passwordAuthorizationFlowStepAuthorize {
		t.Fatalf("passwordAuthorizationWorkflowStartStep(register) = %q, want %q", got, passwordAuthorizationFlowStepAuthorize)
	}
}

func TestPasswordAuthorizationWorkflowAfterAuthorize(t *testing.T) {
	if got := passwordAuthorizationWorkflowAfterAuthorize(passwordAuthorizationWorkflowLogin); got != passwordAuthorizationFlowStepSendEmailOTP {
		t.Fatalf("passwordAuthorizationWorkflowAfterAuthorize(login) = %q, want %q", got, passwordAuthorizationFlowStepSendEmailOTP)
	}
	if got := passwordAuthorizationWorkflowAfterAuthorize(passwordAuthorizationWorkflowRegister); got != passwordAuthorizationFlowStepWarmupRegisterSession {
		t.Fatalf("passwordAuthorizationWorkflowAfterAuthorize(register) = %q, want %q", got, passwordAuthorizationFlowStepWarmupRegisterSession)
	}
}

func TestAuthorizeEmailWorkflowSwitchUsesDetectedPage(t *testing.T) {
	page := newPasswordAuthorizationFlowPage("https://auth.openai.com/create-account/password?from=authorize_continue", "")
	workflow := detectPasswordAuthorizationWorkflow(page)
	if workflow != passwordAuthorizationWorkflowRegister {
		t.Fatalf("workflow = %q, want %q", workflow, passwordAuthorizationWorkflowRegister)
	}
	next := passwordAuthorizationWorkflowAfterAuthorize(workflow)
	if next != passwordAuthorizationFlowStepWarmupRegisterSession {
		t.Fatalf("next step = %q, want %q", next, passwordAuthorizationFlowStepWarmupRegisterSession)
	}
}

func TestEmailOTPConfigTimingFields(t *testing.T) {
	cfg := &FreeMailOTPConfig{
		PollIntervalMillis: 3000,
		ResendAfterSeconds: 15,
		MaxAttempts:        5,
		Interval:           3 * time.Second,
		ResendAfter:        15 * time.Second,
	}

	if cfg.Interval != 3*time.Second {
		t.Fatalf("Interval = %v, want %v", cfg.Interval, 3*time.Second)
	}
	if cfg.ResendAfter != 15*time.Second {
		t.Fatalf("ResendAfter = %v, want %v", cfg.ResendAfter, 15*time.Second)
	}
	if cfg.MaxAttempts != 5 {
		t.Fatalf("MaxAttempts = %d, want %d", cfg.MaxAttempts, 5)
	}
}
