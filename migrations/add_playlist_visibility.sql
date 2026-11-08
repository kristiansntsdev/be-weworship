-- Playlist visibility: 0 = private (default), 1 = public.
-- Existing playlists stay private.
ALTER TABLE playlists ADD COLUMN IF NOT EXISTS is_public SMALLINT NOT NULL DEFAULT 0;
