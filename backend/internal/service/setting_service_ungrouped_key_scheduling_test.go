//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type ungroupedKeySchedulingRepoStub struct {
	getValueFn func(ctx context.Context, key string) (string, error)
	calls      int
}

func (s *ungroupedKeySchedulingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *ungroupedKeySchedulingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	s.calls++
	if s.getValueFn == nil {
		panic("unexpected GetValue call")
	}
	return s.getValueFn(ctx, key)
}

func (s *ungroupedKeySchedulingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *ungroupedKeySchedulingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *ungroupedKeySchedulingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *ungroupedKeySchedulingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *ungroupedKeySchedulingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type ungroupedKeySchedulingUpdateRepoStub struct {
	updates map[string]string
}

func (s *ungroupedKeySchedulingUpdateRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *ungroupedKeySchedulingUpdateRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *ungroupedKeySchedulingUpdateRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *ungroupedKeySchedulingUpdateRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *ungroupedKeySchedulingUpdateRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for k, v := range settings {
		s.updates[k] = v
	}
	return nil
}

func (s *ungroupedKeySchedulingUpdateRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *ungroupedKeySchedulingUpdateRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func resetUngroupedKeySchedulingTestCache(t *testing.T) {
	t.Helper()

	ungroupedKeySchedulingCache.Store((*cachedUngroupedKeyScheduling)(nil))
	t.Cleanup(func() {
		ungroupedKeySchedulingCache.Store((*cachedUngroupedKeyScheduling)(nil))
	})
}

func TestIsUngroupedKeySchedulingAllowed_ReturnsTrue(t *testing.T) {
	resetUngroupedKeySchedulingTestCache(t)

	repo := &ungroupedKeySchedulingRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyAllowUngroupedKeyScheduling, key)
			return "true", nil
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.IsUngroupedKeySchedulingAllowed(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsUngroupedKeySchedulingAllowed_ReturnsFalseOnNotFound(t *testing.T) {
	resetUngroupedKeySchedulingTestCache(t)

	repo := &ungroupedKeySchedulingRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyAllowUngroupedKeyScheduling, key)
			return "", ErrSettingNotFound
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.False(t, svc.IsUngroupedKeySchedulingAllowed(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsUngroupedKeySchedulingAllowed_ReturnsFalseOnDBError(t *testing.T) {
	resetUngroupedKeySchedulingTestCache(t)

	repo := &ungroupedKeySchedulingRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyAllowUngroupedKeyScheduling, key)
			return "", errors.New("db down")
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.False(t, svc.IsUngroupedKeySchedulingAllowed(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsUngroupedKeySchedulingAllowed_CachesResult(t *testing.T) {
	resetUngroupedKeySchedulingTestCache(t)

	repo := &ungroupedKeySchedulingRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyAllowUngroupedKeyScheduling, key)
			return "true", nil
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.IsUngroupedKeySchedulingAllowed(context.Background()))
	require.True(t, svc.IsUngroupedKeySchedulingAllowed(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestUpdateSettings_InvalidatesUngroupedKeySchedulingCache(t *testing.T) {
	resetUngroupedKeySchedulingTestCache(t)

	ungroupedKeySchedulingCache.Store(&cachedUngroupedKeyScheduling{
		value:     true,
		expiresAt: time.Now().Add(ungroupedKeySchedulingCacheTTL).UnixNano(),
	})

	repo := &ungroupedKeySchedulingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AllowUngroupedKeyScheduling: false,
	})
	require.NoError(t, err)
	require.Equal(t, "false", repo.updates[SettingKeyAllowUngroupedKeyScheduling])
	require.False(t, svc.IsUngroupedKeySchedulingAllowed(context.Background()))
}
