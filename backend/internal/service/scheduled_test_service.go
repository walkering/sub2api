package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

var scheduledTestCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

const (
	scheduledTestDefaultMaxResults   = 50
	scheduledTestDefaultJobBatchSize = 5
	scheduledTestJobMaxConcurrency   = 5
	scheduledTestDefaultSnapshotLogs = 200
)

type scheduledTestJobHub struct {
	mu   sync.RWMutex
	subs map[int64]map[chan *AccountTestJobSnapshot]struct{}
}

func newScheduledTestJobHub() *scheduledTestJobHub {
	return &scheduledTestJobHub{
		subs: make(map[int64]map[chan *AccountTestJobSnapshot]struct{}),
	}
}

func (h *scheduledTestJobHub) Subscribe(jobID int64) (<-chan *AccountTestJobSnapshot, func()) {
	ch := make(chan *AccountTestJobSnapshot, 4)

	h.mu.Lock()
	if _, ok := h.subs[jobID]; !ok {
		h.subs[jobID] = make(map[chan *AccountTestJobSnapshot]struct{})
	}
	h.subs[jobID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if subs, ok := h.subs[jobID]; ok {
			if _, ok := subs[ch]; ok {
				delete(subs, ch)
				close(ch)
			}
			if len(subs) == 0 {
				delete(h.subs, jobID)
			}
		}
		h.mu.Unlock()
	}

	return ch, cancel
}

func (h *scheduledTestJobHub) Publish(jobID int64, snapshot *AccountTestJobSnapshot) {
	if h == nil || snapshot == nil {
		return
	}

	h.mu.RLock()
	subs := h.subs[jobID]
	for ch := range subs {
		select {
		case ch <- snapshot:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- snapshot:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

// ScheduledTestService provides CRUD operations for scheduled test plans and group test jobs.
type ScheduledTestService struct {
	planRepo      ScheduledTestPlanRepository
	resultRepo    ScheduledTestResultRepository
	jobRepo       AccountTestJobRepository
	accountRepo   AccountRepository
	groupRepo     GroupRepository
	accountTester ScheduledAccountTester
	recovery      ScheduledAccountRecovery
	jobHub        *scheduledTestJobHub
}

// NewScheduledTestService creates a new ScheduledTestService.
func NewScheduledTestService(
	planRepo ScheduledTestPlanRepository,
	resultRepo ScheduledTestResultRepository,
	jobRepo AccountTestJobRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	accountTester ScheduledAccountTester,
	recovery ScheduledAccountRecovery,
) *ScheduledTestService {
	return &ScheduledTestService{
		planRepo:      planRepo,
		resultRepo:    resultRepo,
		jobRepo:       jobRepo,
		accountRepo:   accountRepo,
		groupRepo:     groupRepo,
		accountTester: accountTester,
		recovery:      recovery,
		jobHub:        newScheduledTestJobHub(),
	}
}

// CreatePlan validates the cron expression, computes next_run_at, and persists the plan.
func (s *ScheduledTestService) CreatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	if err := s.validatePlan(ctx, plan); err != nil {
		return nil, err
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	return s.planRepo.Create(ctx, plan)
}

// GetPlan retrieves a plan by ID.
func (s *ScheduledTestService) GetPlan(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	return s.planRepo.GetByID(ctx, id)
}

// ListPlansByAccount returns all plans for a given account.
func (s *ScheduledTestService) ListPlansByAccount(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	return s.planRepo.ListByAccountID(ctx, accountID)
}

// ListPlansByGroup returns all plans for a given group.
func (s *ScheduledTestService) ListPlansByGroup(ctx context.Context, groupID int64) ([]*ScheduledTestPlan, error) {
	return s.planRepo.ListByGroupID(ctx, groupID)
}

// UpdatePlan validates cron and updates the plan.
func (s *ScheduledTestService) UpdatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	if err := s.validatePlan(ctx, plan); err != nil {
		return nil, err
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	return s.planRepo.Update(ctx, plan)
}

// DeletePlan removes a plan and its results (via CASCADE).
func (s *ScheduledTestService) DeletePlan(ctx context.Context, id int64) error {
	return s.planRepo.Delete(ctx, id)
}

// ListResults returns the most recent results for a plan.
func (s *ScheduledTestService) ListResults(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	if limit <= 0 {
		limit = scheduledTestDefaultMaxResults
	}
	return s.resultRepo.ListByPlanID(ctx, planID, limit)
}

// SaveResult inserts a result and prunes old entries beyond maxResults.
func (s *ScheduledTestService) SaveResult(ctx context.Context, planID int64, maxResults int, result *ScheduledTestResult) error {
	result.PlanID = planID
	if _, err := s.resultRepo.Create(ctx, result); err != nil {
		return err
	}
	return s.resultRepo.PruneOldResults(ctx, planID, maxResults)
}

func (s *ScheduledTestService) CreateJob(ctx context.Context, req CreateAccountTestJobRequest) (*AccountTestJob, error) {
	if err := s.validateCreateJobRequest(ctx, &req); err != nil {
		return nil, err
	}
	if s.accountRepo == nil {
		return nil, fmt.Errorf("account repository not configured")
	}

	accounts, err := s.accountRepo.ListSchedulableByGroupID(ctx, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("list schedulable accounts: %w", err)
	}

	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].ID == accounts[j].ID {
			return accounts[i].Name < accounts[j].Name
		}
		return accounts[i].ID < accounts[j].ID
	})

	return s.createJobWithAccounts(ctx, req, accounts)
}

