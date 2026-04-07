package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAccountMixedChannelRouter(adminSvc *stubAdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/check-mixed-channel", accountHandler.CheckMixedChannel)
	router.POST("/api/v1/admin/accounts", accountHandler.Create)
	router.PUT("/api/v1/admin/accounts/:id", accountHandler.Update)
	router.POST("/api/v1/admin/accounts/bulk-update", accountHandler.BulkUpdate)
	router.POST("/api/v1/admin/accounts/batch-clear-error", accountHandler.BatchClearError)
	router.POST("/api/v1/admin/accounts/batch-refresh", accountHandler.BatchRefresh)
	router.POST("/api/v1/admin/accounts/group-transfer", accountHandler.TransferAccountsByGroup)
	return router
}

func TestAccountHandlerCheckMixedChannelNoRisk(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"platform":  "antigravity",
		"group_ids": []int64{27},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/check-mixed-channel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, data["has_risk"])
	require.Equal(t, int64(0), adminSvc.lastMixedCheck.accountID)
	require.Equal(t, "antigravity", adminSvc.lastMixedCheck.platform)
	require.Equal(t, []int64{27}, adminSvc.lastMixedCheck.groupIDs)
}

func TestAccountHandlerCheckMixedChannelWithRisk(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.checkMixedErr = &service.MixedChannelError{
		GroupID:         27,
		GroupName:       "claude-max",
		CurrentPlatform: "Antigravity",
		OtherPlatform:   "Anthropic",
	}
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"platform":   "antigravity",
		"group_ids":  []int64{27},
		"account_id": 99,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/check-mixed-channel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, data["has_risk"])
	require.Equal(t, "mixed_channel_warning", data["error"])
	details, ok := data["details"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(27), details["group_id"])
	require.Equal(t, "claude-max", details["group_name"])
	require.Equal(t, "Antigravity", details["current_platform"])
	require.Equal(t, "Anthropic", details["other_platform"])
	require.Equal(t, int64(99), adminSvc.lastMixedCheck.accountID)
}

func TestAccountHandlerCreateMixedChannelConflictSimplifiedResponse(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.createAccountErr = &service.MixedChannelError{
		GroupID:         27,
		GroupName:       "claude-max",
		CurrentPlatform: "Antigravity",
		OtherPlatform:   "Anthropic",
	}
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"name":        "ag-oauth-1",
		"platform":    "antigravity",
		"type":        "oauth",
		"credentials": map[string]any{"refresh_token": "rt"},
		"group_ids":   []int64{27},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "mixed_channel_warning", resp["error"])
	require.Contains(t, resp["message"], "mixed_channel_warning")
	_, hasDetails := resp["details"]
	_, hasRequireConfirmation := resp["require_confirmation"]
	require.False(t, hasDetails)
	require.False(t, hasRequireConfirmation)
}

func TestAccountHandlerUpdateMixedChannelConflictSimplifiedResponse(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.updateAccountErr = &service.MixedChannelError{
		GroupID:         27,
		GroupName:       "claude-max",
		CurrentPlatform: "Antigravity",
		OtherPlatform:   "Anthropic",
	}
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"group_ids": []int64{27},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/3", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "mixed_channel_warning", resp["error"])
	require.Contains(t, resp["message"], "mixed_channel_warning")
	_, hasDetails := resp["details"]
	_, hasRequireConfirmation := resp["require_confirmation"]
	require.False(t, hasDetails)
	require.False(t, hasRequireConfirmation)
}

func TestAccountHandlerBulkUpdateMixedChannelConflict(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.bulkUpdateAccountErr = &service.MixedChannelError{
		GroupID:         27,
		GroupName:       "claude-max",
		CurrentPlatform: "Antigravity",
		OtherPlatform:   "Anthropic",
	}
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"account_ids": []int64{1, 2, 3},
		"group_ids":   []int64{27},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "mixed_channel_warning", resp["error"])
	require.Contains(t, resp["message"], "claude-max")
}

func TestAccountHandlerBulkUpdateMixedChannelConfirmSkips(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"account_ids":                []int64{1, 2},
		"group_ids":                  []int64{27},
		"confirm_mixed_channel_risk": true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), data["success"])
	require.Equal(t, float64(0), data["failed"])
}

func TestAccountHandlerBatchClearErrorScopeGroupFilters(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "a1", Status: service.StatusActive, GroupIDs: []int64{10}},
		{ID: 2, Name: "a2", Status: service.StatusActive, GroupIDs: []int64{11}},
		{ID: 3, Name: "a3", Status: service.StatusActive, GroupIDs: []int64{10, 11}},
	}
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"account_ids":    []int64{1, 2, 3},
		"scope_group_id": 10,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-clear-error", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), data["total"])
	require.Equal(t, float64(2), data["success"])
	require.Equal(t, float64(0), data["failed"])
}

func TestAccountHandlerBatchClearErrorScopeGroupEmptyRejected(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "a1", Status: service.StatusActive, GroupIDs: []int64{11}},
	}
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"account_ids":    []int64{1},
		"scope_group_id": 10,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-clear-error", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandlerTransferAccountsByGroupSuccess(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"source_group_id": 10,
		"target_group_id": 20,
		"count":           2,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/group-transfer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), data["moved_count"])
	require.Equal(t, float64(10), data["source_group_id"])
	require.Equal(t, float64(20), data["target_group_id"])
}

func TestAccountHandlerBulkUpdatePassesScopeGroupID(t *testing.T) {
	adminSvc := newStubAdminService()
	router := setupAccountMixedChannelRouter(adminSvc)

	body, _ := json.Marshal(map[string]any{
		"account_ids":    []int64{1, 2},
		"scope_group_id": 10,
		"group_ids":      []int64{20},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, adminSvc.lastBulkUpdateInput)
	require.NotNil(t, adminSvc.lastBulkUpdateInput.ScopeGroupID)
	require.Equal(t, int64(10), *adminSvc.lastBulkUpdateInput.ScopeGroupID)
	require.NotNil(t, adminSvc.lastBulkUpdateInput.GroupIDs)
	require.Equal(t, []int64{20}, *adminSvc.lastBulkUpdateInput.GroupIDs)
}
