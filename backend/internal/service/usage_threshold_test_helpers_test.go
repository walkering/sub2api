package service

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type stubSessionLimitCache struct {
	SessionLimitCache
	activeCounts map[int64]int
}

func (s *stubSessionLimitCache) RegisterSession(ctx context.Context, accountID int64, sessionUUID string, maxSessions int, idleTimeout time.Duration) (bool, error) {
	return true, nil
}

func (s *stubSessionLimitCache) RefreshSession(ctx context.Context, accountID int64, sessionUUID string, idleTimeout time.Duration) error {
	return nil
}

func (s *stubSessionLimitCache) GetActiveSessionCount(ctx context.Context, accountID int64) (int, error) {
	if s.activeCounts == nil {
		return 0, nil
	}
	return s.activeCounts[accountID], nil
}

func (s *stubSessionLimitCache) GetActiveSessionCountBatch(ctx context.Context, accountIDs []int64, idleTimeouts map[int64]time.Duration) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = 0
		if s.activeCounts != nil {
			result[accountID] = s.activeCounts[accountID]
		}
	}
	return result, nil
}

func (s *stubSessionLimitCache) IsSessionActive(ctx context.Context, accountID int64, sessionUUID string) (bool, error) {
	return false, nil
}

func (s *stubSessionLimitCache) GetWindowCost(ctx context.Context, accountID int64) (float64, bool, error) {
	return 0, false, nil
}

func (s *stubSessionLimitCache) SetWindowCost(ctx context.Context, accountID int64, cost float64) error {
	return nil
}

func (s *stubSessionLimitCache) GetWindowCostBatch(ctx context.Context, accountIDs []int64) (map[int64]float64, error) {
	return map[int64]float64{}, nil
}

var _ SessionLimitCache = (*stubSessionLimitCache)(nil)

type stubGroupRepoForUsageThreshold struct {
	GroupRepository
	groups map[int64]*Group
}

func (s *stubGroupRepoForUsageThreshold) GetByID(ctx context.Context, id int64) (*Group, error) {
	return s.GetByIDLite(ctx, id)
}

func (s *stubGroupRepoForUsageThreshold) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	if s.groups == nil {
		return nil, errors.New("group not found")
	}
	group, ok := s.groups[id]
	if !ok {
		return nil, errors.New("group not found")
	}
	return group, nil
}

func (s *stubGroupRepoForUsageThreshold) Create(ctx context.Context, group *Group) error { return nil }
func (s *stubGroupRepoForUsageThreshold) Update(ctx context.Context, group *Group) error { return nil }
func (s *stubGroupRepoForUsageThreshold) Delete(ctx context.Context, id int64) error     { return nil }
func (s *stubGroupRepoForUsageThreshold) DeleteCascade(ctx context.Context, id int64) ([]int64, error) {
	return nil, nil
}
func (s *stubGroupRepoForUsageThreshold) List(ctx context.Context, params pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *stubGroupRepoForUsageThreshold) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, status, search string, isExclusive *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *stubGroupRepoForUsageThreshold) ListActive(ctx context.Context) ([]Group, error) {
	return nil, nil
}
func (s *stubGroupRepoForUsageThreshold) ListActiveByPlatform(ctx context.Context, platform string) ([]Group, error) {
	return nil, nil
}
func (s *stubGroupRepoForUsageThreshold) ExistsByName(ctx context.Context, name string) (bool, error) {
	return false, nil
}
func (s *stubGroupRepoForUsageThreshold) GetAccountCount(ctx context.Context, groupID int64) (int64, int64, error) {
	return 0, 0, nil
}
func (s *stubGroupRepoForUsageThreshold) DeleteAccountGroupsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	return 0, nil
}
func (s *stubGroupRepoForUsageThreshold) BindAccountsToGroup(ctx context.Context, groupID int64, accountIDs []int64) error {
	return nil
}
func (s *stubGroupRepoForUsageThreshold) GetAccountIDsByGroupIDs(ctx context.Context, groupIDs []int64) ([]int64, error) {
	return nil, nil
}
func (s *stubGroupRepoForUsageThreshold) UpdateSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error {
	return nil
}

var _ GroupRepository = (*stubGroupRepoForUsageThreshold)(nil)
