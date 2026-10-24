-- Per-user starred songs (favorites).
CREATE TABLE IF NOT EXISTS song_favorites (
    user_id     INTEGER   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id     INTEGER   NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    "createdAt" TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, song_id)
);

CREATE INDEX IF NOT EXISTS idx_song_favorites_user_id ON song_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_song_favorites_song_id ON song_favorites(song_id);