func (s *ScheduledTestService) CreateJobForPlan(ctx context.Context, plan *ScheduledTestPlan) (*AccountTestJob, error) {
	if plan == nil || !plan.IsGroupPlan() {
		return nil, infraerrors.BadRequest("SCHEDULED_TEST_INVALID_GROUP_PLAN", "group scheduled test plan is required")
	}

	return s.CreateJob(ctx, CreateAccountTestJobRequest{
		GroupID:       *plan.GroupID,
		PlanID:        &plan.ID,
		ModelID:       plan.ModelID,
		TriggerSource: ScheduledTestJobTriggerScheduled,
		BatchSize:     plan.BatchSize,
		OffsetSeconds: plan.OffsetSeconds,
		AutoRecover:   plan.AutoRecover,
	})
}

func (s *ScheduledTestService) ListJobsByGroup(ctx context.Context, groupID int64, limit int) ([]*AccountTestJob, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.jobRepo.ListJobsByGroupID(ctx, groupID, limit)
}

func (s *ScheduledTestService) GetJob(ctx context.Context, id int64) (*AccountTestJob, error) {
	return s.jobRepo.GetJobByID(ctx, id)
}

func (s *ScheduledTestService) GetJobSnapshot(ctx context.Context, jobID int64, logLimit int) (*AccountTestJobSnapshot, error) {
	if logLimit <= 0 {
		logLimit = scheduledTestDefaultSnapshotLogs
	}

	job, err := s.jobRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	items, err := s.jobRepo.ListItemsByJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	logs, err := s.jobRepo.ListLogsByJobID(ctx, jobID, logLimit)
	if err != nil {
		return nil, err
	}
	return &AccountTestJobSnapshot{
		Job:   job,
		Items: items,
		Logs:  logs,
	}, nil
}

func (s *ScheduledTestService) SubscribeJob(jobID int64) (<-chan *AccountTestJobSnapshot, func(), error) {
	if s.jobHub == nil {
		return nil, nil, fmt.Errorf("job hub not configured")
	}
	ch, cancel := s.jobHub.Subscribe(jobID)
	return ch, cancel, nil
}

