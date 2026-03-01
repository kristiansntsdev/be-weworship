-- PostgreSQL schema for be-weworship (public rewrite)
-- Run once to initialize the database. Safe to re-run (IF NOT EXISTS).

-- Unified users table (replaces peserta + pengurus)
CREATE TABLE IF NOT EXISTS users (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(255)        NOT NULL,
    email        VARCHAR(255) UNIQUE NOT NULL,
    password     TEXT,                              -- null for OAuth-only users
    avatar_url   TEXT,
    role         VARCHAR(20)         NOT NULL DEFAULT 'user',     -- 'user' | 'admin'
    provider     VARCHAR(20)         NOT NULL DEFAULT 'local',    -- 'local' | 'google'
    provider_id  TEXT,                              -- Google sub ID
    verified     BOOLEAN             NOT NULL DEFAULT FALSE,
    status       VARCHAR(20)         NOT NULL DEFAULT 'active',   -- 'active' | 'inactive' | 'banned'
    "createdAt"  TIMESTAMP           NOT NULL DEFAULT NOW(),
    "updatedAt"  TIMESTAMP           NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS songs (
    id                SERIAL PRIMARY KEY,
    slug              VARCHAR(300) UNIQUE,
    title             VARCHAR(255)  NOT NULL,
    artist            TEXT,
    base_chord        VARCHAR(10),
    lyrics_and_chords TEXT,
    external_links    JSONB,                          -- {spotify: "url", youtube: "url"}
    dmca_takedown     BOOLEAN       NOT NULL DEFAULT FALSE,
    dmca_status_notes TEXT,
    created_by        INTEGER REFERENCES users(id) ON DELETE SET NULL,
    "createdAt"       TIMESTAMP     NOT NULL DEFAULT NOW(),
    "updatedAt"       TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tags (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    "createdAt" TIMESTAMP           NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMP           NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS song_tags (
    song_id     INTEGER     NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    tag_id      INTEGER     NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    "createdAt" TIMESTAMP   NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMP   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (song_id, tag_id)
);

CREATE TABLE IF NOT EXISTS playlists (
    id               SERIAL PRIMARY KEY,
    playlist_name    VARCHAR(255)  NOT NULL,
    user_id          INTEGER       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    songs            TEXT,
    playlist_notes   TEXT,
    sharable_link    TEXT,
    share_token      VARCHAR(100),
    playlist_team_id INTEGER,
    is_shared        SMALLINT      NOT NULL DEFAULT 0,
    is_locked        SMALLINT      NOT NULL DEFAULT 0,
    "createdAt"      TIMESTAMP     NOT NULL DEFAULT NOW(),
    "updatedAt"      TIMESTAMP     NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS playlist_teams (
    id          SERIAL PRIMARY KEY,
    playlist_id INTEGER   NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    lead_id     INTEGER   NOT NULL REFERENCES users(id),
    members     TEXT,
    "createdAt" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ── Analytics ────────────────────────────────────────────────────────────────

-- Song interaction events
CREATE TABLE IF NOT EXISTS song_events (
    id          SERIAL PRIMARY KEY,
    song_id     INTEGER,                             -- nullable: keep data if song deleted
    user_id     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    event_type  VARCHAR(30) NOT NULL,                -- 'view' | 'play' | 'add_to_playlist' | 'share'
    platform    VARCHAR(20) NOT NULL DEFAULT 'unknown', -- 'mobile' | 'web' | 'unknown'
    duration_ms INTEGER,                             -- for 'play' events
    "createdAt" TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- Search query logs
CREATE TABLE IF NOT EXISTS search_logs (
    id             SERIAL PRIMARY KEY,
    user_id        INTEGER REFERENCES users(id) ON DELETE SET NULL,
    query          TEXT        NOT NULL,
    filters        JSONB,                            -- {base_chord, tag_ids}
    results_count  INTEGER     NOT NULL DEFAULT 0,
    platform       VARCHAR(20) NOT NULL DEFAULT 'unknown',
    "createdAt"    TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- App sessions for DAU / MAU
CREATE TABLE IF NOT EXISTS app_sessions (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    platform    VARCHAR(20) NOT NULL DEFAULT 'unknown', -- 'mobile' | 'web'
    app_version VARCHAR(50),
    device_os   VARCHAR(50),
    "createdAt" TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- Performance metrics (API response times + client-side screen/song load)
CREATE TABLE IF NOT EXISTS performance_logs (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER REFERENCES users(id) ON DELETE SET NULL,
    platform     VARCHAR(20)  NOT NULL DEFAULT 'unknown',
    metric_type  VARCHAR(50)  NOT NULL,              -- 'api_response' | 'screen_load' | 'song_load' | 'search_response'
    endpoint     VARCHAR(255),                        -- for 'api_response'
    screen_name  VARCHAR(100),                        -- for 'screen_load'
    duration_ms  INTEGER      NOT NULL,
    status_code  INTEGER,                             -- for API metrics
    app_version  VARCHAR(50),
    device_os    VARCHAR(50),
    "createdAt"  TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- ── Indexes ───────────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_users_email              ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role               ON users(role);
CREATE INDEX IF NOT EXISTS idx_playlists_user_id        ON playlists(user_id);
CREATE INDEX IF NOT EXISTS idx_playlists_share_token    ON playlists(share_token);
CREATE INDEX IF NOT EXISTS idx_playlist_teams_playlist  ON playlist_teams(playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_teams_lead      ON playlist_teams(lead_id);
CREATE INDEX IF NOT EXISTS idx_songs_title              ON songs(title);
CREATE INDEX IF NOT EXISTS idx_songs_artist             ON songs(artist);
CREATE INDEX IF NOT EXISTS idx_song_events_song_id      ON song_events(song_id);
CREATE INDEX IF NOT EXISTS idx_song_events_created      ON song_events("createdAt");
CREATE INDEX IF NOT EXISTS idx_search_logs_created      ON search_logs("createdAt");
CREATE INDEX IF NOT EXISTS idx_app_sessions_created     ON app_sessions("createdAt");
CREATE INDEX IF NOT EXISTS idx_perf_logs_metric_type    ON performance_logs(metric_type);
CREATE INDEX IF NOT EXISTS idx_perf_logs_created        ON performance_logs("createdAt");
