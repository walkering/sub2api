package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ScheduledTestHandler handles admin scheduled-test-plan and group-test-job management.
type ScheduledTestHandler struct {
	scheduledTestSvc *service.ScheduledTestService
}

// NewScheduledTestHandler creates a new ScheduledTestHandler.
func NewScheduledTestHandler(scheduledTestSvc *service.ScheduledTestService) *ScheduledTestHandler {
	return &ScheduledTestHandler{scheduledTestSvc: scheduledTestSvc}
}

type createScheduledTestPlanRequest struct {
	AccountID      *int64 `json:"account_id"`
	GroupID        *int64 `json:"group_id"`
	ModelID        string `json:"model_id"`
	CronExpression string `json:"cron_expression" binding:"required"`
	Enabled        *bool  `json:"enabled"`
	MaxResults     int    `json:"max_results"`
	AutoRecover    *bool  `json:"auto_recover"`
	BatchSize      int    `json:"batch_size"`
	OffsetSeconds  int    `json:"offset"`
}

type updateScheduledTestPlanRequest struct {
	ModelID        *string `json:"model_id"`
	CronExpression *string `json:"cron_expression"`
	Enabled        *bool   `json:"enabled"`
	MaxResults     *int    `json:"max_results"`
	AutoRecover    *bool   `json:"auto_recover"`
	BatchSize      *int    `json:"batch_size"`
	OffsetSeconds  *int    `json:"offset"`
}

type createAccountTestJobRequest struct {
	ModelID       string `json:"model_id"`
	BatchSize     *int   `json:"batch_size"`
	OffsetSeconds *int   `json:"offset"`
}

// ListByAccount GET /admin/accounts/:id/scheduled-test-plans
func (h *ScheduledTestHandler) ListByAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	plans, err := h.scheduledTestSvc.ListPlansByAccount(c.Request.Context(), accountID)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, plans)
}

// ListByGroup GET /admin/groups/:id/scheduled-test-plans
func (h *ScheduledTestHandler) ListByGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	plans, err := h.scheduledTestSvc.ListPlansByGroup(c.Request.Context(), groupID)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, plans)
}

// Create POST /admin/scheduled-test-plans
func (h *ScheduledTestHandler) Create(c *gin.Context) {
	var req createScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	plan := &service.ScheduledTestPlan{
		AccountID:      req.AccountID,
		GroupID:        req.GroupID,
		ModelID:        req.ModelID,
		CronExpression: req.CronExpression,
		Enabled:        true,
		MaxResults:     req.MaxResults,
		BatchSize:      req.BatchSize,
		OffsetSeconds:  req.OffsetSeconds,
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	if req.AutoRecover != nil {
		plan.AutoRecover = *req.AutoRecover
	}

	created, err := h.scheduledTestSvc.CreatePlan(c.Request.Context(), plan)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, created)
}

// Update PUT /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Update(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	existing, err := h.scheduledTestSvc.GetPlan(c.Request.Context(), planID)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.NotFound(c, "plan not found")
		return
	}

	var req updateScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.ModelID != nil {
		existing.ModelID = *req.ModelID
	}
	if req.CronExpression != nil {
		existing.CronExpression = *req.CronExpression
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.MaxResults != nil {
		existing.MaxResults = *req.MaxResults
	}
	if req.AutoRecover != nil {
		existing.AutoRecover = *req.AutoRecover
	}
	if req.BatchSize != nil {
		existing.BatchSize = *req.BatchSize
	}
	if req.OffsetSeconds != nil {
		existing.OffsetSeconds = *req.OffsetSeconds
	}

	updated, err := h.scheduledTestSvc.UpdatePlan(c.Request.Context(), existing)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, updated)
}

// Delete DELETE /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Delete(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	if err := h.scheduledTestSvc.DeletePlan(c.Request.Context(), planID); err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ListResults GET /admin/scheduled-test-plans/:id/results
func (h *ScheduledTestHandler) ListResults(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	results, err := h.scheduledTestSvc.ListResults(c.Request.Context(), planID, limit)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, results)
}

// CreateJob POST /admin/groups/:id/test-jobs
func (h *ScheduledTestHandler) CreateJob(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	var req createAccountTestJobRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, err.Error())
		return
	}

	var createdBy *int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		createdBy = &subject.UserID
	}

	createReq := service.CreateAccountTestJobRequest{
		GroupID:       groupID,
		ModelID:       req.ModelID,
		TriggerSource: service.ScheduledTestJobTriggerManual,
		CreatedBy:     createdBy,
	}
	if req.BatchSize != nil {
		createReq.BatchSize = *req.BatchSize
	}
	if req.OffsetSeconds != nil {
		createReq.OffsetSeconds = *req.OffsetSeconds
	}

	job, err := h.scheduledTestSvc.CreateJob(c.Request.Context(), createReq)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Accepted(c, job)
}

// ListJobsByGroup GET /admin/groups/:id/test-jobs
func (h *ScheduledTestHandler) ListJobsByGroup(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	limit := 20
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	jobs, err := h.scheduledTestSvc.ListJobsByGroup(c.Request.Context(), groupID, limit)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, jobs)
}

// GetJob GET /admin/test-jobs/:id
func (h *ScheduledTestHandler) GetJob(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	job, err := h.scheduledTestSvc.GetJob(c.Request.Context(), jobID)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.NotFound(c, "job not found")
		return
	}
	response.Success(c, job)
}

// GetJobSnapshot GET /admin/test-jobs/:id/snapshot
func (h *ScheduledTestHandler) GetJobSnapshot(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	logLimit := 200
	if l, err := strconv.Atoi(c.Query("log_limit")); err == nil && l > 0 {
		logLimit = l
	}

	snapshot, err := h.scheduledTestSvc.GetJobSnapshot(c.Request.Context(), jobID, logLimit)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.NotFound(c, "job not found")
		return
	}
	response.Success(c, snapshot)
}

// StreamJobLogs GET /admin/test-jobs/:id/logs/stream
func (h *ScheduledTestHandler) StreamJobLogs(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}

	logLimit := 200
	if l, err := strconv.Atoi(c.Query("log_limit")); err == nil && l > 0 {
		logLimit = l
	}

	initialSnapshot, err := h.scheduledTestSvc.GetJobSnapshot(c.Request.Context(), jobID, logLimit)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return
		}
		response.NotFound(c, "job not found")
		return
	}

	updates, cancel, err := h.scheduledTestSvc.SubscribeJob(jobID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	defer cancel()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	if err := writeSSEJSON(c, gin.H{"type": "snapshot", "snapshot": initialSnapshot}); err != nil {
		return
	}
	if isTerminalJobStatus(initialSnapshot.Job) {
		return
	}

	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case snapshot, ok := <-updates:
			if !ok {
				return
			}
			if err := writeSSEJSON(c, gin.H{"type": "snapshot", "snapshot": snapshot}); err != nil {
				return
			}
			if isTerminalJobStatus(snapshot.Job) {
				return
			}
		case <-pingTicker.C:
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

func writeSSEJSON(c *gin.Context, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func isTerminalJobStatus(job *service.AccountTestJob) bool {
	if job == nil {
		return true
	}
	switch job.Status {
	case service.ScheduledTestJobStatusCompleted, service.ScheduledTestJobStatusFailed, service.ScheduledTestJobStatusNoAccounts:
		return true
	default:
		return false
	}
}
