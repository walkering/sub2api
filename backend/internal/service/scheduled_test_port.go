package service

import (
	"context"
	"time"
)

const (
	ScheduledTestJobTriggerManual    = "manual"
	ScheduledTestJobTriggerScheduled = "scheduled"

	ScheduledTestJobStatusPending    = "pending"
	ScheduledTestJobStatusRunning    = "running"
	ScheduledTestJobStatusCompleted  = "completed"
	ScheduledTestJobStatusFailed     = "failed"
	ScheduledTestJobStatusNoAccounts = "no_accounts"

	ScheduledTestJobItemStatusPending   = "pending"
	ScheduledTestJobItemStatusRunning   = "running"
	ScheduledTestJobItemStatusSucceeded = "succeeded"
	ScheduledTestJobItemStatusFailed    = "failed"

	ScheduledTestJobLogTypeJobStarted      = "job_started"
	ScheduledTestJobLogTypeAccountStarted  = "account_started"
	ScheduledTestJobLogTypeAccountFinished = "account_finished"
	ScheduledTestJobLogTypeJobFinished     = "job_finished"
	ScheduledTestJobLogTypeInfo            = "info"
)

// ScheduledTestPlan represents a scheduled test plan domain model.
type ScheduledTestPlan struct {
	ID             int64      `json:"id"`
	AccountID      *int64     `json:"account_id,omitempty"`
	GroupID        *int64     `json:"group_id,omitempty"`
	ModelID        string     `json:"model_id"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	MaxResults     int        `json:"max_results"`
	AutoRecover    bool       `json:"auto_recover"`
	BatchSize      int        `json:"batch_size"`
	OffsetSeconds  int        `json:"offset"`
	LastRunAt      *time.Time `json:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (p *ScheduledTestPlan) IsGroupPlan() bool {
	return p != nil && p.GroupID != nil && *p.GroupID > 0
}

func (p *ScheduledTestPlan) IsAccountPlan() bool {
	return p != nil && p.AccountID != nil && *p.AccountID > 0
}

// ScheduledTestResult represents a single test execution result.
type ScheduledTestResult struct {
	ID           int64     `json:"id"`
	PlanID       int64     `json:"plan_id"`
	Status       string    `json:"status"`
	ResponseText string    `json:"response_text"`
	ErrorMessage string    `json:"error_message"`
	LatencyMs    int64     `json:"latency_ms"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type AccountTestJob struct {
	ID                int64      `json:"id"`
	GroupID           int64      `json:"group_id"`
	PlanID            *int64     `json:"plan_id,omitempty"`
	ModelID           string     `json:"model_id"`
	TriggerSource     string     `json:"trigger_source"`
	Status            string     `json:"status"`
	BatchSize         int        `json:"batch_size"`
	OffsetSeconds     int        `json:"offset"`
	TotalAccounts     int        `json:"total_accounts"`
	PendingAccounts   int        `json:"pending_accounts"`
	RunningAccounts   int        `json:"running_accounts"`
	SucceededAccounts int        `json:"succeeded_accounts"`
	FailedAccounts    int        `json:"failed_accounts"`
	CreatedBy         *int64     `json:"created_by,omitempty"`
	StartedAt         *time.Time `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type AccountTestJobItem struct {
	ID           int64      `json:"id"`
	JobID        int64      `json:"job_id"`
	AccountID    int64      `json:"account_id"`
	AccountName  string     `json:"account_name"`
	ScheduledFor time.Time  `json:"scheduled_for"`
	Status       string     `json:"status"`
	ResponseText string     `json:"response_text"`
	ErrorMessage string     `json:"error_message"`
	LatencyMs    int64      `json:"latency_ms"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AccountTestJobLog struct {
	ID           int64     `json:"id"`
	JobID        int64     `json:"job_id"`
	AccountID    *int64    `json:"account_id,omitempty"`
	AccountName  string    `json:"account_name"`
	EventType    string    `json:"event_type"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	ResponseText string    `json:"response_text"`
	ErrorMessage string    `json:"error_message"`
	LatencyMs    int64     `json:"latency_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

type AccountTestJobSnapshot struct {
	Job   *AccountTestJob       `json:"job"`
	Items []*AccountTestJobItem `json:"items"`
	Logs  []*AccountTestJobLog  `json:"logs"`
}

type CreateAccountTestJobRequest struct {
	GroupID       int64
	PlanID        *int64
	ModelID       string
	TriggerSource string
	BatchSize     int
	OffsetSeconds int
	CreatedBy     *int64
	AutoRecover   bool
}

// ScheduledTestPlanRepository defines the data access interface for test plans.
type ScheduledTestPlanRepository interface {
	Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error)
	GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error)
	ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error)
	ListByGroupID(ctx context.Context, groupID int64) ([]*ScheduledTestPlan, error)
	ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error)
	Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error)
	Delete(ctx context.Context, id int64) error
	UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error
}

// ScheduledTestResultRepository defines the data access interface for test results.
type ScheduledTestResultRepository interface {
	Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error)
	ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error)
	PruneOldResults(ctx context.Context, planID int64, keepCount int) error
}

type AccountTestJobRepository interface {
	CreateJob(ctx context.Context, job *AccountTestJob, items []*AccountTestJobItem) error
	GetJobByID(ctx context.Context, id int64) (*AccountTestJob, error)
	ListJobsByGroupID(ctx context.Context, groupID int64, limit int) ([]*AccountTestJob, error)
	ListItemsByJobID(ctx context.Context, jobID int64) ([]*AccountTestJobItem, error)
	ListLogsByJobID(ctx context.Context, jobID int64, limit int) ([]*AccountTestJobLog, error)
	MarkJobRunning(ctx context.Context, jobID int64, startedAt time.Time) error
	MarkJobFinished(ctx context.Context, jobID int64, status string, finishedAt time.Time) error
	UpdateJobCounters(ctx context.Context, jobID int64, pendingDelta, runningDelta, succeededDelta, failedDelta int) error
	MarkItemRunning(ctx context.Context, itemID int64, startedAt time.Time) error
	MarkItemFinished(ctx context.Context, itemID int64, status, responseText, errorMessage string, latencyMs int64, finishedAt time.Time) error
	CreateLog(ctx context.Context, log *AccountTestJobLog) (*AccountTestJobLog, error)
}

type ScheduledAccountTester interface {
	RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

type ScheduledAccountRecovery interface {
	RecoverAccountAfterSuccessfulTest(ctx context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error)
}
