package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// --- Plan Repository ---

type scheduledTestPlanRepository struct {
	db *sql.DB
}

func NewScheduledTestPlanRepository(db *sql.DB) service.ScheduledTestPlanRepository {
	return &scheduledTestPlanRepository{db: db}
}

func (r *scheduledTestPlanRepository) Create(ctx context.Context, plan *service.ScheduledTestPlan) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_plans (
			account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, next_run_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, last_run_at, next_run_at, created_at, updated_at
	`,
		nullableInt64(plan.AccountID),
		nullableInt64(plan.GroupID),
		plan.ModelID,
		plan.CronExpression,
		plan.Enabled,
		plan.MaxResults,
		plan.AutoRecover,
		plan.BatchSize,
		plan.OffsetSeconds,
		plan.NextRunAt,
	)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) GetByID(ctx context.Context, id int64) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_test_plans
		WHERE id = $1
	`, id)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) ListByAccountID(ctx context.Context, accountID int64) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_test_plans
		WHERE account_id = $1
		ORDER BY created_at DESC, id DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) ListByGroupID(ctx context.Context, groupID int64) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_test_plans
		WHERE group_id = $1
		ORDER BY created_at DESC, id DESC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) ListDue(ctx context.Context, now time.Time) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_test_plans
		WHERE enabled = true AND next_run_at <= $1
		ORDER BY next_run_at ASC, id ASC
	`, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}

func (r *scheduledTestPlanRepository) Update(ctx context.Context, plan *service.ScheduledTestPlan) (*service.ScheduledTestPlan, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE scheduled_test_plans
		SET model_id = $2,
			cron_expression = $3,
			enabled = $4,
			max_results = $5,
			auto_recover = $6,
			batch_size = $7,
			offset_seconds = $8,
			next_run_at = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, account_id, group_id, model_id, cron_expression, enabled, max_results, auto_recover,
			batch_size, offset_seconds, last_run_at, next_run_at, created_at, updated_at
	`, plan.ID, plan.ModelID, plan.CronExpression, plan.Enabled, plan.MaxResults, plan.AutoRecover, plan.BatchSize, plan.OffsetSeconds, plan.NextRunAt)
	return scanPlan(row)
}

func (r *scheduledTestPlanRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM scheduled_test_plans WHERE id = $1`, id)
	return err
}

func (r *scheduledTestPlanRepository) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_test_plans
		SET last_run_at = $2, next_run_at = $3, updated_at = NOW()
		WHERE id = $1
	`, id, lastRunAt, nextRunAt)
	return err
}

// --- Result Repository ---

type scheduledTestResultRepository struct {
	db *sql.DB
}

func NewScheduledTestResultRepository(db *sql.DB) service.ScheduledTestResultRepository {
	return &scheduledTestResultRepository{db: db}
}

func (r *scheduledTestResultRepository) Create(ctx context.Context, result *service.ScheduledTestResult) (*service.ScheduledTestResult, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_test_results (plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at
	`, result.PlanID, result.Status, result.ResponseText, result.ErrorMessage, result.LatencyMs, result.StartedAt, result.FinishedAt)

	out := &service.ScheduledTestResult{}
	if err := row.Scan(
		&out.ID, &out.PlanID, &out.Status, &out.ResponseText, &out.ErrorMessage,
		&out.LatencyMs, &out.StartedAt, &out.FinishedAt, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *scheduledTestResultRepository) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*service.ScheduledTestResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, plan_id, status, response_text, error_message, latency_ms, started_at, finished_at, created_at
		FROM scheduled_test_results
		WHERE plan_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, planID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*service.ScheduledTestResult
	for rows.Next() {
		result := &service.ScheduledTestResult{}
		if err := rows.Scan(
			&result.ID, &result.PlanID, &result.Status, &result.ResponseText, &result.ErrorMessage,
			&result.LatencyMs, &result.StartedAt, &result.FinishedAt, &result.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

func (r *scheduledTestResultRepository) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM scheduled_test_results
		WHERE id IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY created_at DESC) AS rn
				FROM scheduled_test_results
				WHERE plan_id = $1
			) ranked
			WHERE rn > $2
		)
	`, planID, keepCount)
	return err
}

// --- Job Repository ---

type accountTestJobRepository struct {
	db *sql.DB
}

func NewAccountTestJobRepository(db *sql.DB) service.AccountTestJobRepository {
	return &accountTestJobRepository{db: db}
}

