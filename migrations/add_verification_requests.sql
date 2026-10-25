CREATE TABLE IF NOT EXISTS verification_requests (
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(120) NOT NULL,
    proof_link   TEXT         NOT NULL,
    note         TEXT,
    status       VARCHAR(20)  NOT NULL DEFAULT 'pending',
    admin_notes  TEXT,
    "createdAt"  TIMESTAMP    NOT NULL DEFAULT NOW(),
    "updatedAt"  TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT verification_requests_status_check CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT verification_requests_display_name_check CHECK (length(trim(display_name)) > 0),
    CONSTRAINT verification_requests_proof_link_check CHECK (proof_link ~* '^https?://')
);

CREATE INDEX IF NOT EXISTS idx_verification_requests_user_id ON verification_requests (user_id);
CREATE INDEX IF NOT EXISTS idx_verification_requests_status ON verification_requests (status);

-- At most one open (pending) request per user.
CREATE UNIQUE INDEX IF NOT EXISTS idx_verification_requests_one_pending
    ON verification_requests (user_id)
    WHERE status = 'pending';
