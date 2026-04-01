-- Add plain_lyrics column to songs table for faster/cleaner searching
ALTER TABLE songs ADD COLUMN IF NOT EXISTS plain_lyrics TEXT;

-- Create index for full-text search on plain lyrics (if not exists)
CREATE INDEX IF NOT EXISTS idx_songs_plain_lyrics ON songs USING gin(to_tsvector('simple', COALESCE(plain_lyrics, '')));