func (s *ScheduledTestService) validatePlan(ctx context.Context, plan *ScheduledTestPlan) error {
	if plan == nil {
		return infraerrors.BadRequest("SCHEDULED_TEST_PLAN_NIL", "scheduled test plan is required")
	}
	if plan.CronExpression == "" {
		return infraerrors.BadRequest("SCHEDULED_TEST_CRON_REQUIRED", "cron_expression is required")
	}

	if plan.AccountID != nil && *plan.AccountID <= 0 {
		plan.AccountID = nil
	}
	if plan.GroupID != nil && *plan.GroupID <= 0 {
		plan.GroupID = nil
	}

	hasAccount := plan.AccountID != nil
	hasGroup := plan.GroupID != nil
	if hasAccount == hasGroup {
		return infraerrors.BadRequest("SCHEDULED_TEST_TARGET_INVALID", "exactly one of account_id or group_id is required")
	}

	if plan.MaxResults <= 0 {
		plan.MaxResults = scheduledTestDefaultMaxResults
	}

	if hasAccount {
		if s.accountRepo == nil {
			return fmt.Errorf("account repository not configured")
		}
		exists, err := s.accountRepo.ExistsByID(ctx, *plan.AccountID)
		if err != nil {
			return fmt.Errorf("check account exists: %w", err)
		}
		if !exists {
			return ErrAccountNotFound
		}
		plan.GroupID = nil
		plan.BatchSize = 0
		plan.OffsetSeconds = 0
		return nil
	}

	if plan.BatchSize <= 0 {
		return infraerrors.BadRequest("SCHEDULED_TEST_BATCH_SIZE_INVALID", "batch_size must be greater than 0")
	}
	if plan.OffsetSeconds < 0 {
		return infraerrors.BadRequest("SCHEDULED_TEST_OFFSET_INVALID", "offset must be greater than or equal to 0")
	}
	if s.groupRepo == nil {
		return fmt.Errorf("group repository not configured")
	}
	if _, err := s.groupRepo.GetByIDLite(ctx, *plan.GroupID); err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	plan.AccountID = nil
	return nil
}

func (s *ScheduledTestService) validateCreateJobRequest(ctx context.Context, req *CreateAccountTestJobRequest) error {
	if req == nil {
		return infraerrors.BadRequest("ACCOUNT_TEST_JOB_REQUEST_NIL", "job request is required")
	}
	if req.GroupID <= 0 {
		return infraerrors.BadRequest("ACCOUNT_TEST_JOB_GROUP_REQUIRED", "group_id is required")
	}
	if req.TriggerSource == "" {
		req.TriggerSource = ScheduledTestJobTriggerManual
	}
	if req.BatchSize <= 0 {
		req.BatchSize = scheduledTestDefaultJobBatchSize
	}
	if req.OffsetSeconds < 0 {
		return infraerrors.BadRequest("ACCOUNT_TEST_JOB_OFFSET_INVALID", "offset must be greater than or equal to 0")
	}
	if req.CreatedBy != nil && *req.CreatedBy <= 0 {
		req.CreatedBy = nil
	}

	if s.groupRepo == nil {
		return fmt.Errorf("group repository not configured")
	}
	if _, err := s.groupRepo.GetByIDLite(ctx, req.GroupID); err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	return nil
}

func (s *ScheduledTestService) createJobWithAccounts(ctx context.Context, req CreateAccountTestJobRequest, accounts []Account) (*AccountTestJob, error) {
	if s.jobRepo == nil {
		return nil, fmt.Errorf("job repository not configured")
	}

	now := time.Now().UTC()
	job := &AccountTestJob{
		GroupID:           req.GroupID,
		PlanID:            req.PlanID,
		ModelID:           req.ModelID,
		TriggerSource:     req.TriggerSource,
		Status:            ScheduledTestJobStatusPending,
		BatchSize:         req.BatchSize,
		OffsetSeconds:     req.OffsetSeconds,
		TotalAccounts:     len(accounts),
		PendingAccounts:   len(accounts),
		RunningAccounts:   0,
		SucceededAccounts: 0,
		FailedAccounts:    0,
		CreatedBy:         req.CreatedBy,
	}

	items := make([]*AccountTestJobItem, 0, len(accounts))
	if len(accounts) == 0 {
		job.Status = ScheduledTestJobStatusNoAccounts
		job.PendingAccounts = 0
		job.StartedAt = &now
		job.FinishedAt = &now
	} else {
		for idx := range accounts {
			account := accounts[idx]
			scheduledFor := now
			if req.OffsetSeconds > 0 {
				batchIndex := idx / req.BatchSize
				scheduledFor = scheduledFor.Add(time.Duration(batchIndex*req.OffsetSeconds) * time.Second)
			}
			items = append(items, &AccountTestJobItem{
				AccountID:    account.ID,
				AccountName:  account.Name,
				ScheduledFor: scheduledFor,
				Status:       ScheduledTestJobItemStatusPending,
			})
		}
	}

	if err := s.jobRepo.CreateJob(ctx, job, items); err != nil {
		return nil, fmt.Errorf("create test job: %w", err)
	}

	if len(items) == 0 {
		s.appendJobLog(context.Background(), &AccountTestJobLog{
			JobID:     job.ID,
			EventType: ScheduledTestJobLogTypeInfo,
			Status:    ScheduledTestJobStatusNoAccounts,
			Message:   "没有可测试账号",
		})
		s.appendJobLog(context.Background(), &AccountTestJobLog{
			JobID:     job.ID,
			EventType: ScheduledTestJobLogTypeJobFinished,
			Status:    ScheduledTestJobStatusNoAccounts,
			Message:   "任务结束：没有可测试账号",
		})
		s.broadcastJobSnapshot(job.ID)
		return job, nil
	}

	s.broadcastJobSnapshot(job.ID)
	go s.executeJob(job, items, req.AutoRecover)
	return job, nil
}

