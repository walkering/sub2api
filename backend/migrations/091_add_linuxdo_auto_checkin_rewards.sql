CREATE TABLE IF NOT EXISTS linuxdo_auto_checkin_rewards (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reward_date DATE NOT NULL,
    source VARCHAR(50) NOT NULL,
    bonus_amount INT NOT NULL CHECK (bonus_amount BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, reward_date, source)
);

CREATE INDEX IF NOT EXISTS idx_linuxdo_auto_checkin_rewards_user_date
    ON linuxdo_auto_checkin_rewards (user_id, reward_date DESC);
