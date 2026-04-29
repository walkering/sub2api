package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestIsTokenExpiredAccount(t *testing.T) {
	t.Run("matches token_expired error payload", func(t *testing.T) {
		account := &service.Account{
			Status:       service.StatusError,
			ErrorMessage: `API returned 401: {"error":{"code":"token_expired","message":"Provided authentication token is expired."}}`,
		}
		require.True(t, isTokenExpiredAccount(account))
	})

	t.Run("requires error status", func(t *testing.T) {
		account := &service.Account{
			Status:       service.StatusActive,
			ErrorMessage: `token_expired`,
		}
		require.False(t, isTokenExpiredAccount(account))
	})
}

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

func TestExtractOpenAIAuthState(t *testing.T) {
	authURL := "https://auth.openai.com/oauth/authorize?client_id=test&state=abc123&code_challenge=xyz"
	require.Equal(t, "abc123", extractOpenAIAuthState(authURL))
	require.Empty(t, extractOpenAIAuthState("::invalid::"))
}
