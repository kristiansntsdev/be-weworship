package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mixFixture is a user with two clear tastes (songs 1–4 along x, 5–8 along y) and a catalog of 40 playable songs.
type mixFixture struct {
	analytics  *mocks.AnalyticsRepo
	embeddings *mocks.EmbeddingRepo
	store      *mocks.MixRepo
	embedder   *mocks.Embedder
	replaced   [][]repositories.MixRow
	upserted   []repositories.MixRow
	nearest    int
}

func newMixFixture() *mixFixture {
	f := &mixFixture{}
	f.analytics = &mocks.AnalyticsRepo{
		TasteSongSignalsFn: func(int, int) ([]repositories.TasteSongRow, error) {
			rows := []repositories.TasteSongRow{}
			for id := 1; id <= 8; id++ {
				rows = append(rows, repositories.TasteSongRow{SongID: id, Kind: "add_to_playlist", Count: 1, AgeSeconds: day})
			}
			return rows, nil
		},
		TasteSearchSignalsFn: func(int, int, int) ([]repositories.TasteSearchRow, error) {
			return []repositories.TasteSearchRow{{Query: "natal", Count: 1}}, nil
		},
	}
	f.embeddings = &mocks.EmbeddingRepo{
		SongVectorsFn: func(ids []int, _ string) (map[int][]float32, error) {
			out := map[int][]float32{}
			for _, id := range ids {
				if id <= 4 {
					out[id] = []float32{1, float32(id) * 0.01, 0}
				} else {
					out[id] = []float32{float32(id) * 0.01, 1, 0}
				}
			}
			return out, nil
		},
		NearestPlayableSongsFn: func([]float32, string, int) ([]repositories.MixCandidate, error) {
			f.nearest++
			cands := []repositories.MixCandidate{}
			for id := 1; id <= 40; id++ {
				cands = append(cands, candidate(id, fmt.Sprintf("Artist %d", id)))
			}
			return cands, nil
		},
	}
	f.store = &mocks.MixRepo{
		ReplacePersonalFn: func(_ int, rows []repositories.MixRow) error {
			f.replaced = append(f.replaced, rows)
			return nil
		},
		UpsertFn: func(_ int, row repositories.MixRow) error {
			f.upserted = append(f.upserted, row)
			return nil
		},
	}
	f.embedder = &mocks.Embedder{ModelName: "bge-m3", EmbedFn: func(_ context.Context, in []string) ([][]float32, error) {
		out := make([][]float32, len(in))
		for i := range in {
			out[i] = []float32{0, 0, 1}
		}
		return out, nil
	}}
	return f
}

func (f *mixFixture) service() *MixService {
	svc := NewMixService(f.analytics, f.embeddings, f.store, NewSongEmbeddingService(&mocks.EmbeddingRepo{}, f.embedder))
	svc.now = func() time.Time { return time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC) }
	return svc
}

func TestMixService_Mixes_ReturnsFreshStoredMixesWithoutRebuilding(t *testing.T) {
	f := newMixFixture()
	f.store.ListForUserFn = func(int) ([]repositories.MixRow, error) {
		return []repositories.MixRow{
			{Key: "mix-1", Title: "Mix 1", Subtitle: "Bethel Music", SongsJSON: `[{"id":4,"title":"Song 4","artist":"Bethel Music","key":"G"}]`, AgeSeconds: 3600},
			{Key: "mood-calm", Title: "Calm", SongsJSON: `[]`, AgeSeconds: 99 * 3600}, // stale moods don't force a rebuild
		}, nil
	}
	res, err := f.service().Mixes(context.Background(), 1)

	require.NoError(t, err)
	assert.True(t, res.Personalized)
	require.Len(t, res.Mixes, 1)
	assert.Equal(t, "mix-1", res.Mixes[0].Key)
	assert.Equal(t, 4, res.Mixes[0].Songs[0].ID)
	assert.Len(t, res.Moods, len(mixMoods))
	assert.Zero(t, f.nearest, "no vector search for a fresh set")
	assert.Empty(t, f.replaced)
}

func TestMixService_Mixes_RebuildsStaleMixes(t *testing.T) {
	f := newMixFixture()
	f.store.ListForUserFn = func(int) ([]repositories.MixRow, error) {
		return []repositories.MixRow{{Key: "mix-1", Title: "Mix 1", SongsJSON: `[]`, AgeSeconds: 25 * 3600}}, nil
	}
	res, err := f.service().Mixes(context.Background(), 1)

	require.NoError(t, err)
	assert.True(t, res.Personalized)
	keys := []string{}
	for _, m := range res.Mixes {
		keys = append(keys, m.Key)
		if m.Key == "supermix" {
			// The mock offers the same 40 songs to every cluster; the Supermix gets what the numbered mixes left.
			assert.GreaterOrEqual(t, len(m.Songs), mixMinSongs)
		} else {
			assert.Len(t, m.Songs, mixSize)
		}
	}
	// 9 signals → up to 2 taste clusters. The search sits on its own axis and may form a cluster, so only
	// require Mix 1, Mix 2 and a Supermix.
	assert.Contains(t, keys, "mix-1")
	assert.Contains(t, keys, "mix-2")
	assert.Equal(t, "supermix", keys[len(keys)-1])
	require.Len(t, f.replaced, 1)
	assert.Len(t, f.replaced[0], len(res.Mixes), "the new set is stored")
}

