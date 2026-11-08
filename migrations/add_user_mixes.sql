-- Personal "Mixed for you" mixes and mood mixes, rebuilt at most once a day per user (services/mix_service.go).
-- songs is a JSON snapshot [{id,title,artist,key}] so reading a mix needs no song lookups.
CREATE TABLE IF NOT EXISTS user_mixes (
    user_id       INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mix_key       VARCHAR(40)  NOT NULL,   -- mix-1..mix-3, supermix, mood-<id>
    title         VARCHAR(120) NOT NULL,
    subtitle      TEXT         NOT NULL DEFAULT '',
    songs         TEXT         NOT NULL,
    "generatedAt" TIMESTAMP    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, mix_key)
);
