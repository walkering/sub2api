package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAIOAuthFreemailDomains(t *testing.T) {
	t.Run("normalizes comma separated domains", func(t *testing.T) {
		require.Equal(t,
			[]string{"webcode.team", "mail.example.com", "third.example.org"},
			ParseOpenAIOAuthFreemailDomains(" webcode.team,MAIL.EXAMPLE.com，user@third.example.org,webcode.team "),
		)
	})

	t.Run("filters blanks", func(t *testing.T) {
		require.Empty(t, ParseOpenAIOAuthFreemailDomains(" , ， ; "))
	})
}

func TestNormalizeOpenAIOAuthFreemailDomains(t *testing.T) {
	require.Equal(t,
		"webcode.team,mail.example.com",
		NormalizeOpenAIOAuthFreemailDomains(" @webcode.team ; mail.example.com,WEBCODE.TEAM "),
	)
}
