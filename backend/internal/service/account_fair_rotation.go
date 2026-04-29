package service

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type fairRotationCursor struct {
	mu   sync.Mutex
	next int
}

var fairRotationCursors sync.Map

func isAccountFairRotationEnabled(settingService *SettingService, ctx context.Context) bool {
	return settingService != nil && settingService.IsAccountFairRotationEnabled(ctx)
}

func buildFairRotationScope(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "|")
}

func fairRotationStart(scope string, size int) int {
	if scope == "" || size <= 1 {
		return 0
	}
	raw, _ := fairRotationCursors.LoadOrStore(scope, &fairRotationCursor{})
	cursor := raw.(*fairRotationCursor)
	cursor.mu.Lock()
	defer cursor.mu.Unlock()

	start := cursor.next % size
	cursor.next = (start + 1) % size
	return start
}

func fairRotationOrderAccounts(scope string, candidates []*Account, preferOAuth bool) []*Account {
	if len(candidates) == 0 {
		return nil
	}

	ordered := append([]*Account(nil), candidates...)
	stableSortAccountsByPriorityAndLastUsed(ordered, preferOAuth)
	if len(ordered) <= 1 {
		return ordered
	}

	topPriority := ordered[0].Priority
	topLen := 1
	for topLen < len(ordered) && ordered[topLen].Priority == topPriority {
		topLen++
	}

	return rotateAccountSlicePrefix(ordered, topLen, fairRotationStart(scope, topLen))
}

func fairRotationOrderAccountLoads(scope string, candidates []accountWithLoad, preferOAuth bool) []accountWithLoad {
	if len(candidates) == 0 {
		return nil
	}

	ordered := append([]accountWithLoad(nil), candidates...)
	stableSortAccountLoadsByPriorityLoadAndLastUsed(ordered, preferOAuth)
	if len(ordered) <= 1 {
		return ordered
	}

	topPriority := ordered[0].account.Priority
	topLoadRate := ordered[0].loadInfo.LoadRate
	topLen := 1
	for topLen < len(ordered) &&
		ordered[topLen].account.Priority == topPriority &&
		ordered[topLen].loadInfo.LoadRate == topLoadRate {
		topLen++
	}

	return rotateAccountLoadSlicePrefix(ordered, topLen, fairRotationStart(scope, topLen))
}

func stableSortAccountsByPriorityAndLastUsed(accounts []*Account, preferOAuth bool) {
	sort.SliceStable(accounts, func(i, j int) bool {
		return compareAccountsByPriorityAndLastUsed(accounts[i], accounts[j], preferOAuth) < 0
	})
}

func stableSortAccountLoadsByPriorityLoadAndLastUsed(accounts []accountWithLoad, preferOAuth bool) {
	sort.SliceStable(accounts, func(i, j int) bool {
		a, b := accounts[i], accounts[j]
		if a.account.Priority != b.account.Priority {
			return a.account.Priority < b.account.Priority
		}
		if a.loadInfo.LoadRate != b.loadInfo.LoadRate {
			return a.loadInfo.LoadRate < b.loadInfo.LoadRate
		}
		return compareAccountsByPriorityAndLastUsed(a.account, b.account, preferOAuth) < 0
	})
}

func compareAccountsByPriorityAndLastUsed(a, b *Account, preferOAuth bool) int {
	if a.Priority != b.Priority {
		if a.Priority < b.Priority {
			return -1
		}
		return 1
	}

	switch {
	case a.LastUsedAt == nil && b.LastUsedAt != nil:
		return -1
	case a.LastUsedAt != nil && b.LastUsedAt == nil:
		return 1
	case a.LastUsedAt == nil && b.LastUsedAt == nil:
		if preferOAuth && a.Type != b.Type {
			if a.Type == AccountTypeOAuth {
				return -1
			}
			if b.Type == AccountTypeOAuth {
				return 1
			}
		}
	default:
		if a.LastUsedAt.Before(*b.LastUsedAt) {
			return -1
		}
		if b.LastUsedAt.Before(*a.LastUsedAt) {
			return 1
		}
	}

	if a.ID < b.ID {
		return -1
	}
	if a.ID > b.ID {
		return 1
	}
	return 0
}

func rotateAccountSlicePrefix(accounts []*Account, prefixLen, start int) []*Account {
	if prefixLen <= 1 || start == 0 {
		return accounts
	}
	rotated := append([]*Account(nil), accounts[:start]...)
	return append(append([]*Account(nil), accounts[start:prefixLen]...), append(rotated, accounts[prefixLen:]...)...)
}

func rotateAccountLoadSlicePrefix(accounts []accountWithLoad, prefixLen, start int) []accountWithLoad {
	if prefixLen <= 1 || start == 0 {
		return accounts
	}
	rotated := append([]accountWithLoad(nil), accounts[:start]...)
	return append(append([]accountWithLoad(nil), accounts[start:prefixLen]...), append(rotated, accounts[prefixLen:]...)...)
}
