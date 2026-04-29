package service

import (
	"math"
	"testing"
	"time"
)

func TestGetMaxQuotaUsageRatio(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		extra     map[string]any
		wantRatio float64
		wantOK    bool
	}{
		{
			name: "max ratio across total daily weekly",
			extra: map[string]any{
				"quota_limit":        100.0,
				"quota_used":         60.0,
				"quota_daily_limit":  20.0,
				"quota_daily_used":   19.0,
				"quota_daily_start":  now.Add(-time.Hour).Format(time.RFC3339),
				"quota_weekly_limit": 50.0,
				"quota_weekly_used":  10.0,
				"quota_weekly_start": now.Add(-2 * time.Hour).Format(time.RFC3339),
			},
			wantRatio: 0.95,
			wantOK:    true,
		},
		{
			name: "expired daily period treated as zero",
			extra: map[string]any{
				"quota_limit":       100.0,
				"quota_used":        40.0,
				"quota_daily_limit": 20.0,
				"quota_daily_used":  19.0,
				"quota_daily_start": now.Add(-25 * time.Hour).Format(time.RFC3339),
			},
			wantRatio: 0.4,
			wantOK:    true,
		},
		{
			name: "expired weekly period treated as zero",
			extra: map[string]any{
				"quota_weekly_limit": 20.0,
				"quota_weekly_used":  19.0,
				"quota_weekly_start": now.Add(-(8 * 24 * time.Hour)).Format(time.RFC3339),
			},
			wantRatio: 0,
			wantOK:    true,
		},
		{
			name: "no quota limits not controlled",
			extra: map[string]any{
				"quota_used":        60.0,
				"quota_daily_used":  19.0,
				"quota_weekly_used": 10.0,
			},
			wantRatio: 0,
			wantOK:    false,
		},
		{
			name: "fixed reset mode also respects expiration",
			extra: map[string]any{
				"quota_daily_limit":      20.0,
				"quota_daily_used":       18.0,
				"quota_daily_reset_mode": "fixed",
				"quota_daily_reset_hour": now.Hour(),
				"quota_reset_timezone":   "UTC",
				"quota_daily_start":      now.Add(-25 * time.Hour).Format(time.RFC3339),
			},
			wantRatio: 0,
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{Extra: tt.extra}
			gotRatio, gotOK := account.GetMaxQuotaUsageRatio()
			if gotOK != tt.wantOK {
				t.Fatalf("GetMaxQuotaUsageRatio() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if math.Abs(gotRatio-tt.wantRatio) > 1e-9 {
				t.Fatalf("GetMaxQuotaUsageRatio() ratio = %v, want %v", gotRatio, tt.wantRatio)
			}
		})
	}
}

func TestCheckQuotaUsageThresholdSchedulability(t *testing.T) {
	account := &Account{
		Extra: map[string]any{
			"quota_limit": 100.0,
			"quota_used":  95.0,
		},
	}

	if got := account.CheckQuotaUsageThresholdSchedulability(95, 0); got != WindowCostNotSchedulable {
		t.Fatalf("CheckQuotaUsageThresholdSchedulability() = %v, want %v", got, WindowCostNotSchedulable)
	}

	if got := account.CheckQuotaUsageThresholdSchedulability(95, 2); got != WindowCostStickyOnly {
		t.Fatalf("CheckQuotaUsageThresholdSchedulability() with active sessions = %v, want %v", got, WindowCostStickyOnly)
	}

	if got := account.CheckQuotaUsageThresholdSchedulability(96, 0); got != WindowCostSchedulable {
		t.Fatalf("CheckQuotaUsageThresholdSchedulability() below threshold = %v, want %v", got, WindowCostSchedulable)
	}

	unlimited := &Account{Extra: map[string]any{"quota_used": 999.0}}
	if got := unlimited.CheckQuotaUsageThresholdSchedulability(10, 0); got != WindowCostSchedulable {
		t.Fatalf("CheckQuotaUsageThresholdSchedulability() unlimited = %v, want %v", got, WindowCostSchedulable)
	}
}
