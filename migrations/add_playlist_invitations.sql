-- playlist_invitations: per-playlist direct invitations sent by leaders/co-leads
CREATE TABLE IF NOT EXISTS playlist_invitations (
    id           SERIAL PRIMARY KEY,
    playlist_id  INTEGER     NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    inviter_id   INTEGER     NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
    invitee_id   INTEGER     NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    "createdAt"  TIMESTAMP   NOT NULL DEFAULT NOW(),
    "updatedAt"  TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- Partial unique: allows re-inviting after decline, blocks duplicate pending invites
CREATE UNIQUE INDEX IF NOT EXISTS uq_pending_invitation
    ON playlist_invitations (playlist_id, invitee_id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_invitations_invitee  ON playlist_invitations (invitee_id, status);
CREATE INDEX IF NOT EXISTS idx_invitations_playlist ON playlist_invitations (playlist_id);

-- Speed up the ww@ username search
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
