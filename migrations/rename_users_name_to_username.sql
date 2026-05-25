-- Migration: rename users.name → users.username
-- Run once. Safe to inspect before applying.

ALTER TABLE users RENAME COLUMN name TO username;
