-- Migration: playlist_members table
-- Adds per-member role assignments (singer / musician / md) for playlist teams.

CREATE TABLE IF NOT EXISTS playlist_members (
    id          SERIAL PRIMARY KEY,
    playlist_id INTEGER     NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    user_id     INTEGER     NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    role        VARCHAR(20) NOT NULL DEFAULT 'singer',   -- 'singer' | 'musician' | 'md'
    joined_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_playlist_members UNIQUE (playlist_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_playlist_members_playlist ON playlist_members (playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_members_user     ON playlist_members (user_id);

-- Backfill from playlist_teams.members TEXT (JSON int array) with role = 'singer'.
-- Handles NULL, empty string, and malformed values gracefully via the CASE.
INSERT INTO playlist_members (playlist_id, user_id, role, joined_at)
SELECT
    pt.playlist_id,
    elem::integer,
    'singer',
    NOW()
FROM playlist_teams pt,
     jsonb_array_elements_text(
         CASE
             WHEN pt.members IS NULL OR TRIM(pt.members) = '' THEN '[]'::jsonb
             ELSE pt.members::jsonb
         END
     ) elem
WHERE TRIM(elem) <> ''
ON CONFLICT (playlist_id, user_id) DO NOTHING;