func TestMixService_Mixes_SupermixAvoidsSongsAlreadyInMixes(t *testing.T) {
	f := newMixFixture()
	res, err := f.service().Mixes(context.Background(), 1)
	require.NoError(t, err)

	inMixes := map[int]bool{}
	var supermix *Mix
	for i, m := range res.Mixes {
		if m.Key == "supermix" {
			supermix = &res.Mixes[i]
			continue
		}
		for _, s := range m.Songs {
			inMixes[s.ID] = true
		}
	}
	require.NotNil(t, supermix)
	for _, s := range supermix.Songs {
		assert.False(t, inMixes[s.ID], "song %d is already in a numbered mix", s.ID)
	}
}

func TestMixService_Mixes_NotEnoughHistory(t *testing.T) {
	f := newMixFixture()
	f.analytics.TasteSongSignalsFn = func(int, int) ([]repositories.TasteSongRow, error) {
		return []repositories.TasteSongRow{{SongID: 1, Kind: "view", Count: 1}}, nil
	}
	f.analytics.TasteSearchSignalsFn = func(int, int, int) ([]repositories.TasteSearchRow, error) { return nil, nil }
	res, err := f.service().Mixes(context.Background(), 1)

	require.NoError(t, err)
	assert.False(t, res.Personalized)
	assert.Empty(t, res.Mixes)
	assert.NotEmpty(t, res.Moods, "mood chips work without history")
	require.Len(t, f.replaced, 1)
	assert.Empty(t, f.replaced[0], "old personal mixes are cleared")
}

func TestMixService_Mixes_EmbedErrorFallsBackToSongs(t *testing.T) {
	f := newMixFixture()
	f.embedder.EmbedFn = func(context.Context, []string) ([][]float32, error) { return nil, errors.New("ollama down") }
	res, err := f.service().Mixes(context.Background(), 1)

	require.NoError(t, err, "embedding failures never fail the request")
	assert.True(t, res.Personalized)
	assert.NotEmpty(t, res.Mixes)
}

func TestMixService_Mixes_DisabledWithoutEmbedder(t *testing.T) {
	f := newMixFixture()
	svc := NewMixService(f.analytics, f.embeddings, f.store, nil)
	res, err := svc.Mixes(context.Background(), 1)

	require.NoError(t, err)
	assert.False(t, res.Personalized)
	assert.Empty(t, res.Mixes)
	assert.Empty(t, res.Moods)
}

func TestMixService_MoodMix_BuildsAndStores(t *testing.T) {
	f := newMixFixture()
	mix, err := f.service().MoodMix(context.Background(), 1, "christmas")

	require.NoError(t, err)
	assert.Equal(t, "mood-christmas", mix.Key)
	assert.Equal(t, "Christmas", mix.Title)
	assert.Len(t, mix.Songs, mixSize)
	require.Len(t, f.upserted, 1)
	assert.Equal(t, "mood-christmas", f.upserted[0].Key)
}

func TestMixService_MoodMix_UsesFreshStoredMix(t *testing.T) {
	f := newMixFixture()
	f.store.ListForUserFn = func(int) ([]repositories.MixRow, error) {
		return []repositories.MixRow{{Key: "mood-calm", Title: "Calm", SongsJSON: `[{"id":2,"title":"Song 2","artist":null,"key":null}]`, AgeSeconds: 60}}, nil
	}
	mix, err := f.service().MoodMix(context.Background(), 1, "calm")

	require.NoError(t, err)
	require.Len(t, mix.Songs, 1)
	assert.Equal(t, 2, mix.Songs[0].ID)
	assert.Zero(t, f.nearest)
}

func TestMixService_MoodMix_Errors(t *testing.T) {
	f := newMixFixture()
	_, err := f.service().MoodMix(context.Background(), 1, "party")
	assert.ErrorIs(t, err, ErrMoodNotFound)

	_, err = NewMixService(f.analytics, f.embeddings, f.store, nil).MoodMix(context.Background(), 1, "calm")
	assert.ErrorIs(t, err, ErrMixesUnavailable)

	f.embedder.EmbedFn = func(context.Context, []string) ([][]float32, error) { return nil, errors.New("ollama down") }
	_, err = f.service().MoodMix(context.Background(), 1, "calm")
	assert.ErrorIs(t, err, ErrMixesUnavailable, "a mood mix can't be built without the mood vector")
}
