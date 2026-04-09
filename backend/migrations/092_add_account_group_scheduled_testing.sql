-- 092_add_account_group_scheduled_testing.sql
-- Add group scheduled test plans and background group test jobs/logs.

ALTER TABLE scheduled_test_plans
    ALTER COLUMN account_id DROP NOT NULL;

ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS group_id BIGINT REFERENCES groups(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS batch_size INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS offset_seconds INT NOT NULL DEFAULT 0;

ALTER TABLE scheduled_test_plans
    DROP CONSTRAINT IF EXISTS chk_scheduled_test_plan_target;

ALTER TABLE scheduled_test_plans
    ADD CONSTRAINT chk_scheduled_test_plan_target
    CHECK (num_nonnulls(account_id, group_id) = 1);

CREATE INDEX IF NOT EXISTS idx_stp_group_id ON scheduled_test_plans(group_id) WHERE group_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS account_test_jobs (
    id                  BIGSERIAL PRIMARY KEY,
    group_id            BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    plan_id             BIGINT REFERENCES scheduled_test_plans(id) ON DELETE SET NULL,
    model_id            VARCHAR(100) NOT NULL DEFAULT '',
    trigger_source      VARCHAR(20) NOT NULL DEFAULT 'manual',
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    batch_size          INT NOT NULL DEFAULT 5,
    offset_seconds      INT NOT NULL DEFAULT 0,
    total_accounts      INT NOT NULL DEFAULT 0,
    pending_accounts    INT NOT NULL DEFAULT 0,
    running_accounts    INT NOT NULL DEFAULT 0,
    succeeded_accounts  INT NOT NULL DEFAULT 0,
    failed_accounts     INT NOT NULL DEFAULT 0,
    created_by          BIGINT REFERENCES users(id) ON DELETE SET NULL,
    started_at          TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_test_jobs_group_created
    ON account_test_jobs(group_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_account_test_jobs_status_created
    ON account_test_jobs(status, created_at DESC);

CREATE TABLE IF NOT EXISTS account_test_job_items (
    id              BIGSERIAL PRIMARY KEY,
    job_id           BIGINT NOT NULL REFERENCES account_test_jobs(id) ON DELETE CASCADE,
    account_id       BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    account_name     VARCHAR(255) NOT NULL DEFAULT '',
    scheduled_for    TIMESTAMPTZ NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    response_text    TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    latency_ms       BIGINT NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_test_job_items_job_scheduled
    ON account_test_job_items(job_id, scheduled_for ASC, account_id ASC);

CREATE TABLE IF NOT EXISTS account_test_job_logs (
    id              BIGSERIAL PRIMARY KEY,
    job_id           BIGINT NOT NULL REFERENCES account_test_jobs(id) ON DELETE CASCADE,
    account_id       BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    account_name     VARCHAR(255) NOT NULL DEFAULT '',
    event_type       VARCHAR(32) NOT NULL DEFAULT 'info',
    status           VARCHAR(32) NOT NULL DEFAULT '',
    message          TEXT NOT NULL DEFAULT '',
    response_text    TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    latency_ms       BIGINT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_test_job_logs_job_created
    ON account_test_job_logs(job_id, created_at ASC, id ASC);
