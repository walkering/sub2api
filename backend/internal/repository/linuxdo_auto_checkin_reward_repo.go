package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type execContextRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type linuxDoAutoCheckinRewardRepository struct {
	client *dbent.Client
	sqlDB  *sql.DB
}

func NewLinuxDoAutoCheckinRewardRepository(client *dbent.Client, sqlDB *sql.DB) service.LinuxDoAutoCheckinRewardRepository {
	return &linuxDoAutoCheckinRewardRepository{
		client: client,
		sqlDB:  sqlDB,
	}
}

func (r *linuxDoAutoCheckinRewardRepository) Create(ctx context.Context, input service.CreateLinuxDoAutoCheckinRewardInput) error {
	var executor execContextRunner = r.sqlDB
	if client := clientFromContext(ctx, r.client); client != nil {
		executor = client
	}

	_, err := executor.ExecContext(
		ctx,
		`INSERT INTO linuxdo_auto_checkin_rewards (user_id, reward_date, source, bonus_amount)
		 VALUES ($1, $2, $3, $4)`,
		input.UserID,
		input.RewardDate,
		input.Source,
		input.BonusAmount,
	)
	if isUniqueConstraintViolation(err) {
		return service.ErrLinuxDoAutoCheckinRewardAlreadyGranted
	}
	return err
}