func (r *accountTestJobRepository) CreateJob(ctx context.Context, job *service.AccountTestJob, items []*service.AccountTestJobItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var startedAt any
	if job.StartedAt != nil {
		startedAt = *job.StartedAt
	}
	var finishedAt any
	if job.FinishedAt != nil {
		finishedAt = *job.FinishedAt
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO account_test_jobs (
			group_id, plan_id, model_id, trigger_source, status, batch_size, offset_seconds,
			total_accounts, pending_accounts, running_accounts, succeeded_accounts, failed_accounts,
			created_by, started_at, finished_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`,
		job.GroupID,
		nullableInt64(job.PlanID),
		job.ModelID,
		job.TriggerSource,
		job.Status,
		job.BatchSize,
		job.OffsetSeconds,
		job.TotalAccounts,
		job.PendingAccounts,
		job.RunningAccounts,
		job.SucceededAccounts,
		job.FailedAccounts,
		nullableInt64(job.CreatedBy),
		startedAt,
		finishedAt,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		err = tx.QueryRowContext(ctx, `
			INSERT INTO account_test_job_items (
				job_id, account_id, account_name, scheduled_for, status, response_text, error_message,
				latency_ms, started_at, finished_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, '', '', 0, NULL, NULL, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`, job.ID, item.AccountID, item.AccountName, item.ScheduledFor, item.Status).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return err
		}
		item.JobID = job.ID
	}

	return tx.Commit()
}

func (r *accountTestJobRepository) GetJobByID(ctx context.Context, id int64) (*service.AccountTestJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, group_id, plan_id, model_id, trigger_source, status, batch_size, offset_seconds,
			total_accounts, pending_accounts, running_accounts, succeeded_accounts, failed_accounts,
			created_by, started_at, finished_at, created_at, updated_at
		FROM account_test_jobs
		WHERE id = $1
	`, id)
	return scanJob(row)
}

func (r *accountTestJobRepository) ListJobsByGroupID(ctx context.Context, groupID int64, limit int) ([]*service.AccountTestJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, group_id, plan_id, model_id, trigger_source, status, batch_size, offset_seconds,
			total_accounts, pending_accounts, running_accounts, succeeded_accounts, failed_accounts,
			created_by, started_at, finished_at, created_at, updated_at
		FROM account_test_jobs
		WHERE group_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var jobs []*service.AccountTestJob
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *accountTestJobRepository) ListItemsByJobID(ctx context.Context, jobID int64) ([]*service.AccountTestJobItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, job_id, account_id, account_name, scheduled_for, status, response_text, error_message,
			latency_ms, started_at, finished_at, created_at, updated_at
		FROM account_test_job_items
		WHERE job_id = $1
		ORDER BY scheduled_for ASC, account_id ASC, id ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []*service.AccountTestJobItem
	for rows.Next() {
		item, err := scanJobItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *accountTestJobRepository) ListLogsByJobID(ctx context.Context, jobID int64, limit int) ([]*service.AccountTestJobLog, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, job_id, account_id, account_name, event_type, status, message, response_text, error_message, latency_ms, created_at
		FROM (
			SELECT id, job_id, account_id, account_name, event_type, status, message, response_text, error_message, latency_ms, created_at
			FROM account_test_job_logs
			WHERE job_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		) logs
		ORDER BY created_at ASC, id ASC
	`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var logs []*service.AccountTestJobLog
	for rows.Next() {
		logEntry, err := scanJobLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, logEntry)
	}
	return logs, rows.Err()
}

func (r *accountTestJobRepository) MarkJobRunning(ctx context.Context, jobID int64, startedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_test_jobs
		SET status = $2,
			started_at = COALESCE(started_at, $3),
			updated_at = NOW()
		WHERE id = $1
	`, jobID, service.ScheduledTestJobStatusRunning, startedAt)
	return err
}

func (r *accountTestJobRepository) MarkJobFinished(ctx context.Context, jobID int64, status string, finishedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_test_jobs
		SET status = $2,
			finished_at = $3,
			updated_at = NOW()
		WHERE id = $1
	`, jobID, status, finishedAt)
	return err
}

func (r *accountTestJobRepository) UpdateJobCounters(ctx context.Context, jobID int64, pendingDelta, runningDelta, succeededDelta, failedDelta int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_test_jobs
		SET pending_accounts = GREATEST(pending_accounts + $2, 0),
			running_accounts = GREATEST(running_accounts + $3, 0),
			succeeded_accounts = GREATEST(succeeded_accounts + $4, 0),
			failed_accounts = GREATEST(failed_accounts + $5, 0),
			updated_at = NOW()
		WHERE id = $1
	`, jobID, pendingDelta, runningDelta, succeededDelta, failedDelta)
	return err
}

func (r *accountTestJobRepository) MarkItemRunning(ctx context.Context, itemID int64, startedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_test_job_items
		SET status = $2,
			started_at = $3,
			updated_at = NOW()
		WHERE id = $1
	`, itemID, service.ScheduledTestJobItemStatusRunning, startedAt)
	return err
}

func (r *accountTestJobRepository) MarkItemFinished(ctx context.Context, itemID int64, status, responseText, errorMessage string, latencyMs int64, finishedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_test_job_items
		SET status = $2,
			response_text = $3,
			error_message = $4,
			latency_ms = $5,
			finished_at = $6,
			updated_at = NOW()
		WHERE id = $1
	`, itemID, status, responseText, errorMessage, latencyMs, finishedAt)
	return err
}

