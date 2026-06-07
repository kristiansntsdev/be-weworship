CREATE TABLE IF NOT EXISTS song_reports (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id      INTEGER      NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    report_type  VARCHAR(20)  NOT NULL,
    description  TEXT         NOT NULL,
    evidence_url TEXT         NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending',
    admin_notes  TEXT,
    "createdAt"  TIMESTAMP    NOT NULL DEFAULT NOW(),
    "updatedAt"  TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT song_reports_report_type_check CHECK (report_type IN ('lyrics', 'chord', 'other')),
    CONSTRAINT song_reports_status_check CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT song_reports_description_check CHECK (length(trim(description)) > 0),
    CONSTRAINT song_reports_evidence_url_check CHECK (evidence_url ~* '^https?://')
);

CREATE INDEX IF NOT EXISTS idx_song_reports_user_id ON song_reports(user_id);
CREATE INDEX IF NOT EXISTS idx_song_reports_song_id ON song_reports(song_id);
CREATE INDEX IF NOT EXISTS idx_song_reports_status ON song_reports(status);
