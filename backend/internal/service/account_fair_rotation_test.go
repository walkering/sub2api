//go:build unit

package service

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func resetAccountFairRotationStateForTest() {
	fairRotationCursors = sync.Map{}
	accountFairRotationSF.Forget(SettingKeyAccountFairRotationEnabled)
	accountFairRotationCache.Store(&cachedAccountFairRotation{
		value:     false,
		expiresAt: 0,
	})
}

func newAccountFairRotationSettingServiceForTest(t *testing.T, enabled bool) *SettingService {
	t.Helper()
	resetAccountFairRotationStateForTest()
	t.Cleanup(resetAccountFairRotationStateForTest)

	return NewSettingService(&settingUpdateRepoStub{
		values: map[string]string{
			SettingKeyAccountFairRotationEnabled: strconv.FormatBool(enabled),
		},
	}, &config.Config{})
}

func TestFairRotationOrderAccounts_RotatesTopPriorityOnly(t *testing.T) {
	resetAccountFairRotationStateForTest()

	now := time.Now()
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-time.Hour)
	candidates := []*Account{
		{ID: 101, Priority: 1, LastUsedAt: &earlier},
		{ID: 102, Priority: 1, LastUsedAt: &later},
		{ID: 201, Priority: 2},
	}

	first := fairRotationOrderAccounts("gateway|legacy|test", candidates, false)
	second := fairRotationOrderAccounts("gateway|legacy|test", candidates, false)
	third := fairRotationOrderAccounts("gateway|legacy|test", candidates, false)

	require.Len(t, first, 3)
	require.Equal(t, []int64{101, 102, 201}, []int64{first[0].ID, first[1].ID, first[2].ID})
	require.Equal(t, []int64{102, 101, 201}, []int64{second[0].ID, second[1].ID, second[2].ID})
	require.Equal(t, []int64{101, 102, 201}, []int64{third[0].ID, third[1].ID, third[2].ID})
}

func TestFairRotationOrderAccountLoads_RotatesLowestLoadWithinTopPriority(t *testing.T) {
	resetAccountFairRotationStateForTest()

	earlier := time.Now().Add(-2 * time.Hour)
	later := time.Now().Add(-time.Hour)
	candidates := []accountWithLoad{
		{
			account:  &Account{ID: 1, Priority: 1, LastUsedAt: &earlier},
			loadInfo: &AccountLoadInfo{AccountID: 1, LoadRate: 10},
		},
		{
			account:  &Account{ID: 2, Priority: 1, LastUsedAt: &later},
			loadInfo: &AccountLoadInfo{AccountID: 2, LoadRate: 10},
		},
		{
			account:  &Account{ID: 3, Priority: 1},
			loadInfo: &AccountLoadInfo{AccountID: 3, LoadRate: 30},
		},
		{
			account:  &Account{ID: 4, Priority: 2},
			loadInfo: &AccountLoadInfo{AccountID: 4, LoadRate: 0},
		},
	}

	first := fairRotationOrderAccountLoads("gateway|load|test", candidates, false)
	second := fairRotationOrderAccountLoads("gateway|load|test", candidates, false)

	require.Equal(t, []int64{1, 2, 3, 4}, []int64{
		first[0].account.ID,
		first[1].account.ID,
		first[2].account.ID,
		first[3].account.ID,
	})
	require.Equal(t, []int64{2, 1, 3, 4}, []int64{
		second[0].account.ID,
		second[1].account.ID,
		second[2].account.ID,
		second[3].account.ID,
	})
}

func TestSettingService_IsAccountFairRotationEnabled_UsesCacheAndDefaultsFalse(t *testing.T) {
	resetAccountFairRotationStateForTest()

	repo := &settingUpdateRepoStub{
		values: map[string]string{
			SettingKeyAccountFairRotationEnabled: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.IsAccountFairRotationEnabled(context.Background()))

	repo.values[SettingKeyAccountFairRotationEnabled] = "false"
	require.True(t, svc.IsAccountFairRotationEnabled(context.Background()))

	resetAccountFairRotationStateForTest()
	svc = NewSettingService(&settingUpdateRepoStub{}, &config.Config{})
	require.False(t, svc.IsAccountFairRotationEnabled(context.Background()))
}
