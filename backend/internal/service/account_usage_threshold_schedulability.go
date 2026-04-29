package service

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

func resolveUsageThresholdGroup(ctx context.Context, groupRepo GroupRepository, groupID int64) (*Group, error) {
	if groupID <= 0 {
		return nil, nil
	}
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(group) && group.ID == groupID {
		return group, nil
	}
	if groupRepo == nil {
		return nil, nil
	}

	group, err := groupRepo.GetByIDLite(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group failed: %w", err)
	}
	return group, nil
}

func checkAccountUsageThresholdSchedulability(
	ctx context.Context,
	groupRepo GroupRepository,
	sessionLimitCache SessionLimitCache,
	groupID *int64,
	account *Account,
) WindowCostSchedulability {
	if account == nil || groupID == nil || *groupID <= 0 {
		return WindowCostSchedulable
	}

	group, err := resolveUsageThresholdGroup(ctx, groupRepo, *groupID)
	if err != nil || group == nil || group.AccountUsageThresholdPercent == nil || *group.AccountUsageThresholdPercent <= 0 {
		return WindowCostSchedulable
	}

	activeSessions := 0
	if sessionLimitCache != nil {
		count, err := sessionLimitCache.GetActiveSessionCount(ctx, account.ID)
		if err != nil {
			return WindowCostSchedulable
		}
		activeSessions = count
	}

	return account.CheckQuotaUsageThresholdSchedulability(*group.AccountUsageThresholdPercent, activeSessions)
}

func isAccountAllowedByUsageThreshold(
	ctx context.Context,
	groupRepo GroupRepository,
	sessionLimitCache SessionLimitCache,
	groupID *int64,
	account *Account,
	isSticky bool,
) bool {
	switch checkAccountUsageThresholdSchedulability(ctx, groupRepo, sessionLimitCache, groupID, account) {
	case WindowCostSchedulable:
		return true
	case WindowCostStickyOnly:
		return isSticky
	case WindowCostNotSchedulable:
		return false
	default:
		return true
	}
}
