package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── BuildSongChunks ───────────────────────────────────────────────────────────

func TestBuildSongChunks_StanzasWithoutLabelsOrRepeats(t *testing.T) {
	lyrics := "Bait :\nBapa Engkau sungguh baik\nKasihMu melimpah di hidupku\n\n" +
		"Reff :\nKunaikkan syukurku buat hari yang Kau beri\n\n" +
		"| • |\n\n" +
		"Reff :\nKunaikkan syukurku buat hari yang Kau beri"

	chunks := BuildSongChunks("Bapa Engkau Sungguh Baik", "Franky Sihombing", lyrics)

	// Labels, the symbol line and the repeated chorus are dropped; the two short stanzas (8 + 7 words)
	// are merged into one chunk because neither reaches minChunkWords on its own.
	require.Len(t, chunks, 2)
	assert.Equal(t, "Bapa Engkau Sungguh Baik\nFranky Sihombing\n\n"+
		"Bapa Engkau sungguh baik\nKasihMu melimpah di hidupku\nKunaikkan syukurku buat hari yang Kau beri", chunks[0])
	assert.Equal(t, "Bapa Engkau sungguh baik\nKasihMu melimpah di hidupku\nKunaikkan syukurku buat hari yang Kau beri", chunks[1])
}

func TestBuildSongChunks_MergesOneLineStanzasAndDropsRepetitiveOnes(t *testing.T) {
	// Blank line between every lyric line, like many songs in the catalog.
	lyrics := "Yesus.. Yesus.. Yesus\n\nYesus.. Yesus..\n\n" +
		"Kaulah Yesus Kaulah Tuhan\n\nHanya Kau kupercaya\n\nKaulah harapan hidupku selamanya"

	chunks := BuildSongChunks("Kubersyukur", "", lyrics)

	require.Len(t, chunks, 2)
	assert.Equal(t, "Yesus.. Yesus.. Yesus\nYesus.. Yesus..\nKaulah Yesus Kaulah Tuhan\nHanya Kau kupercaya\nKaulah harapan hidupku selamanya", chunks[1])

	onlyRepeats := BuildSongChunks("Yesus", "", "Yesus.. Yesus.. Yesus\n\nYesus.. Yesus.. Yesus")
	assert.Len(t, onlyRepeats, 1, "a stanza with fewer than minDistinctWords distinct words is not embedded on its own")
}

func TestBuildSongChunks_CapsChunkCount(t *testing.T) {
	lyrics := ""
	for i := 0; i < 30; i++ {
		lyrics += "baris lirik yang berbeda nomor " + string(rune('a'+i%26)) + string(rune('a'+i/26)) + "\n\n"
	}
	assert.Len(t, BuildSongChunks("Judul", "", lyrics), maxChunksPerSong)
}

// ── rankCandidates ────────────────────────────────────────────────────────────

func TestRankCandidates_FusesTextAndSemanticRanks(t *testing.T) {
	cands := []repositories.SearchCandidate{
		{ID: 1, Title: "Alpha", LexScore: 2, LexMatch: true},
		{ID: 2, Title: "Bravo", LexScore: 9, LexMatch: true}, // best text match, but not semantic
		{ID: 3, Title: "Charlie", LexScore: 0, LexMatch: false},
		{ID: 4, Title: "Delta", LexScore: 5, LexMatch: true},
	}
	// 4 is text #2 and semantic #1 → top; 2 is text #1 only; 1 text #3 + semantic #2; 3 semantic-only #3.
	// 99 was a semantic hit removed by filters, so it must not shift the semantic ranks.
	got := rankCandidates(cands, []int{4, 99, 1, 3})
	assert.Equal(t, []int{4, 1, 2, 3}, got)
}

