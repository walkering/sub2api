//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type errorAccountCleanupRepoStub struct {
	listResponses []errorAccountCleanupListResponse
	listErr       error
	deleteErrs    map[int64]error
	listCalls     int
	deleteCalls   []int64
}

type errorAccountCleanupListResponse struct {
	accounts []Account
	pager    *pagination.PaginationResult
}

func (s *errorAccountCleanupRepoStub) Create(context.Context, *Account) error {
	panic("unexpected Create call")
}

func (s *errorAccountCleanupRepoStub) GetByID(context.Context, int64) (*Account, error) {
	panic("unexpected GetByID call")
}

func (s *errorAccountCleanupRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	panic("unexpected GetByIDs call")
}

func (s *errorAccountCleanupRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	panic("unexpected ExistsByID call")
}

func (s *errorAccountCleanupRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	panic("unexpected GetByCRSAccountID call")
}

func (s *errorAccountCleanupRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	panic("unexpected FindByExtraField call")
}

func (s *errorAccountCleanupRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	panic("unexpected ListCRSAccountIDs call")
}

func (s *errorAccountCleanupRepoStub) Update(context.Context, *Account) error {
	panic("unexpected Update call")
}

func (s *errorAccountCleanupRepoStub) Delete(_ context.Context, id int64) error {
	s.deleteCalls = append(s.deleteCalls, id)
	if s.deleteErrs != nil {
		return s.deleteErrs[id]
	}
	return nil
}

func (s *errorAccountCleanupRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *errorAccountCleanupRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, _, _, _ string) ([]Account, *pagination.PaginationResult, error) {
	if s.listErr != nil {
		return nil, nil, s.listErr
	}
	if s.listCalls >= len(s.listResponses) {
		return []Account{}, &pagination.PaginationResult{Page: s.listCalls + 1, PageSize: errorAccountCleanupPageSize}, nil
	}
	resp := s.listResponses[s.listCalls]
	s.listCalls++
	return resp.accounts, resp.pager, nil
}

func (s *errorAccountCleanupRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	panic("unexpected ListByGroup call")
}

func (s *errorAccountCleanupRepoStub) ListActive(context.Context) ([]Account, error) {
	panic("unexpected ListActive call")
}

func (s *errorAccountCleanupRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected ListByPlatform call")
}

func (s *errorAccountCleanupRepoStub) UpdateLastUsed(context.Context, int64) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *errorAccountCleanupRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	panic("unexpected BatchUpdateLastUsed call")
}

func (s *errorAccountCleanupRepoStub) SetError(context.Context, int64, string) error {
	panic("unexpected SetError call")
}

func (s *errorAccountCleanupRepoStub) ClearError(context.Context, int64) error {
	panic("unexpected ClearError call")
}

func (s *errorAccountCleanupRepoStub) SetSchedulable(context.Context, int64, bool) error {
	panic("unexpected SetSchedulable call")
}

func (s *errorAccountCleanupRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	panic("unexpected AutoPauseExpiredAccounts call")
}

func (s *errorAccountCleanupRepoStub) BindGroups(_ context.Context, _ int64, _ []int64) error {
	panic("unexpected BindGroups call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	panic("unexpected ListSchedulable call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupID call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatform call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatform call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatforms call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatforms call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatform call")
}

func (s *errorAccountCleanupRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatforms call")
}

func (s *errorAccountCleanupRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	panic("unexpected SetRateLimited call")
}

func (s *errorAccountCleanupRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	panic("unexpected SetModelRateLimit call")
}

func (s *errorAccountCleanupRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	panic("unexpected SetOverloaded call")
}

func (s *errorAccountCleanupRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	panic("unexpected SetTempUnschedulable call")
}

func (s *errorAccountCleanupRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	panic("unexpected ClearTempUnschedulable call")
}

func (s *errorAccountCleanupRepoStub) ClearRateLimit(context.Context, int64) error {
	panic("unexpected ClearRateLimit call")
}

func (s *errorAccountCleanupRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	panic("unexpected ClearAntigravityQuotaScopes call")
}

func (s *errorAccountCleanupRepoStub) ClearModelRateLimits(context.Context, int64) error {
	panic("unexpected ClearModelRateLimits call")
}

func (s *errorAccountCleanupRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	panic("unexpected UpdateSessionWindow call")
}

func (s *errorAccountCleanupRepoStub) UpdateExtra(context.Context, int64, map[string]any) error {
	panic("unexpected UpdateExtra call")
}

func (s *errorAccountCleanupRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	panic("unexpected BulkUpdate call")
}

func (s *errorAccountCleanupRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *errorAccountCleanupRepoStub) ResetQuotaUsed(context.Context, int64) error {
	panic("unexpected ResetQuotaUsed call")
}

func TestErrorAccountCleanupServiceRunOnceDeletesErrorAccounts(t *testing.T) {
	repo := &errorAccountCleanupRepoStub{
		listResponses: []errorAccountCleanupListResponse{
			{
				accounts: []Account{
					{ID: 1, Status: StatusError, GroupIDs: []int64{10, 20}},
					{ID: 2, Status: StatusError},
				},
				pager: &pagination.PaginationResult{Page: 1, PageSize: errorAccountCleanupPageSize, Pages: 1, Total: 2},
			},
		},
	}

	svc := NewErrorAccountCleanupService(repo, 5*time.Minute)
	svc.runOnce()

	require.Equal(t, []int64{1, 2}, repo.deleteCalls)
}

func TestErrorAccountCleanupServiceRunOnceContinuesAfterDeleteError(t *testing.T) {
	repo := &errorAccountCleanupRepoStub{
		listResponses: []errorAccountCleanupListResponse{
			{
				accounts: []Account{
					{ID: 1, Status: StatusError, GroupIDs: []int64{10}},
					{ID: 2, Status: StatusError, GroupIDs: []int64{20}},
				},
				pager: &pagination.PaginationResult{Page: 1, PageSize: errorAccountCleanupPageSize, Pages: 1, Total: 2},
			},
		},
		deleteErrs: map[int64]error{1: errors.New("delete failed")},
	}

	svc := NewErrorAccountCleanupService(repo, 5*time.Minute)
	svc.runOnce()

	require.Equal(t, []int64{1, 2}, repo.deleteCalls)
}

func TestErrorAccountCleanupServiceRunOnceScansAllPages(t *testing.T) {
	repo := &errorAccountCleanupRepoStub{
		listResponses: []errorAccountCleanupListResponse{
			{
				accounts: []Account{
					{ID: 1, Status: StatusError, GroupIDs: []int64{10}},
				},
				pager: &pagination.PaginationResult{Page: 1, PageSize: errorAccountCleanupPageSize, Pages: 2, Total: 2},
			},
			{
				accounts: []Account{
					{ID: 2, Status: StatusError, GroupIDs: []int64{20}},
				},
				pager: &pagination.PaginationResult{Page: 2, PageSize: errorAccountCleanupPageSize, Pages: 2, Total: 2},
			},
		},
	}

	svc := NewErrorAccountCleanupService(repo, 5*time.Minute)
	svc.runOnce()

	require.Equal(t, 2, repo.listCalls)
	require.Equal(t, []int64{1, 2}, repo.deleteCalls)
}