func (r *accountTestJobRepository) CreateLog(ctx context.Context, logEntry *service.AccountTestJobLog) (*service.AccountTestJobLog, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO account_test_job_logs (
			job_id, account_id, account_name, event_type, status, message, response_text, error_message, latency_ms, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id, created_at
	`,
		logEntry.JobID,
		nullableInt64(logEntry.AccountID),
		logEntry.AccountName,
		logEntry.EventType,
		logEntry.Status,
		logEntry.Message,
		logEntry.ResponseText,
		logEntry.ErrorMessage,
		logEntry.LatencyMs,
	)
	out := *logEntry
	if err := row.Scan(&out.ID, &out.CreatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- scan helpers ---

type scannable interface {
	Scan(dest ...any) error
}

func scanPlan(row scannable) (*service.ScheduledTestPlan, error) {
	plan := &service.ScheduledTestPlan{}
	var accountID sql.NullInt64
	var groupID sql.NullInt64
	var lastRunAt sql.NullTime
	var nextRunAt sql.NullTime
	if err := row.Scan(
		&plan.ID,
		&accountID,
		&groupID,
		&plan.ModelID,
		&plan.CronExpression,
		&plan.Enabled,
		&plan.MaxResults,
		&plan.AutoRecover,
		&plan.BatchSize,
		&plan.OffsetSeconds,
		&lastRunAt,
		&nextRunAt,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	); err != nil {
		return nil, err
	}
	plan.AccountID = nullInt64Ptr(accountID)
	plan.GroupID = nullInt64Ptr(groupID)
	plan.LastRunAt = nullTimePtr(lastRunAt)
	plan.NextRunAt = nullTimePtr(nextRunAt)
	return plan, nil
}

func scanPlans(rows *sql.Rows) ([]*service.ScheduledTestPlan, error) {
	var plans []*service.ScheduledTestPlan
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func scanJob(row scannable) (*service.AccountTestJob, error) {
	job := &service.AccountTestJob{}
	var planID sql.NullInt64
	var createdBy sql.NullInt64
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := row.Scan(
		&job.ID,
		&job.GroupID,
		&planID,
		&job.ModelID,
		&job.TriggerSource,
		&job.Status,
		&job.BatchSize,
		&job.OffsetSeconds,
		&job.TotalAccounts,
		&job.PendingAccounts,
		&job.RunningAccounts,
		&job.SucceededAccounts,
		&job.FailedAccounts,
		&createdBy,
		&startedAt,
		&finishedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	job.PlanID = nullInt64Ptr(planID)
	job.CreatedBy = nullInt64Ptr(createdBy)
	job.StartedAt = nullTimePtr(startedAt)
	job.FinishedAt = nullTimePtr(finishedAt)
	return job, nil
}

func scanJobItem(row scannable) (*service.AccountTestJobItem, error) {
	item := &service.AccountTestJobItem{}
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.JobID,
		&item.AccountID,
		&item.AccountName,
		&item.ScheduledFor,
		&item.Status,
		&item.ResponseText,
		&item.ErrorMessage,
		&item.LatencyMs,
		&startedAt,
		&finishedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.StartedAt = nullTimePtr(startedAt)
	item.FinishedAt = nullTimePtr(finishedAt)
	return item, nil
}

func scanJobLog(row scannable) (*service.AccountTestJobLog, error) {
	logEntry := &service.AccountTestJobLog{}
	var accountID sql.NullInt64
	if err := row.Scan(
		&logEntry.ID,
		&logEntry.JobID,
		&accountID,
		&logEntry.AccountName,
		&logEntry.EventType,
		&logEntry.Status,
		&logEntry.Message,
		&logEntry.ResponseText,
		&logEntry.ErrorMessage,
		&logEntry.LatencyMs,
		&logEntry.CreatedAt,
	); err != nil {
		return nil, err
	}
	logEntry.AccountID = nullInt64Ptr(accountID)
	return logEntry, nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}
