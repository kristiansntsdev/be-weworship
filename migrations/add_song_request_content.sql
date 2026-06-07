ALTER TABLE song_requests
  ADD COLUMN IF NOT EXISTS lyrics_type VARCHAR(20) NOT NULL DEFAULT 'lyrics',
  ADD COLUMN IF NOT EXISTS lyrics TEXT NOT NULL DEFAULT '';

ALTER TABLE song_requests
  DROP CONSTRAINT IF EXISTS song_requests_lyrics_type_check;

ALTER TABLE song_requests
  ADD CONSTRAINT song_requests_lyrics_type_check
  CHECK (lyrics_type IN ('lyrics', 'lyrics_chords'));
