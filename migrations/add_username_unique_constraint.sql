-- Migration: enforce unique + length constraint on users.username
-- Run after rename_users_name_to_username.sql

-- Shrink the column to match the 30-char app-level limit
ALTER TABLE users ALTER COLUMN username TYPE VARCHAR(30);

-- Add unique constraint (the index name is used to detect conflicts in Go)
ALTER TABLE users ADD CONSTRAINT users_username_unique UNIQUE (username);