func (s *ScheduledTestService) executeJob(job *AccountTestJob, items []*AccountTestJobItem, autoRecover bool) {
	if job == nil {
		return
	}
	if s.jobRepo == nil || s.accountTester == nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d missing dependency: jobRepo=%t accountTester=%t", job.ID, s.jobRepo != nil, s.accountTester != nil)
		return
	}

	ctx := context.Background()
	startedAt := time.Now().UTC()
	if err := s.jobRepo.MarkJobRunning(ctx, job.ID, startedAt); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d mark running failed: %v", job.ID, err)
	}
	s.appendJobLog(ctx, &AccountTestJobLog{
		JobID:     job.ID,
		EventType: ScheduledTestJobLogTypeJobStarted,
		Status:    ScheduledTestJobStatusRunning,
		Message:   fmt.Sprintf("任务开始，共 %d 个账号", len(items)),
	})
	s.broadcastJobSnapshot(job.ID)

	sem := make(chan struct{}, scheduledTestJobMaxConcurrency)
	var wg sync.WaitGroup

	for _, item := range items {
		if wait := time.Until(item.ScheduledFor); wait > 0 {
			timer := time.NewTimer(wait)
			<-timer.C
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(jobItem *AccountTestJobItem) {
			defer wg.Done()
			defer func() { <-sem }()
			s.executeJobItem(ctx, job, jobItem, autoRecover)
		}(item)
	}

	wg.Wait()

	finishedAt := time.Now().UTC()
	if err := s.jobRepo.MarkJobFinished(ctx, job.ID, ScheduledTestJobStatusCompleted, finishedAt); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d mark finished failed: %v", job.ID, err)
	}
	s.appendJobLog(ctx, &AccountTestJobLog{
		JobID:     job.ID,
		EventType: ScheduledTestJobLogTypeJobFinished,
		Status:    ScheduledTestJobStatusCompleted,
		Message:   "任务执行完成",
	})
	s.broadcastJobSnapshot(job.ID)
}

