# be-songbanks-v1 — CLAUDE.md

> Claude reads this file on every message. Keep it lean — every line costs tokens.
> Global rules (architecture, conventions, file map protocol) live in root CLAUDE.md.

## Project Overview

- **Purpose**: REST API for the WeWorship church song management platform
- **Stack**: Go, Fiber, pgx (PostgreSQL), go-redis
- **Entry points**: `api/index.go` (route registration), `vercel.json` (Vercel serverless deployment)
- **Data layer**: PostgreSQL (pgx/sqlx), Redis (go-redis)

## Architecture Rules

> Add project-specific rules here. Global rules are in root CLAUDE.md.

1. TODO

## Conventions

> Add project-specific conventions here. Global conventions are in root CLAUDE.md.

- Layered structure: `handlers/` → `services/` → `repositories/` → `models/`
- TODO

## File Map

Global db: `/Users/rpay/.claude/file-map.db` — see root CLAUDE.md for full query reference.

Project ID for be-songbanks-v1: `5`

```bash
# Search files
sqlite3 /Users/rpay/.claude/file-map.db \
  "SELECT key, path, description, exports FROM files
   WHERE project_id = 5
   AND (description LIKE '%<keyword>%' OR path LIKE '%<keyword>%')"

# Log a change
sqlite3 /Users/rpay/.claude/file-map.db \
  "INSERT INTO updates (project_id, file_key, datetime, changes)
   VALUES (5, '<key>', '<YYYY/MM/DD HH:mm>', '<what changed>')"
```
