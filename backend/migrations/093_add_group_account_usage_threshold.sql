-- 093_add_group_account_usage_threshold.sql
-- Add per-group account usage threshold percent for scheduling control.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS account_usage_threshold_percent DECIMAL(10,4);
