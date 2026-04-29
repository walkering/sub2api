package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIAutoReauthFields(t *testing.T) {
	account := &service.Account{
		Name: "fallback@example.com",
		Credentials: map[string]any{
			"email": "credential@example.com",
		},
		Extra: map[string]any{
			"email_address":         "extra@example.com",
			"password":              "secret-123",
			"openai_email_provider": "freemail",
			"openai_phone_provider": "hero-sms",
		},
	}

	require.Equal(t, "credential@example.com", extractOpenAIAutoReauthEmail(account))
	require.Equal(t, "secret-123", extractOpenAIAutoReauthPassword(account))
	require.Equal(t, "freemail", extractOpenAIAutoReauthEmailProvider(account))
	require.Equal(t, "hero-sms", extractOpenAIAutoReauthPhoneProvider(account))
}

func TestExtractEmailDomain(t *testing.T) {
	require.Equal(t, "webcode.team", extractEmailDomain("User@webcode.team"))
	require.Empty(t, extractEmailDomain("invalid-email"))
}

func TestIsOpenAIAutoReauthEmailAllowed(t *testing.T) {
	allowedDomains := buildStringSet([]string{"webcode.team", "mail.example.com"})
	require.True(t, isOpenAIAutoReauthEmailAllowed("user@webcode.team", allowedDomains))
	require.False(t, isOpenAIAutoReauthEmailAllowed("user@other.example", allowedDomains))
	require.False(t, isOpenAIAutoReauthEmailAllowed("", allowedDomains))
}

func TestExtractOpenAIAuthState(t *testing.T) {
	authURL := "https://auth.openai.com/oauth/authorize?client_id=test&state=abc123&code_challenge=xyz"
	require.Equal(t, "abc123", extractOpenAIAuthState(authURL))
	require.Empty(t, extractOpenAIAuthState("::invalid::"))
}