func (s *ScheduledTestService) executeJobItem(ctx context.Context, job *AccountTestJob, item *AccountTestJobItem, autoRecover bool) {
	if job == nil || item == nil {
		return
	}

	startedAt := time.Now().UTC()
	if err := s.jobRepo.MarkItemRunning(ctx, item.ID, startedAt); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d item=%d mark running failed: %v", job.ID, item.ID, err)
	}
	if err := s.jobRepo.UpdateJobCounters(ctx, job.ID, -1, 1, 0, 0); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d update counters(start) failed: %v", job.ID, err)
	}
	accountID := item.AccountID
	s.appendJobLog(ctx, &AccountTestJobLog{
		JobID:       job.ID,
		AccountID:   &accountID,
		AccountName: item.AccountName,
		EventType:   ScheduledTestJobLogTypeAccountStarted,
		Status:      ScheduledTestJobItemStatusRunning,
		Message:     fmt.Sprintf("开始测试账号：%s", item.AccountName),
	})
	s.broadcastJobSnapshot(job.ID)

	result, err := s.accountTester.RunTestBackground(ctx, item.AccountID, job.ModelID)
	finishedAt := time.Now().UTC()
	if result == nil {
		result = &ScheduledTestResult{
			Status:       "failed",
			ResponseText: "",
			ErrorMessage: "",
			LatencyMs:    finishedAt.Sub(startedAt).Milliseconds(),
			StartedAt:    startedAt,
			FinishedAt:   finishedAt,
		}
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = startedAt
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = finishedAt
	}
	if result.LatencyMs == 0 && !result.FinishedAt.IsZero() && !result.StartedAt.IsZero() {
		result.LatencyMs = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
	}
	if err != nil && result.ErrorMessage == "" {
		result.ErrorMessage = err.Error()
	}

	itemStatus := ScheduledTestJobItemStatusSucceeded
	jobSucceededDelta := 1
	jobFailedDelta := 0
	logStatus := result.Status
	if result.Status != "success" {
		itemStatus = ScheduledTestJobItemStatusFailed
		jobSucceededDelta = 0
		jobFailedDelta = 1
		if logStatus == "" {
			logStatus = ScheduledTestJobItemStatusFailed
		}
	}

	if err := s.jobRepo.MarkItemFinished(ctx, item.ID, itemStatus, result.ResponseText, result.ErrorMessage, result.LatencyMs, result.FinishedAt); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d item=%d mark finished failed: %v", job.ID, item.ID, err)
	}
	if err := s.jobRepo.UpdateJobCounters(ctx, job.ID, 0, -1, jobSucceededDelta, jobFailedDelta); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d update counters(finish) failed: %v", job.ID, err)
	}

	if autoRecover && itemStatus == ScheduledTestJobItemStatusSucceeded {
		s.tryRecoverAccount(ctx, item.AccountID, job.ID)
	}

	message := fmt.Sprintf("账号测试成功：%s", item.AccountName)
	if itemStatus == ScheduledTestJobItemStatusFailed {
		message = fmt.Sprintf("账号测试失败：%s", item.AccountName)
	}
	s.appendJobLog(ctx, &AccountTestJobLog{
		JobID:        job.ID,
		AccountID:    &accountID,
		AccountName:  item.AccountName,
		EventType:    ScheduledTestJobLogTypeAccountFinished,
		Status:       logStatus,
		Message:      message,
		ResponseText: result.ResponseText,
		ErrorMessage: result.ErrorMessage,
		LatencyMs:    result.LatencyMs,
	})
	s.broadcastJobSnapshot(job.ID)
}

func (s *ScheduledTestService) appendJobLog(ctx context.Context, logItem *AccountTestJobLog) {
	if s == nil || s.jobRepo == nil || logItem == nil {
		return
	}
	if _, err := s.jobRepo.CreateLog(ctx, logItem); err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] append log failed: job=%d event=%s err=%v", logItem.JobID, logItem.EventType, err)
	}
}

func (s *ScheduledTestService) broadcastJobSnapshot(jobID int64) {
	if s == nil || s.jobHub == nil || jobID <= 0 {
		return
	}
	snapshot, err := s.GetJobSnapshot(context.Background(), jobID, scheduledTestDefaultSnapshotLogs)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] build snapshot failed: job=%d err=%v", jobID, err)
		return
	}
	s.jobHub.Publish(jobID, snapshot)
}

func (s *ScheduledTestService) tryRecoverAccount(ctx context.Context, accountID int64, jobID int64) {
	if s.recovery == nil {
		return
	}

	recovery, err := s.recovery.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test", "[ScheduledTest] job=%d auto-recover failed: account=%d err=%v", jobID, accountID, err)
		return
	}
	if recovery == nil {
		return
	}
	if recovery.ClearedError || recovery.ClearedRateLimit {
		msg := fmt.Sprintf("账号恢复完成：account=%d cleared_error=%t cleared_rate_limit=%t", accountID, recovery.ClearedError, recovery.ClearedRateLimit)
		s.appendJobLog(context.Background(), &AccountTestJobLog{
			JobID:     jobID,
			AccountID: &accountID,
			EventType: ScheduledTestJobLogTypeInfo,
			Status:    http.StatusText(http.StatusOK),
			Message:   msg,
		})
	}
}

func computeNextRun(cronExpr string, from time.Time) (time.Time, error) {
	sched, err := scheduledTestCronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from), nil
}