func TestRankCandidates_StrongTextMatchesComeFirst(t *testing.T) {
	cands := []repositories.SearchCandidate{
		{ID: 1, Title: "Bapa Engkau Baik", LexScore: 3, LexMatch: true},
		{ID: 2, Title: "Bapa Engkau Sungguh Baik", LexScore: 2, LexMatch: true, Strength: 2},
		{ID: 3, Title: "Another Song", LexScore: 1, LexMatch: true, Strength: 1},
	}
	// Song 1 wins both rankings on score, but song 2 is the exact title the user typed and song 3 contains
	// the typed line verbatim; an exact title outranks a verbatim line.
	assert.Equal(t, []int{2, 3, 1}, rankCandidates(cands, []int{1, 3, 2}))
}

func TestRankCandidates_TextOnlyKeepsTextOrder(t *testing.T) {
	cands := []repositories.SearchCandidate{
		{ID: 1, Title: "B", LexScore: 1, LexMatch: true},
		{ID: 2, Title: "A", LexScore: 3, LexMatch: true},
		{ID: 3, Title: "C", LexScore: 1, LexMatch: true},
	}
	assert.Equal(t, []int{2, 1, 3}, rankCandidates(cands, nil))
}

// ── SongService.List with sortBy=relevance ────────────────────────────────────

func rankedSearchService(t *testing.T, embedErr error) (*SongService, *[]int) {
	t.Helper()
	var semanticPassed []int
	songs := &mocks.SongRepo{
		SearchCandidatesFn: func(_ string, extraIDs []int, _ string, _ []int, _, _ []string, _, _ *bool) ([]repositories.SearchCandidate, error) {
			semanticPassed = extraIDs
			return []repositories.SearchCandidate{
				{ID: 10, Title: "Text Match", LexScore: 3, LexMatch: true},
				{ID: 20, Title: "Meaning Match", LexMatch: false},
			}, nil
		},
		ListByIDsFn: func(ids []int) ([]models.Song, error) {
			out := []models.Song{}
			for i := len(ids) - 1; i >= 0; i-- { // deliberately unordered
				out = append(out, models.Song{ID: ids[i], Title: "song"})
			}
			return out, nil
		},
	}
	embedder := &mocks.Embedder{ModelName: "bge-m3", EmbedFn: func(context.Context, []string) ([][]float32, error) {
		if embedErr != nil {
			return nil, embedErr
		}
		return [][]float32{{0.1, 0.2}}, nil
	}}
	repo := &mocks.EmbeddingRepo{NearestSongsFn: func([]float32, string, float64, int) ([]repositories.SongDistance, error) {
		return []repositories.SongDistance{{SongID: 20, Distance: 0.2}}, nil
	}}
	svc := &SongService{songs: songs, tags: &mocks.TagRepo{}, embeddings: NewSongEmbeddingService(repo, embedder)}
	return svc, &semanticPassed
}

func TestSongService_List_RelevanceMergesSemanticMatches(t *testing.T) {
	svc, semanticPassed := rankedSearchService(t, nil)
	data, pagination, err := svc.List(1, 20, "kasih setia", "", "relevance", "DESC", nil, nil, nil, nil, nil, false)

	require.NoError(t, err)
	assert.Equal(t, []int{20}, *semanticPassed)
	require.Len(t, data, 2)
	assert.Equal(t, 2, pagination["totalItems"])
	assert.ElementsMatch(t, []int{10, 20}, []int{data[0]["id"].(int), data[1]["id"].(int)})
}

func TestSongService_List_RelevanceFallsBackToTextWhenEmbedderFails(t *testing.T) {
	svc, semanticPassed := rankedSearchService(t, errors.New("embedding service down"))
	data, _, err := svc.List(1, 20, "kasih setia", "", "relevance", "DESC", nil, nil, nil, nil, nil, false)

	require.NoError(t, err)
	assert.Empty(t, *semanticPassed)
	assert.Equal(t, 10, data[0]["id"], "text match first, order taken from the ranking not from ListByIDs")
}

func TestSongService_List_RelevancePaginates(t *testing.T) {
	svc, _ := rankedSearchService(t, nil)
	data, pagination, err := svc.List(2, 1, "kasih", "", "relevance", "DESC", nil, nil, nil, nil, nil, false)

	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, 2, pagination["totalItems"])
	assert.Equal(t, false, pagination["hasNextPage"])
}

