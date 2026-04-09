package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type scheduledTestAccountRepoStub struct {
	exists             bool
	existsErr          error
	schedulableByGroup map[int64][]Account
}

func (s *scheduledTestAccountRepoStub) Create(context.Context, *Account) error { return nil }
func (s *scheduledTestAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return nil, ErrAccountNotFound
}
func (s *scheduledTestAccountRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	return s.exists, s.existsErr
}
func (s *scheduledTestAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) Update(context.Context, *Account) error { return nil }
func (s *scheduledTestAccountRepoStub) Delete(context.Context, int64) error    { return nil }
func (s *scheduledTestAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *scheduledTestAccountRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string, string, string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *scheduledTestAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListActive(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (s *scheduledTestAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) SetError(context.Context, int64, string) error     { return nil }
func (s *scheduledTestAccountRepoStub) ClearError(context.Context, int64) error           { return nil }
func (s *scheduledTestAccountRepoStub) SetSchedulable(context.Context, int64, bool) error { return nil }
func (s *scheduledTestAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (s *scheduledTestAccountRepoStub) BindGroups(context.Context, int64, []int64) error { return nil }
func (s *scheduledTestAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	return append([]Account(nil), s.schedulableByGroup[groupID]...), nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *scheduledTestAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) ClearRateLimit(context.Context, int64) error { return nil }
func (s *scheduledTestAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) ClearModelRateLimits(context.Context, int64) error { return nil }
func (s *scheduledTestAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (s *scheduledTestAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (s *scheduledTestAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

type scheduledTestGroupRepoStub struct {
	groups map[int64]*Group
	err    error
}

func (s *scheduledTestGroupRepoStub) Create(context.Context, *Group) error           { return nil }
func (s *scheduledTestGroupRepoStub) GetByID(context.Context, int64) (*Group, error) { return nil, nil }
func (s *scheduledTestGroupRepoStub) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	if s.err != nil {
		return nil, s.err
	}
	group, ok := s.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return group, nil
}
func (s *scheduledTestGroupRepoStub) Update(context.Context, *Group) error { return nil }
func (s *scheduledTestGroupRepoStub) Delete(context.Context, int64) error  { return nil }
func (s *scheduledTestGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (s *scheduledTestGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *scheduledTestGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *scheduledTestGroupRepoStub) ListActive(context.Context) ([]Group, error) { return nil, nil }
func (s *scheduledTestGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return nil, nil
}
func (s *scheduledTestGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}
func (s *scheduledTestGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	return 0, 0, nil
}
func (s *scheduledTestGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *scheduledTestGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}
func (s *scheduledTestGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	return nil
}
func (s *scheduledTestGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	return nil
}

type scheduledTestJobRepoStub struct {
	mu         sync.Mutex
	nextJobID  int64
	nextItemID int64
	nextLogID  int64
	jobs       map[int64]*AccountTestJob
	items      map[int64][]*AccountTestJobItem
	logs       map[int64][]*AccountTestJobLog
}

func newScheduledTestJobRepoStub() *scheduledTestJobRepoStub {
	return &scheduledTestJobRepoStub{
		nextJobID:  1,
		nextItemID: 1,
		nextLogID:  1,
		jobs:       make(map[int64]*AccountTestJob),
		items:      make(map[int64][]*AccountTestJobItem),
		logs:       make(map[int64][]*AccountTestJobLog),
	}
}

func (s *scheduledTestJobRepoStub) CreateJob(ctx context.Context, job *AccountTestJob, items []*AccountTestJobItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobCopy := *job
	jobCopy.ID = s.nextJobID
	s.nextJobID++
	if jobCopy.CreatedAt.IsZero() {
		jobCopy.CreatedAt = time.Now().UTC()
	}
	jobCopy.UpdatedAt = jobCopy.CreatedAt
	s.jobs[jobCopy.ID] = &jobCopy
	job.ID = jobCopy.ID
	job.CreatedAt = jobCopy.CreatedAt
	job.UpdatedAt = jobCopy.UpdatedAt

	itemCopies := make([]*AccountTestJobItem, 0, len(items))
	for _, item := range items {
		itemCopy := *item
		itemCopy.ID = s.nextItemID
		s.nextItemID++
		itemCopy.JobID = jobCopy.ID
		if itemCopy.CreatedAt.IsZero() {
			itemCopy.CreatedAt = jobCopy.CreatedAt
		}
		itemCopy.UpdatedAt = itemCopy.CreatedAt
		item.ID = itemCopy.ID
		item.JobID = itemCopy.JobID
		item.CreatedAt = itemCopy.CreatedAt
		item.UpdatedAt = itemCopy.UpdatedAt
		itemCopies = append(itemCopies, &itemCopy)
	}
	s.items[jobCopy.ID] = itemCopies
	return nil
}

func (s *scheduledTestJobRepoStub) GetJobByID(ctx context.Context, id int64) (*AccountTestJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, errors.New("job not found")
	}
	jobCopy := *job
	return &jobCopy, nil
}

func (s *scheduledTestJobRepoStub) ListJobsByGroupID(ctx context.Context, groupID int64, limit int) ([]*AccountTestJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*AccountTestJob
	for _, job := range s.jobs {
		if job.GroupID == groupID {
			jobCopy := *job
			out = append(out, &jobCopy)
		}
	}
	return out, nil
}

func (s *scheduledTestJobRepoStub) ListItemsByJobID(ctx context.Context, jobID int64) ([]*AccountTestJobItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.items[jobID]
	out := make([]*AccountTestJobItem, 0, len(items))
	for _, item := range items {
		itemCopy := *item
		out = append(out, &itemCopy)
	}
	return out, nil
}

func (s *scheduledTestJobRepoStub) ListLogsByJobID(ctx context.Context, jobID int64, limit int) ([]*AccountTestJobLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logs := s.logs[jobID]
	out := make([]*AccountTestJobLog, 0, len(logs))
	for _, logEntry := range logs {
		logCopy := *logEntry
		out = append(out, &logCopy)
	}
	return out, nil
}

func (s *scheduledTestJobRepoStub) MarkJobRunning(ctx context.Context, jobID int64, startedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[jobID].Status = ScheduledTestJobStatusRunning
	s.jobs[jobID].StartedAt = &startedAt
	s.jobs[jobID].UpdatedAt = startedAt
	return nil
}

func (s *scheduledTestJobRepoStub) MarkJobFinished(ctx context.Context, jobID int64, status string, finishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[jobID].Status = status
	s.jobs[jobID].FinishedAt = &finishedAt
	s.jobs[jobID].UpdatedAt = finishedAt
	return nil
}

func (s *scheduledTestJobRepoStub) UpdateJobCounters(ctx context.Context, jobID int64, pendingDelta, runningDelta, succeededDelta, failedDelta int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[jobID]
	job.PendingAccounts += pendingDelta
	job.RunningAccounts += runningDelta
	job.SucceededAccounts += succeededDelta
	job.FailedAccounts += failedDelta
	job.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *scheduledTestJobRepoStub) MarkItemRunning(ctx context.Context, itemID int64, startedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, items := range s.items {
		for _, item := range items {
			if item.ID == itemID {
				item.Status = ScheduledTestJobItemStatusRunning
				item.StartedAt = &startedAt
				item.UpdatedAt = startedAt
				return nil
			}
		}
	}
	return nil
}

func (s *scheduledTestJobRepoStub) MarkItemFinished(ctx context.Context, itemID int64, status, responseText, errorMessage string, latencyMs int64, finishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, items := range s.items {
		for _, item := range items {
			if item.ID == itemID {
				item.Status = status
				item.ResponseText = responseText
				item.ErrorMessage = errorMessage
				item.LatencyMs = latencyMs
				item.FinishedAt = &finishedAt
				item.UpdatedAt = finishedAt
				return nil
			}
		}
	}
	return nil
}

func (s *scheduledTestJobRepoStub) CreateLog(ctx context.Context, logEntry *AccountTestJobLog) (*AccountTestJobLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logCopy := *logEntry
	logCopy.ID = s.nextLogID
	s.nextLogID++
	if logCopy.CreatedAt.IsZero() {
		logCopy.CreatedAt = time.Now().UTC()
	}
	s.logs[logCopy.JobID] = append(s.logs[logCopy.JobID], &logCopy)
	return &logCopy, nil
}

type scheduledTestRunnerStub struct {
	active    int64
	maxActive int64
	delay     time.Duration
}

func (s *scheduledTestRunnerStub) RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	active := atomic.AddInt64(&s.active, 1)
	for {
		currentMax := atomic.LoadInt64(&s.maxActive)
		if active <= currentMax || atomic.CompareAndSwapInt64(&s.maxActive, currentMax, active) {
			break
		}
	}
	time.Sleep(s.delay)
	atomic.AddInt64(&s.active, -1)
	return &ScheduledTestResult{
		Status:       "success",
		ResponseText: "ok",
		LatencyMs:    s.delay.Milliseconds(),
		StartedAt:    time.Now().UTC(),
		FinishedAt:   time.Now().UTC(),
	}, nil
}

type scheduledRecoveryStub struct{}

func (s *scheduledRecoveryStub) RecoverAccountAfterSuccessfulTest(context.Context, int64) (*SuccessfulTestRecoveryResult, error) {
	return &SuccessfulTestRecoveryResult{}, nil
}

func TestScheduledTestServiceValidatePlan_GroupPlanValidation(t *testing.T) {
	groupID := int64(7)
	accountID := int64(9)

	svc := NewScheduledTestService(
		nil,
		nil,
		nil,
		&scheduledTestAccountRepoStub{exists: true},
		&scheduledTestGroupRepoStub{groups: map[int64]*Group{groupID: {ID: groupID}}},
		nil,
		nil,
	)

	err := svc.validatePlan(context.Background(), &ScheduledTestPlan{
		GroupID:        &groupID,
		CronExpression: "*/5 * * * *",
		BatchSize:      0,
		OffsetSeconds:  0,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "batch_size")

	err = svc.validatePlan(context.Background(), &ScheduledTestPlan{
		GroupID:        &groupID,
		CronExpression: "*/5 * * * *",
		BatchSize:      2,
		OffsetSeconds:  -1,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "offset")

	err = svc.validatePlan(context.Background(), &ScheduledTestPlan{
		AccountID:      &accountID,
		GroupID:        &groupID,
		CronExpression: "*/5 * * * *",
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "exactly one")
}

func TestScheduledTestServiceCreateJobWithAccounts_StaggersBatches(t *testing.T) {
	jobRepo := newScheduledTestJobRepoStub()
	svc := NewScheduledTestService(
		nil,
		nil,
		jobRepo,
		nil,
		nil,
		nil,
		nil,
	)

	job, err := svc.createJobWithAccounts(context.Background(), CreateAccountTestJobRequest{
		GroupID:       1,
		TriggerSource: ScheduledTestJobTriggerScheduled,
		BatchSize:     2,
		OffsetSeconds: 30,
	}, []Account{
		{ID: 1, Name: "a1"},
		{ID: 2, Name: "a2"},
		{ID: 3, Name: "a3"},
		{ID: 4, Name: "a4"},
		{ID: 5, Name: "a5"},
	})
	require.NoError(t, err)
	require.NotNil(t, job)

	items, err := jobRepo.ListItemsByJobID(context.Background(), job.ID)
	require.NoError(t, err)
	require.Len(t, items, 5)
	require.Equal(t, 0*time.Second, items[1].ScheduledFor.Sub(items[0].ScheduledFor))
	require.Equal(t, 30*time.Second, items[2].ScheduledFor.Sub(items[0].ScheduledFor))
	require.Equal(t, 30*time.Second, items[3].ScheduledFor.Sub(items[1].ScheduledFor))
	require.Equal(t, 60*time.Second, items[4].ScheduledFor.Sub(items[0].ScheduledFor))
}

func TestScheduledTestServiceExecuteJob_LimitsConcurrencyToFive(t *testing.T) {
	jobRepo := newScheduledTestJobRepoStub()
	tester := &scheduledTestRunnerStub{delay: 25 * time.Millisecond}
	svc := NewScheduledTestService(
		nil,
		nil,
		jobRepo,
		nil,
		nil,
		tester,
		&scheduledRecoveryStub{},
	)

	job := &AccountTestJob{
		ID:              1,
		GroupID:         1,
		Status:          ScheduledTestJobStatusPending,
		PendingAccounts: 9,
		BatchSize:       9,
	}
	jobRepo.jobs[job.ID] = job

	items := make([]*AccountTestJobItem, 0, 9)
	now := time.Now().UTC()
	for i := 0; i < 9; i++ {
		item := &AccountTestJobItem{
			ID:           int64(i + 1),
			JobID:        job.ID,
			AccountID:    int64(i + 1),
			AccountName:  "acc",
			ScheduledFor: now,
			Status:       ScheduledTestJobItemStatusPending,
		}
		items = append(items, item)
	}
	jobRepo.items[job.ID] = items

	svc.executeJob(job, items, false)

	require.LessOrEqual(t, tester.maxActive, int64(5))
	require.Equal(t, ScheduledTestJobStatusCompleted, jobRepo.jobs[job.ID].Status)
	require.Equal(t, 9, jobRepo.jobs[job.ID].SucceededAccounts)
	require.Equal(t, 0, jobRepo.jobs[job.ID].RunningAccounts)
}
