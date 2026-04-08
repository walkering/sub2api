package handler

import (
	"context"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

func clearOpenAIStickySessionOnRateLimitFailover(
	ctx context.Context,
	reqLog *zap.Logger,
	gatewayService *service.OpenAIGatewayService,
	groupID *int64,
	sessionHash string,
	account *service.Account,
	failoverErr *service.UpstreamFailoverError,
) {
	if gatewayService == nil || failoverErr == nil || failoverErr.StatusCode != http.StatusTooManyRequests || sessionHash == "" {
		return
	}
	if err := gatewayService.ClearStickySession(ctx, groupID, sessionHash); err != nil {
		if reqLog != nil {
			reqLog.Warn("openai.rate_limit_clear_sticky_failed",
				zap.Int64("account_id", openAIAccountIDForLog(account)),
				zap.Error(err),
			)
		}
		return
	}
	if reqLog != nil {
		reqLog.Warn("openai.rate_limit_sticky_cleared",
			zap.Int64("account_id", openAIAccountIDForLog(account)),
		)
	}
}

func openAIAccountIDForLog(account *service.Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}
