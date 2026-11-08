package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"be-songbanks-v1/api/repositories"
)

const (
	mixFreshFor     = 24 * time.Hour
	mixEmbedTimeout = 2500 * time.Millisecond
	moodKeyPrefix   = "mood-"
)

var (
	ErrMoodNotFound     = errors.New("mood not found")
	ErrMixesUnavailable = errors.New("mixes are not available right now")
)

// Mix is one mix as the API returns it.
type Mix struct {
	Key      string    `json:"key"`
	Title    string    `json:"title"`
	Subtitle string    `json:"subtitle"`
	Songs    []MixSong `json:"songs"`
}

// MixesResult is the "Mixed for you" shelf: the user's personal mixes plus the mood chips.
type MixesResult struct {
	Personalized bool      `json:"personalized"` // false: not enough history yet for personal mixes
	Mixes        []Mix     `json:"mixes"`
	Moods        []MixMood `json:"moods"`
}

// MixService builds "Mixed for you" and mood mixes from embeddings (see mix_builder.go) and keeps each one for a day,
// so a mix stays the same between the home card, the mix page and play mode.
type MixService struct {
	analytics  repositories.AnalyticsRepoIface
	embeddings repositories.EmbeddingRepoIface
	store      repositories.MixRepoIface
	vectors    *SongEmbeddingService // nil = no embedder configured, so mixes are disabled
	now        func() time.Time
}

// vectors may be nil (no embedder configured): Mixes is then empty and MoodMix unavailable.
func NewMixService(analytics repositories.AnalyticsRepoIface, embeddings repositories.EmbeddingRepoIface, store repositories.MixRepoIface, vectors *SongEmbeddingService) *MixService {
	return &MixService{analytics: analytics, embeddings: embeddings, store: store, vectors: vectors, now: time.Now}
}

// Mixes returns the user's personal mixes, rebuilding them when the stored set is missing or older than a day.
func (s *MixService) Mixes(ctx context.Context, userID int) (MixesResult, error) {
	if s.vectors == nil {
		return MixesResult{Mixes: []Mix{}, Moods: []MixMood{}}, nil
	}
	rows, err := s.store.ListForUser(userID)
	if err != nil {
		return MixesResult{}, err
	}
	if cached, ok := freshPersonalMixes(rows); ok {
		return MixesResult{Personalized: true, Mixes: cached, Moods: mixMoods}, nil
	}

	mixes, err := s.buildPersonal(ctx, userID)
	if err != nil {
		return MixesResult{}, err
	}
	stored := make([]repositories.MixRow, len(mixes))
	for i, m := range mixes {
		stored[i] = toMixRow(m)
	}
	if err := s.store.ReplacePersonal(userID, stored); err != nil {
		// The mixes are still right for this response; the next request rebuilds them.
		log.Printf("[mixes] storing mixes for user %d failed: %v", userID, err)
	}
	return MixesResult{Personalized: len(mixes) > 0, Mixes: mixes, Moods: mixMoods}, nil
}

// MoodMix returns the mix for one mood chip: songs near the mood prompt, leaning toward the user's overall taste
// when there is enough history. Errors are ErrMoodNotFound, ErrMixesUnavailable, or a storage error.
func (s *MixService) MoodMix(ctx context.Context, userID int, moodID string) (*Mix, error) {
	mood, ok := findMood(moodID)
	if !ok {
		return nil, ErrMoodNotFound
	}
	if s.vectors == nil {
		return nil, ErrMixesUnavailable
	}
	key := moodKeyPrefix + mood.ID
	rows, err := s.store.ListForUser(userID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Key != key {
			continue
		}
		if m, ok := decodeFreshMix(r); ok {
			return &m, nil
		}
	}

	ectx, cancel := context.WithTimeout(ctx, mixEmbedTimeout)
	moodVecs, err := s.vectors.QueryVectors(ectx, []string{mood.Prompt})
	cancel()
	if err != nil {
		log.Printf("[mixes] embedding mood %q failed: %v", mood.ID, err)
		return nil, ErrMixesUnavailable
	}
	signals, err := s.tasteSignals(ctx, userID)
	if err != nil {
		return nil, err
	}
	var taste []float32
	if len(signals) >= mixMinSignals {
		taste = weightedCentroid(signals, nil)
	}
	candidates, err := s.embeddings.NearestPlayableSongs(blendVectors(moodVecs[0], taste, mixMoodWeight), s.vectors.Model(), mixCandidateLimit)
	if err != nil {
		return nil, err
	}
	songs, subtitle := fillMix(candidates, nil, false)
	mix := Mix{Key: key, Title: mood.Label, Subtitle: subtitle, Songs: songs}
	if len(songs) > 0 {
		if err := s.store.Upsert(userID, toMixRow(mix)); err != nil {
			log.Printf("[mixes] storing mood mix %q for user %d failed: %v", key, userID, err)
		}
	}
	return &mix, nil
}

