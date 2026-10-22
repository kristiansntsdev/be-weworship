-- Semantic search: one embedding per song chunk (chunk 0 = title + artist + full lyrics, 1.. = stanzas).
-- Requires the pgvector extension on the database server.
-- Vectors come from bge-m3 (1024 dims). Changing the model means re-running the backfill with --force.
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS song_embeddings (
    id           SERIAL PRIMARY KEY,
    song_id      INTEGER      NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    chunk_index  INTEGER      NOT NULL,
    chunk_text   TEXT         NOT NULL,
    embedding    vector(1024) NOT NULL,
    model        VARCHAR(100) NOT NULL,
    content_hash VARCHAR(64)  NOT NULL,   -- sha256 of model + title + artist + plain lyrics; unchanged hash = skip re-embedding
    "createdAt"  TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_song_embeddings_chunk UNIQUE (song_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_song_embeddings_song ON song_embeddings (song_id);

-- No ANN index yet: at a few thousand chunks an exact scan takes milliseconds. Add
-- `USING hnsw (embedding vector_cosine_ops)` once the catalog grows into tens of thousands of songs.
