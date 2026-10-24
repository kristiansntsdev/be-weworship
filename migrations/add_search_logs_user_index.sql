-- Speeds up per-user recent song search history lookups.
CREATE INDEX IF NOT EXISTS idx_search_logs_user_created
    ON search_logs(user_id, "createdAt" DESC);
