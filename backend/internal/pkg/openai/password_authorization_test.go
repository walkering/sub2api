package openai

import "testing"

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