// ── IndexSong ─────────────────────────────────────────────────────────────────

func TestSongEmbeddingService_IndexSong_SkipsUnchangedAndEmbedsChanged(t *testing.T) {
	src := &repositories.EmbeddingSource{
		ID:          7,
		Title:       "Hati Hamba",
		Artist:      sql.NullString{String: `["Sari Simorangkir"]`, Valid: true},
		PlainLyrics: sql.NullString{String: "Ku tak dapat lupakan kebaikan yang kutrima", Valid: true},
	}
	artist, lyrics := embeddingText(src)
	currentHash := songContentHash("bge-m3", src.Title, artist, lyrics)

	var stored map[int]string
	replaced := 0
	repo := &mocks.EmbeddingRepo{
		SourceFn:       func(int) (*repositories.EmbeddingSource, error) { return src, nil },
		StoredHashesFn: func(string) (map[int]string, error) { return stored, nil },
		ReplaceSongChunksFn: func(songID int, model, hash string, chunks []repositories.SongChunk) error {
			replaced++
			assert.Equal(t, 7, songID)
			assert.Equal(t, currentHash, hash)
			assert.Len(t, chunks, 2)
			return nil
		},
	}
	embedder := &mocks.Embedder{ModelName: "bge-m3", EmbedFn: func(_ context.Context, in []string) ([][]float32, error) {
		return make([][]float32, len(in)), nil
	}}
	svc := NewSongEmbeddingService(repo, embedder)

	stored = map[int]string{7: currentHash}
	embedded, err := svc.IndexSong(context.Background(), 7)
	require.NoError(t, err)
	assert.False(t, embedded)
	assert.Equal(t, 0, replaced)

	stored = map[int]string{7: "old-hash"}
	embedded, err = svc.IndexSong(context.Background(), 7)
	require.NoError(t, err)
	assert.True(t, embedded)
	assert.Equal(t, 1, replaced)
}

// ── SemanticMatches query cache ───────────────────────────────────────────────

func TestSongEmbeddingService_SemanticMatches_ReusesQueryVector(t *testing.T) {
	var embedded []string
	fail := true
	embedder := &mocks.Embedder{ModelName: "bge-m3", EmbedFn: func(_ context.Context, in []string) ([][]float32, error) {
		embedded = append(embedded, in...)
		if fail {
			return nil, errors.New("timeout")
		}
		return [][]float32{{0.1, 0.2}}, nil
	}}
	nearest := 0
	repo := &mocks.EmbeddingRepo{NearestSongsFn: func(vec []float32, _ string, _ float64, _ int) ([]repositories.SongDistance, error) {
		nearest++
		assert.Equal(t, []float32{0.1, 0.2}, vec)
		return nil, nil
	}}
	svc := NewSongEmbeddingService(repo, embedder)

	_, err := svc.SemanticMatches(context.Background(), "kasih setia")
	require.Error(t, err)
	fail = false
	_, err = svc.SemanticMatches(context.Background(), "kasih setia")
	require.NoError(t, err)
	_, err = svc.SemanticMatches(context.Background(), "  kasih   setia ")
	require.NoError(t, err)

	assert.Equal(t, []string{"kasih setia", "kasih setia"}, embedded, "a failed embedding is not cached; spacing variants share one vector")
	assert.Equal(t, 2, nearest, "the song lookup itself is never cached")
}

func TestVectorCache_EvictsLeastRecentlyUsed(t *testing.T) {
	c := newVectorCache(2)
	c.put("a", []float32{1})
	c.put("b", []float32{2})
	_, _ = c.get("a") // a is now more recent than b
	c.put("c", []float32{3})

	_, hasA := c.get("a")
	_, hasB := c.get("b")
	_, hasC := c.get("c")
	assert.True(t, hasA)
	assert.False(t, hasB)
	assert.True(t, hasC)
}

func TestNewSongEmbeddingService_NilWithoutEmbedder(t *testing.T) {
	assert.Nil(t, NewSongEmbeddingService(&mocks.EmbeddingRepo{}, nil))
}
