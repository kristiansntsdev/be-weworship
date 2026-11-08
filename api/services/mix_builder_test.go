package services

import (
	"database/sql"
	"fmt"
	"testing"

	"be-songbanks-v1/api/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = 86400.0

func candidate(id int, artist string) repositories.MixCandidate {
	return repositories.MixCandidate{
		SongID: id,
		Title:  fmt.Sprintf("Song %d", id),
		Artist: sql.NullString{String: fmt.Sprintf("[%q]", artist), Valid: artist != ""},
		Key:    sql.NullString{String: "G", Valid: true},
	}
}

func songIDs(songs []MixSong) []int {
	ids := make([]int, len(songs))
	for i, s := range songs {
		ids[i] = s.ID
	}
	return ids
}

// ── signalWeight ────────────────────────────────────────────────────────────────

func TestSignalWeight_DecaysWithHalfLife(t *testing.T) {
	assert.InDelta(t, 1.0, signalWeight("view", 1, 0), 1e-9)
	assert.InDelta(t, 0.5, signalWeight("view", 1, mixHalfLifeDays*day), 1e-9)
	assert.InDelta(t, 1.5, signalWeight("add_to_playlist", 1, mixHalfLifeDays*day), 1e-9)
}

func TestSignalWeight_FavoritesDoNotDecayOrRepeat(t *testing.T) {
	assert.Equal(t, 3.0, signalWeight("favorite", 1, 365*day))
	assert.Equal(t, 3.0, signalWeight("favorite", 5, 0))
}

func TestSignalWeight_RepeatsAreCapped(t *testing.T) {
	assert.InDelta(t, 3.0, signalWeight("view", 50, 0), 1e-9)
	assert.InDelta(t, 4.0, signalWeight("search", 2, 0), 1e-9)
}

func TestSignalWeight_UnknownKindIsZero(t *testing.T) {
	assert.Equal(t, 0.0, signalWeight("share", 1, 0))
}

// ── buildTasteSignals ───────────────────────────────────────────────────────────

func TestBuildTasteSignals_MergesKindsPerSongAndSkipsMissingVectors(t *testing.T) {
	rows := []repositories.TasteSongRow{
		{SongID: 7, Kind: "view", Count: 1},
		{SongID: 7, Kind: "favorite", Count: 1},
		{SongID: 3, Kind: "add_to_playlist", Count: 1},
		{SongID: 9, Kind: "view", Count: 1}, // no embedding
	}
	vecs := map[int][]float32{7: {1, 0}, 3: {0, 2}}
	searches := []repositories.TasteSearchRow{{Query: "natal", Count: 1}}
	signals := buildTasteSignals(rows, vecs, searches, [][]float32{{0, 1}})

	require.Len(t, signals, 3)
	assert.Equal(t, 3, signals[0].songID, "songs come first, by id")
	assert.Equal(t, 7, signals[1].songID)
	assert.InDelta(t, 4.0, signals[1].weight, 1e-9, "view 1 + favorite 3")
	assert.Equal(t, 0, signals[2].songID, "searches last")
	assert.InDelta(t, 1.0, dot(signals[0].vec, signals[0].vec), 1e-6, "vectors are unit length")
}

// ── clusterSignals ──────────────────────────────────────────────────────────────

func TestClusterSignals_SeparatesTwoTastes(t *testing.T) {
	signals := []tasteSignal{
		{songID: 1, vec: normalize([]float32{1, 0.05, 0}), weight: 1},
		{songID: 2, vec: normalize([]float32{1, -0.05, 0}), weight: 1},
		{songID: 3, vec: normalize([]float32{0.95, 0, 0.05}), weight: 1},
		{songID: 4, vec: normalize([]float32{0, 1, 0.05}), weight: 1},
		{songID: 5, vec: normalize([]float32{0.05, 1, 0}), weight: 1},
		{songID: 6, vec: normalize([]float32{0, 0.95, -0.05}), weight: 1},
	}
	clusters := clusterSignals(signals, 2, 42)

	require.Len(t, clusters, 2)
	groups := [][]int{}
	for _, c := range clusters {
		ids := []int{}
		for _, m := range c.members {
			ids = append(ids, signals[m].songID)
		}
		groups = append(groups, ids)
	}
	assert.ElementsMatch(t, [][]int{{1, 2, 3}, {4, 5, 6}}, groups)
}

func TestClusterSignals_SameSeedSameClusters(t *testing.T) {
	signals := []tasteSignal{
		{songID: 1, vec: normalize([]float32{1, 0.2}), weight: 1},
		{songID: 2, vec: normalize([]float32{0.2, 1}), weight: 2},
		{songID: 3, vec: normalize([]float32{0.7, 0.7}), weight: 1},
		{songID: 4, vec: normalize([]float32{1, -0.3}), weight: 3},
	}
	a := clusterSignals(signals, 2, 7)
	b := clusterSignals(signals, 2, 7)
	assert.Equal(t, a, b)
}

func TestClusterSignals_FoldsTinyClusterAndSortsByWeight(t *testing.T) {
	signals := []tasteSignal{
		{songID: 1, vec: normalize([]float32{1, 0}), weight: 5},
		{songID: 2, vec: normalize([]float32{1, 0.05}), weight: 5},
		{songID: 3, vec: normalize([]float32{0, 1}), weight: 0.2}, // 2% of the weight
	}
	clusters := clusterSignals(signals, 2, 1)

	require.Len(t, clusters, 1, "a taste under mixMinClusterShare joins its nearest neighbour")
	assert.Len(t, clusters[0].members, 3)
}

func TestClusterCount(t *testing.T) {
	assert.Equal(t, 1, clusterCount(3))
	assert.Equal(t, 2, clusterCount(6))
	assert.Equal(t, 3, clusterCount(15))
}

// ── fillMix ─────────────────────────────────────────────────────────────────────

func TestFillMix_CapsSongsPerArtist(t *testing.T) {
	cands := []repositories.MixCandidate{
		candidate(1, "Bethel Music"), candidate(2, "Bethel Music"), candidate(3, "bethel music"),
		candidate(4, "JPCC Worship"), candidate(5, "Bethel Music"),
	}
	songs, _ := fillMix(cands, nil, false)
	assert.Equal(t, []int{1, 2, 4}, songIDs(songs), "artist cap ignores case")
}

func TestFillMix_PutsKnownSongsOnEveryThirdSlot(t *testing.T) {
	cands := []repositories.MixCandidate{}
	for i := 1; i <= 12; i++ {
		cands = append(cands, candidate(i, fmt.Sprintf("Artist %d", i)))
	}
	known := map[int]bool{5: true, 9: true, 11: true}
	songs, _ := fillMix(cands, known, true)

	ids := songIDs(songs)
	require.Len(t, ids, 12)
	assert.Equal(t, 5, ids[0], "a familiar song opens the mix")
	assert.Equal(t, 9, ids[3])
	assert.Equal(t, 11, ids[6])
	assert.Equal(t, []int{1, 2}, ids[1:3], "fresh songs keep distance order")
}

func TestFillMix_FallsBackWhenOneGroupRunsOut(t *testing.T) {
	cands := []repositories.MixCandidate{candidate(1, "A"), candidate(2, "B"), candidate(3, "C")}
	songs, _ := fillMix(cands, map[int]bool{}, true)
	assert.Equal(t, []int{1, 2, 3}, songIDs(songs), "no known songs: every slot takes the next fresh song")
}

func TestFillMix_StopsAtMixSize(t *testing.T) {
	cands := []repositories.MixCandidate{}
	for i := 1; i <= mixSize+10; i++ {
		cands = append(cands, candidate(i, fmt.Sprintf("Artist %d", i)))
	}
	songs, _ := fillMix(cands, nil, false)
	assert.Len(t, songs, mixSize)
}

func TestFillMix_SubtitleListsTopArtists(t *testing.T) {
	cands := []repositories.MixCandidate{
		candidate(1, "JPCC Worship"), candidate(2, "Bethel Music"), candidate(3, "Bethel Music"),
		candidate(4, "Elevation Worship"), candidate(5, "Hillsong"), candidate(6, ""),
	}
	songs, subtitle := fillMix(cands, nil, false)
	assert.Equal(t, "Bethel Music, JPCC Worship, Elevation Worship", subtitle)
	assert.Nil(t, songs[5].Artist, "a song without artist has a null artist")
	require.NotNil(t, songs[0].Key)
	assert.Equal(t, "G", *songs[0].Key)
}

// ── roundRobin / blendVectors ───────────────────────────────────────────────────

func TestRoundRobin_TakesFromEachListInTurn(t *testing.T) {
	a := []repositories.MixCandidate{candidate(1, ""), candidate(2, ""), candidate(3, "")}
	b := []repositories.MixCandidate{candidate(10, ""), candidate(2, "")}
	got := roundRobin([][]repositories.MixCandidate{a, b}, map[int]bool{3: true})

	ids := []int{}
	for _, c := range got {
		ids = append(ids, c.SongID)
	}
	assert.Equal(t, []int{1, 10, 2}, ids, "skips listed songs and repeats")
}

func TestBlendVectors(t *testing.T) {
	assert.Equal(t, normalize([]float32{3, 4}), blendVectors([]float32{3, 4}, nil, 0.6), "no taste: just the mood")
	v := blendVectors([]float32{1, 0}, []float32{0, 1}, 0.6)
	assert.InDelta(t, 1.0, dot(v, v), 1e-6)
	assert.Greater(t, v[0], v[1], "the heavier side wins")
}

func TestFindMood(t *testing.T) {
	m, ok := findMood("calm")
	assert.True(t, ok)
	assert.Equal(t, "Calm", m.Label)
	_, ok = findMood("party")
	assert.False(t, ok)
}

func TestFillMix_DropsPlaceholderArtists(t *testing.T) {
	cands := []repositories.MixCandidate{candidate(1, "-"), candidate(2, "[none]"), candidate(3, "Unknown Artist"), candidate(4, "Regina")}
	songs, subtitle := fillMix(cands, nil, false)

	assert.Nil(t, songs[0].Artist, `"-" means no artist`)
	assert.Nil(t, songs[1].Artist)
	assert.Nil(t, songs[2].Artist)
	require.NotNil(t, songs[3].Artist)
	assert.Equal(t, "Regina", subtitle, "placeholders never reach the subtitle")
}
