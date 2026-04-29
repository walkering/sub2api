package admin

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type openAIAutoReauthJobStatus string

const (
	openAIAutoReauthJobStatusRunning   openAIAutoReauthJobStatus = "running"
	openAIAutoReauthJobStatusCompleted openAIAutoReauthJobStatus = "completed"
)

type openAIAutoReauthJobIssue struct {
	AccountID int64  `json:"account_id"`
	Message   string `json:"message"`
}

type openAIAutoReauthJobLogEntry struct {
	Seq       int64  `json:"seq"`
	At        string `json:"at"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	AccountID *int64 `json:"account_id,omitempty"`
}

type openAIAutoReauthJobSnapshot struct {
	JobID      string                        `json:"job_id"`
	Status     openAIAutoReauthJobStatus     `json:"status"`
	Total      int                           `json:"total"`
	Success    int                           `json:"success"`
	Failed     int                           `json:"failed"`
	Skipped    int                           `json:"skipped"`
	StartedAt  string                        `json:"started_at"`
	FinishedAt string                        `json:"finished_at,omitempty"`
	Logs       []openAIAutoReauthJobLogEntry `json:"logs"`
	Errors     []openAIAutoReauthJobIssue    `json:"errors"`
	Warnings   []openAIAutoReauthJobIssue    `json:"warnings"`
}

type openAIAutoReauthJob struct {
	mu         sync.Mutex
	ID         string
	Status     openAIAutoReauthJobStatus
	Total      int
	Success    int
	Failed     int
	Skipped    int
	StartedAt  time.Time
	FinishedAt time.Time
	logSeq     int64
	Logs       []openAIAutoReauthJobLogEntry
	Errors     []openAIAutoReauthJobIssue
	Warnings   []openAIAutoReauthJobIssue
}

func newOpenAIAutoReauthJob(total int) *openAIAutoReauthJob {
	return &openAIAutoReauthJob{
		ID:        uuid.NewString(),
		Status:    openAIAutoReauthJobStatusRunning,
		Total:     total,
		StartedAt: time.Now(),
		Logs:      make([]openAIAutoReauthJobLogEntry, 0, 64),
		Errors:    make([]openAIAutoReauthJobIssue, 0),
		Warnings:  make([]openAIAutoReauthJobIssue, 0),
	}
}

func (j *openAIAutoReauthJob) appendLog(level, message string, accountID *int64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.logSeq++
	j.Logs = append(j.Logs, openAIAutoReauthJobLogEntry{
		Seq:       j.logSeq,
		At:        time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		AccountID: accountID,
	})
}

func (j *openAIAutoReauthJob) addError(accountID int64, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Errors = append(j.Errors, openAIAutoReauthJobIssue{AccountID: accountID, Message: message})
}

func (j *openAIAutoReauthJob) addWarning(accountID int64, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Warnings = append(j.Warnings, openAIAutoReauthJobIssue{AccountID: accountID, Message: message})
}

func (j *openAIAutoReauthJob) setCounts(success, failed, skipped int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Success = success
	j.Failed = failed
	j.Skipped = skipped
}

func (j *openAIAutoReauthJob) complete() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = openAIAutoReauthJobStatusCompleted
	j.FinishedAt = time.Now()
}

func (j *openAIAutoReauthJob) snapshot(after int64) openAIAutoReauthJobSnapshot {
	j.mu.Lock()
	defer j.mu.Unlock()

	logs := make([]openAIAutoReauthJobLogEntry, 0)
	if after <= 0 {
		logs = append(logs, j.Logs...)
	} else {
		for _, item := range j.Logs {
			if item.Seq > after {
				logs = append(logs, item)
			}
		}
	}

	errorsList := append([]openAIAutoReauthJobIssue(nil), j.Errors...)
	warningsList := append([]openAIAutoReauthJobIssue(nil), j.Warnings...)

	snapshot := openAIAutoReauthJobSnapshot{
		JobID:     j.ID,
		Status:    j.Status,
		Total:     j.Total,
		Success:   j.Success,
		Failed:    j.Failed,
		Skipped:   j.Skipped,
		StartedAt: j.StartedAt.Format(time.RFC3339),
		Logs:      logs,
		Errors:    errorsList,
		Warnings:  warningsList,
	}
	if !j.FinishedAt.IsZero() {
		snapshot.FinishedAt = j.FinishedAt.Format(time.RFC3339)
	}
	return snapshot
}

type openAIAutoReauthJobStore struct {
	mu   sync.Mutex
	jobs map[string]*openAIAutoReauthJob
}

func newOpenAIAutoReauthJobStore() *openAIAutoReauthJobStore {
	return &openAIAutoReauthJobStore{
		jobs: make(map[string]*openAIAutoReauthJob),
	}
}

func (s *openAIAutoReauthJobStore) create(total int) *openAIAutoReauthJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	job := newOpenAIAutoReauthJob(total)
	s.jobs[job.ID] = job
	return job
}

func (s *openAIAutoReauthJobStore) get(id string) (*openAIAutoReauthJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()
	job, ok := s.jobs[id]
	return job, ok
}

func (s *openAIAutoReauthJobStore) cleanupLocked() {
	cutoff := time.Now().Add(-2 * time.Hour)
	for id, job := range s.jobs {
		job.mu.Lock()
		removable := job.Status == openAIAutoReauthJobStatusCompleted && !job.FinishedAt.IsZero() && job.FinishedAt.Before(cutoff)
		job.mu.Unlock()
		if removable {
			delete(s.jobs, id)
		}
	}
}

var openAIAutoReauthJobs = newOpenAIAutoReauthJobStore()