// buildPersonal clusters the user's taste signals and fills one mix per cluster, plus a Supermix when there are two
// or more. Returns no mixes when the history has fewer than mixMinSignals signals.
func (s *MixService) buildPersonal(ctx context.Context, userID int) ([]Mix, error) {
	mixes := []Mix{}
	signals, err := s.tasteSignals(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(signals) < mixMinSignals {
		return mixes, nil
	}

	known := knownSongs(signals)
	model := s.vectors.Model()
	lists := [][]repositories.MixCandidate{}
	used := map[int]bool{}
	for _, c := range clusterSignals(signals, clusterCount(len(signals)), mixSeed(userID, s.now())) {
		candidates, err := s.embeddings.NearestPlayableSongs(c.centroid, model, mixCandidateLimit)
		if err != nil {
			return nil, err
		}
		songs, subtitle := fillMix(candidates, known, true)
		if len(songs) < mixMinSongs {
			continue
		}
		n := len(mixes) + 1
		mixes = append(mixes, Mix{Key: fmt.Sprintf("mix-%d", n), Title: fmt.Sprintf("Mix %d", n), Subtitle: subtitle, Songs: songs})
		lists = append(lists, candidates)
		for _, song := range songs {
			used[song.ID] = true
		}
	}

	if len(lists) >= 2 {
		// The Supermix takes from every taste in turn. Prefer songs not already in Mix 1..N; allow repeats if that
		// leaves too few.
		songs, subtitle := fillMix(roundRobin(lists, used), nil, false)
		if len(songs) < mixMinSongs {
			songs, subtitle = fillMix(roundRobin(lists, nil), nil, false)
		}
		if len(songs) >= mixMinSongs {
			mixes = append(mixes, Mix{Key: "supermix", Title: "Supermix", Subtitle: subtitle, Songs: songs})
		}
	}
	return mixes, nil
}

// tasteSignals loads the user's history and turns it into weighted vectors. Song vectors come from the database;
// searches are embedded through the query cache. If embedding the searches fails, mixes are built from songs alone
// (embeddings degrade, they never fail a request).
func (s *MixService) tasteSignals(ctx context.Context, userID int) ([]tasteSignal, error) {
	songRows, err := s.analytics.TasteSongSignals(userID, mixSignalDays)
	if err != nil {
		return nil, err
	}
	searches, err := s.analytics.TasteSearchSignals(userID, mixSignalDays, mixSearchLimit)
	if err != nil {
		return nil, err
	}

	ids := []int{}
	seen := map[int]bool{}
	for _, r := range songRows {
		if !seen[r.SongID] {
			seen[r.SongID] = true
			ids = append(ids, r.SongID)
		}
	}
	songVecs, err := s.embeddings.SongVectors(ids, s.vectors.Model())
	if err != nil {
		return nil, err
	}

	var searchVecs [][]float32
	if len(searches) > 0 {
		texts := make([]string, len(searches))
		for i, r := range searches {
			texts[i] = r.Query
		}
		ectx, cancel := context.WithTimeout(ctx, mixEmbedTimeout)
		searchVecs, err = s.vectors.QueryVectors(ectx, texts)
		cancel()
		if err != nil {
			log.Printf("[mixes] embedding searches for user %d failed, using songs only: %v", userID, err)
			searches, searchVecs = nil, nil
		}
	}
	return buildTasteSignals(songRows, songVecs, searches, searchVecs), nil
}

// freshPersonalMixes returns the stored personal mixes (every key except mood-*) when there is at least one and all
// of them are younger than mixFreshFor.
func freshPersonalMixes(rows []repositories.MixRow) ([]Mix, bool) {
	out := []Mix{}
	for _, r := range rows {
		if strings.HasPrefix(r.Key, moodKeyPrefix) {
			continue
		}
		m, ok := decodeFreshMix(r)
		if !ok {
			return nil, false
		}
		out = append(out, m)
	}
	return out, len(out) > 0
}

func decodeFreshMix(r repositories.MixRow) (Mix, bool) {
	if time.Duration(r.AgeSeconds*float64(time.Second)) >= mixFreshFor {
		return Mix{}, false
	}
	songs := []MixSong{}
	if err := json.Unmarshal([]byte(r.SongsJSON), &songs); err != nil {
		return Mix{}, false
	}
	return Mix{Key: r.Key, Title: r.Title, Subtitle: r.Subtitle, Songs: songs}, true
}

func toMixRow(m Mix) repositories.MixRow {
	buf, _ := json.Marshal(m.Songs)
	return repositories.MixRow{Key: m.Key, Title: m.Title, Subtitle: m.Subtitle, SongsJSON: string(buf)}
}
