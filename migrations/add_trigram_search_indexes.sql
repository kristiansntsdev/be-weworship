-- Improve PostgreSQL contains searches used by ILIKE '%text%' filters.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_users_username_trgm
    ON users USING gin (username gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_users_email_trgm
    ON users USING gin (email gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_tags_name_trgm
    ON tags USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_tags_description_trgm
    ON tags USING gin (description gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_songs_title_trgm
    ON songs USING gin (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_songs_artist_trgm
    ON songs USING gin (artist gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_songs_base_chord_trgm
    ON songs USING gin (base_chord gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_songs_lyrics_search_trgm
    ON songs USING gin ((COALESCE(plain_lyrics, lyrics_and_chords)) gin_trgm_ops);
