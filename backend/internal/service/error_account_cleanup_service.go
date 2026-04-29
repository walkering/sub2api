package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const errorAccountCleanupPageSize = 100

// ErrorAccountCleanupService periodically deletes error accounts.
type ErrorAccountCleanupService struct {
	accountRepo AccountRepository
	interval    time.Duration
	stopCh      chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewErrorAccountCleanupService(accountRepo AccountRepository, interval time.Duration) *ErrorAccountCleanupService {
	return &ErrorAccountCleanupService{
		accountRepo: accountRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

func (s *ErrorAccountCleanupService) Start() {
	if s == nil || s.accountRepo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *ErrorAccountCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *ErrorAccountCleanupService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	page := 1
	deletedAccounts := 0

	for {
		accounts, pager, err := s.accountRepo.ListWithFilters(
			ctx,
			pagination.PaginationParams{Page: page, PageSize: errorAccountCleanupPageSize},
			"",
			"",
			StatusError,
			"",
			0,
			"",
			"",
		)
		if err != nil {
			log.Printf("[ErrorAccountCleanup] List error accounts failed: %v", err)
			return
		}
		if len(accounts) == 0 {
			break
		}

		for i := range accounts {
			account := accounts[i]
			if err := s.accountRepo.Delete(ctx, account.ID); err != nil {
				log.Printf("[ErrorAccountCleanup] Delete account=%d failed: %v", account.ID, err)
				continue
			}
			deletedAccounts++
		}

		if pager == nil || page >= pager.Pages {
			break
		}
		page++
	}

	if deletedAccounts > 0 {
		log.Printf("[ErrorAccountCleanup] Deleted %d error accounts", deletedAccounts)
	}
}
