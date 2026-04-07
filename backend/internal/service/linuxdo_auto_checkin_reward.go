package service

import (
	"context"
	"errors"
	"time"
)

const LinuxDoAutoCheckinRewardSourceOAuthLogin = "linuxdo_connect_oauth_login"

var ErrLinuxDoAutoCheckinRewardAlreadyGranted = errors.New("linuxdo auto checkin reward already granted")

type AutoCheckinResult struct {
	Awarded     bool `json:"auto_checkin_awarded"`
	BonusAmount int  `json:"auto_checkin_bonus_amount"`
}

type CreateLinuxDoAutoCheckinRewardInput struct {
	UserID      int64
	RewardDate  time.Time
	Source      string
	BonusAmount int
}

type LinuxDoAutoCheckinRewardRepository interface {
	Create(ctx context.Context, input CreateLinuxDoAutoCheckinRewardInput) error
}
