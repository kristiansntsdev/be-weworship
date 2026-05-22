-- Account deletion requests (required for Google Play data deletion policy)
CREATE TABLE IF NOT EXISTS deletion_requests (
    id          SERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL,
    user_id     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending',  -- 'pending' | 'processed'
    ip_address  VARCHAR(45),
    "createdAt" TIMESTAMP    NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deletion_requests_email ON deletion_requests(email);
