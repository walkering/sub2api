DO $$
DECLARE
    duplicate_names TEXT;
BEGIN
    SELECT string_agg(format('%s (count=%s)', name, duplicate_count), ', ' ORDER BY name)
      INTO duplicate_names
    FROM (
        SELECT name, COUNT(*) AS duplicate_count
        FROM accounts
        WHERE deleted_at IS NULL
        GROUP BY name
        HAVING COUNT(*) > 1
        ORDER BY name
        LIMIT 10
    ) duplicates;

    IF duplicate_names IS NOT NULL THEN
        RAISE EXCEPTION 'duplicate active account names block unique index creation: %', duplicate_names;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_name_unique
ON accounts(name)
WHERE deleted_at IS NULL;
