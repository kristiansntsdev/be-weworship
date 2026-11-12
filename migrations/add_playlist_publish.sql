-- Publish a playlist to the web: a random public token (separate from the team-invite share_token) and the
-- display mode the owner chose. The public page is served only while is_public = 1.
ALTER TABLE playlists ADD COLUMN IF NOT EXISTS public_token VARCHAR(32);
ALTER TABLE playlists ADD COLUMN IF NOT EXISTS public_view_mode VARCHAR(16) NOT NULL DEFAULT 'lyrics_chords';
CREATE UNIQUE INDEX IF NOT EXISTS idx_playlists_public_token ON playlists(public_token);
