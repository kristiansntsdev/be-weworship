package services

import (
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

// Personal mixes come from embeddings only (bge-m3 + pgvector); there is no generative model:
//  1. every song the user favorited, added to a playlist or opened, and every search that found something, becomes a
//     weighted signal (kind weight × repeat count × time decay) with a unit vector;
//  2. signals are grouped with weighted spherical k-means, and each group becomes one "Mix N";
//  3. each mix is filled with the playable songs nearest its centroid, every third one a song the user already knows.
// Mood mixes blend a fixed mood prompt's vector with the user's overall taste.

const (
	mixSignalDays       = 90
	mixHalfLifeDays     = 14.0
	mixSearchLimit      = 20
	mixMaxRepeat        = 3 // repeats past this add nothing; views are inflated because opening a playlist fetches every song
	mixMinSignals       = 3
	mixSize             = 25
	mixMinSongs         = 8 // a mix with fewer playable songs is dropped
	mixCandidateLimit   = 80
	mixFamiliarEvery    = 3 // slots 0, 3, 6, … go to songs the user already knows
	mixMaxPerArtist     = 2
	mixMinClusterShare  = 0.1
	mixKMeansIterations = 12
	mixKMeansRestarts   = 8
	mixMoodWeight       = 0.6
	moodMinSongs        = 5 // a mood chip is shown only when its tag is on at least this many playable songs
)

var mixKindWeight = map[string]float64{"favorite": 3, "add_to_playlist": 3, "search": 2, "view": 1}

// MixMood is one mood chip. Songs belong to a mood through the tag named Label (e.g. "Calm"), which admins can also
// set on a song. Prompt is embedded only to order a mood's songs; it mixes Indonesian and English because most
// lyrics are Indonesian and bge-m3 is multilingual.
type MixMood struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Prompt string `json:"-"`
}

var mixMoods = []MixMood{
	{ID: "calm", Label: "Calm", Prompt: "Lagu penyembahan yang tenang, teduh, lembut dan intim di hadirat Tuhan. Quiet, gentle, intimate worship."},
	{ID: "joyful", Label: "Joyful", Prompt: "Lagu pujian penuh sukacita, bersorak, menari dan bertepuk tangan bagi Tuhan. Joyful, upbeat praise and celebration."},
	{ID: "prayer", Label: "Prayer", Prompt: "Lagu doa dan penyerahan diri, berserah dan memohon pertolongan serta pengampunan Tuhan. Songs of prayer and surrender."},
	{ID: "hope", Label: "Hope", Prompt: "Lagu pengharapan dan kekuatan di tengah pergumulan, Tuhan setia dan tidak pernah meninggalkan. Songs of hope and God's faithfulness."},
	{ID: "christmas", Label: "Christmas", Prompt: "Lagu Natal tentang kelahiran Yesus di Betlehem, malam kudus, bayi di palungan. Christmas carols about the birth of Jesus."},
	{ID: "easter", Label: "Easter", Prompt: "Lagu Paskah tentang salib, kematian dan kebangkitan Yesus, kubur kosong, Ia hidup. Easter songs of the cross and the resurrection."},
}

func findMood(id string) (MixMood, bool) {
	for _, m := range mixMoods {
		if m.ID == id {
			return m, true
		}
	}
	return MixMood{}, false
}

// MixSong is one song in a mix, as stored in user_mixes and returned by the API.
type MixSong struct {
	ID     int     `json:"id"`
	Title  string  `json:"title"`
	Artist *string `json:"artist"`
	Key    *string `json:"key"`
}

// ── Signals ─────────────────────────────────────────────────────────────────────

// tasteSignal is one weighted piece of evidence about what a user likes.
type tasteSignal struct {
	songID int       // 0 for a search
	vec    []float32 // unit length
	weight float64
}

// signalWeight is kind weight × repeat count (capped) × time decay with a mixHalfLifeDays half-life.
// Favorites neither repeat nor decay: they are a standing, explicit choice.
func signalWeight(kind string, count int, ageSeconds float64) float64 {
	w := mixKindWeight[kind]
	if w == 0 || count < 1 {
		return 0
	}
	if kind == "favorite" {
		return w
	}
	count = min(count, mixMaxRepeat)
	ageDays := math.Max(ageSeconds, 0) / 86400
	return w * float64(count) * math.Pow(0.5, ageDays/mixHalfLifeDays)
}

// buildTasteSignals merges every kind of interaction with a song into one signal and adds one signal per search.
// Songs without an embedding and searches without a vector are skipped. Output order is fixed (songs by id, then
// searches in input order) so clustering gives the same mixes for the same history.
func buildTasteSignals(songRows []repositories.TasteSongRow, songVecs map[int][]float32, searches []repositories.TasteSearchRow, searchVecs [][]float32) []tasteSignal {
	weights := map[int]float64{}
	for _, r := range songRows {
		weights[r.SongID] += signalWeight(r.Kind, r.Count, r.AgeSeconds)
	}
	ids := make([]int, 0, len(weights))
	for id, w := range weights {
		if _, ok := songVecs[id]; ok && w > 0 {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)

	out := make([]tasteSignal, 0, len(ids)+len(searches))
	for _, id := range ids {
		if v := normalize(songVecs[id]); v != nil {
			out = append(out, tasteSignal{songID: id, vec: v, weight: weights[id]})
		}
	}
	for i, r := range searches {
		if i >= len(searchVecs) {
			break
		}
		w := signalWeight("search", r.Count, r.AgeSeconds)
		if v := normalize(searchVecs[i]); v != nil && w > 0 {
			out = append(out, tasteSignal{vec: v, weight: w})
		}
	}
	return out
}

func knownSongs(signals []tasteSignal) map[int]bool {
	known := map[int]bool{}
	for _, s := range signals {
		if s.songID > 0 {
			known[s.songID] = true
		}
	}
	return known
}

// mixSeed changes once a day per user, so rebuilding within the same day reproduces the same clusters.
func mixSeed(userID int, now time.Time) int64 {
	return int64(userID)*1_000_003 + now.Unix()/86400
}

// ── Vectors ─────────────────────────────────────────────────────────────────────

// normalize returns v scaled to unit length, or nil for an empty or zero vector.
func normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return nil
	}
	n := math.Sqrt(sum)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / n)
	}
	return out
}

func dot(a, b []float32) float64 {
	var s float64
	for i := range min(len(a), len(b)) {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}

// weightedCentroid is the unit-length, weight-averaged direction of the signals at idx (all signals when idx is nil).
func weightedCentroid(signals []tasteSignal, idx []int) []float32 {
	if len(signals) == 0 {
		return nil
	}
	acc := make([]float64, len(signals[0].vec))
	add := func(s tasteSignal) {
		for i := range min(len(acc), len(s.vec)) {
			acc[i] += s.weight * float64(s.vec[i])
		}
	}
	if idx == nil {
		for _, s := range signals {
			add(s)
		}
	} else {
		for _, i := range idx {
			add(signals[i])
		}
	}
	out := make([]float32, len(acc))
	for i, x := range acc {
		out[i] = float32(x)
	}
	return normalize(out)
}

// blendVectors returns the direction of w·a + (1−w)·b. With b nil it is a's direction (and vice versa).
func blendVectors(a, b []float32, w float64) []float32 {
	a, b = normalize(a), normalize(b)
	if b == nil {
		return a
	}
	if a == nil {
		return b
	}
	out := make([]float32, len(a))
	for i := range a {
		var bv float64
		if i < len(b) {
			bv = float64(b[i])
		}
		out[i] = float32(w*float64(a[i]) + (1-w)*bv)
	}
	return normalize(out)
}

// ── Clustering ──────────────────────────────────────────────────────────────────

type tasteCluster struct {
	members  []int     // indexes into the signals slice
	centroid []float32 // unit length
	weight   float64
}

// clusterCount picks how many taste clusters (and so personal mixes) a history of n signals supports.
func clusterCount(n int) int {
	switch {
	case n < 6:
		return 1
	case n < 15:
		return 2
	default:
		return 3
	}
}

// clusterSignals groups signals into at most k clusters with weighted spherical k-means (cosine similarity).
// Clusters holding less than mixMinClusterShare of the total weight are folded into their nearest neighbour.
// k-means runs mixKMeansRestarts times and keeps the tightest result: a single run can give a lone, light signal its
// own cluster and lump two real tastes together, and folding that cluster away would then leave one mix instead of two.
// All restarts draw from one generator seeded with seed, so the same history and seed give the same clusters.
// The result is ordered by weight, heaviest first.
func clusterSignals(signals []tasteSignal, k int, seed int64) []tasteCluster {
	if len(signals) == 0 {
		return nil
	}
	k = min(max(k, 1), len(signals))
	rng := rand.New(rand.NewSource(seed))
	var best []tasteCluster
	bestScore := math.Inf(-1)
	for range mixKMeansRestarts {
		clusters := foldSmallClusters(signals, kmeans(signals, k, rng))
		if score := clusterScore(signals, clusters); score > bestScore {
			best, bestScore = clusters, score
		}
		if k == 1 {
			break // one cluster is always all signals
		}
	}
	sort.SliceStable(best, func(i, j int) bool { return best[i].weight > best[j].weight })
	return best
}

func kmeans(signals []tasteSignal, k int, rng *rand.Rand) []tasteCluster {
	centers := seedCenters(signals, k, rng)
	assign := nearestCenters(signals, centers)
	for range mixKMeansIterations {
		centers = centersOf(signals, assign, len(centers))
		next := nearestCenters(signals, centers)
		if slices.Equal(next, assign) {
			break
		}
		assign = next
	}
	return groupClusters(signals, assign, len(centers))
}

// clusterScore is the weighted similarity of every signal to its cluster's centroid; higher means tighter clusters.
func clusterScore(signals []tasteSignal, clusters []tasteCluster) float64 {
	score := 0.0
	for _, c := range clusters {
		for _, i := range c.members {
			score += signals[i].weight * dot(signals[i].vec, c.centroid)
		}
	}
	return score
}

// foldSmallClusters repeatedly dissolves the lightest cluster into its neighbours while it holds less than
// mixMinClusterShare of the total weight.
func foldSmallClusters(signals []tasteSignal, clusters []tasteCluster) []tasteCluster {
	for len(clusters) > 1 {
		total, smallest := 0.0, 0
		for i, c := range clusters {
			total += c.weight
			if c.weight < clusters[smallest].weight {
				smallest = i
			}
		}
		if clusters[smallest].weight >= mixMinClusterShare*total {
			break
		}
		remaining := make([][]float32, 0, len(clusters)-1)
		for i, c := range clusters {
			if i != smallest {
				remaining = append(remaining, c.centroid)
			}
		}
		clusters = groupClusters(signals, nearestCenters(signals, remaining), len(remaining))
	}
	return clusters
}

// seedCenters picks up to k starting centers: the first by weight, each next one by weight × (cosine distance to the
// nearest chosen center)², so centers start spread across different tastes.
func seedCenters(signals []tasteSignal, k int, rng *rand.Rand) [][]float32 {
	first := pickWeighted(len(signals), func(i int) float64 { return signals[i].weight }, rng)
	centers := [][]float32{signals[first].vec}
	for len(centers) < k {
		next := pickWeighted(len(signals), func(i int) float64 {
			d := 1 - bestSimilarity(signals[i].vec, centers)
			return signals[i].weight * d * d
		}, rng)
		if next < 0 {
			break // every signal already sits on a center
		}
		centers = append(centers, signals[next].vec)
	}
	return centers
}

// pickWeighted returns an index with probability proportional to score(i), or -1 when every score is 0.
func pickWeighted(n int, score func(int) float64, rng *rand.Rand) int {
	scores := make([]float64, n)
	total := 0.0
	for i := range n {
		scores[i] = math.Max(score(i), 0)
		total += scores[i]
	}
	if total <= 0 {
		return -1
	}
	r := rng.Float64() * total
	last := -1
	for i, s := range scores {
		if s <= 0 {
			continue
		}
		last = i
		r -= s
		if r < 0 {
			return i
		}
	}
	return last // floating-point leftovers land on the last positive score
}

func bestSimilarity(v []float32, centers [][]float32) float64 {
	best := -1.0
	for _, c := range centers {
		if c != nil {
			best = math.Max(best, dot(v, c))
		}
	}
	return best
}

// nearestCenters assigns each signal to its most similar non-nil center.
func nearestCenters(signals []tasteSignal, centers [][]float32) []int {
	assign := make([]int, len(signals))
	for i, s := range signals {
		best, bestSim := -1, 0.0
		for c, center := range centers {
			if center == nil {
				continue
			}
			if sim := dot(s.vec, center); best < 0 || sim > bestSim {
				best, bestSim = c, sim
			}
		}
		assign[i] = best
	}
	return assign
}

// centersOf recomputes each center as the weighted centroid of its members; a center with no members becomes nil.
func centersOf(signals []tasteSignal, assign []int, k int) [][]float32 {
	members := make([][]int, k)
	for i, c := range assign {
		if c >= 0 {
			members[c] = append(members[c], i)
		}
	}
	centers := make([][]float32, k)
	for c, idx := range members {
		if len(idx) > 0 {
			centers[c] = weightedCentroid(signals, idx)
		}
	}
	return centers
}

func groupClusters(signals []tasteSignal, assign []int, k int) []tasteCluster {
	members := make([][]int, k)
	for i, c := range assign {
		if c >= 0 {
			members[c] = append(members[c], i)
		}
	}
	clusters := []tasteCluster{}
	for _, idx := range members {
		if len(idx) == 0 {
			continue
		}
		w := 0.0
		for _, i := range idx {
			w += signals[i].weight
		}
		clusters = append(clusters, tasteCluster{members: idx, centroid: weightedCentroid(signals, idx), weight: w})
	}
	return clusters
}

// ── Filling a mix ───────────────────────────────────────────────────────────────

type mixPick struct {
	song    MixSong
	artists []string // display names, for the per-artist cap and the subtitle
}

// isPlaceholderArtist reports catalog fillers that mean "no artist": "-", ".", "[none]", "Unknown Artist".
func isPlaceholderArtist(name string) bool {
	switch strings.ToLower(name) {
	case "", "[none]", "none", "unknown", "unknown artist":
		return true
	}
	return strings.IndexFunc(name, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0
}

func toPick(c repositories.MixCandidate) mixPick {
	names := []string{}
	for _, a := range utils.ParseArtists(c.Artist.String) {
		if a = strings.TrimSpace(a); !isPlaceholderArtist(a) {
			names = append(names, a)
		}
	}
	song := MixSong{ID: c.SongID, Title: c.Title}
	if len(names) > 0 {
		joined := strings.Join(names, ", ")
		song.Artist = &joined
	}
	if key := strings.TrimSpace(c.Key.String); c.Key.Valid && key != "" {
		song.Key = &key
	}
	return mixPick{song: song, artists: names}
}

// fillMix picks up to mixSize songs from nearest-first candidates, with no artist more than mixMaxPerArtist times.
// With interleave, every mixFamiliarEvery-th slot (starting with the first) goes to a song in known and the others to
// songs the user hasn't touched, falling back to the other group when one runs out: familiar enough to trust, with
// new songs to discover. Without interleave it fills purely by distance.
// Returns the songs and a subtitle naming the most frequent artists.
func fillMix(candidates []repositories.MixCandidate, known map[int]bool, interleave bool) ([]MixSong, string) {
	var familiar, fresh []mixPick
	for _, c := range candidates {
		if interleave && known[c.SongID] {
			familiar = append(familiar, toPick(c))
		} else {
			fresh = append(fresh, toPick(c))
		}
	}

	perArtist := map[string]int{}
	seen := map[int]bool{}
	allowed := func(p mixPick) bool {
		if seen[p.song.ID] {
			return false
		}
		for _, a := range p.artists {
			if perArtist[strings.ToLower(a)] >= mixMaxPerArtist {
				return false
			}
		}
		return true
	}
	// take pops picks off q until one is allowed. A pick over the artist cap never becomes allowed later.
	take := func(q *[]mixPick) (mixPick, bool) {
		for len(*q) > 0 {
			p := (*q)[0]
			*q = (*q)[1:]
			if allowed(p) {
				return p, true
			}
		}
		return mixPick{}, false
	}

	chosen := []mixPick{}
	for slot := 0; len(chosen) < mixSize; slot++ {
		first, second := &fresh, &familiar
		if interleave && slot%mixFamiliarEvery == 0 {
			first, second = &familiar, &fresh
		}
		p, ok := take(first)
		if !ok {
			p, ok = take(second)
		}
		if !ok {
			break
		}
		chosen = append(chosen, p)
		seen[p.song.ID] = true
		for _, a := range p.artists {
			perArtist[strings.ToLower(a)]++
		}
	}

	songs := make([]MixSong, len(chosen))
	for i, p := range chosen {
		songs[i] = p.song
	}
	return songs, mixSubtitle(chosen)
}

// mixSubtitle names up to three artists by how often they appear, like "Elevation Worship, Bethel Music, JPCC Worship".
func mixSubtitle(picks []mixPick) string {
	type tally struct {
		name  string
		count int
	}
	byKey := map[string]*tally{}
	order := []*tally{} // first-appearance order, so ties keep the artist heard first
	for _, p := range picks {
		for _, a := range p.artists {
			key := strings.ToLower(a)
			if t, ok := byKey[key]; ok {
				t.count++
				continue
			}
			t := &tally{name: a, count: 1}
			byKey[key] = t
			order = append(order, t)
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].count > order[j].count })
	names := make([]string, 0, 3)
	for _, t := range order {
		if len(names) == 3 {
			break
		}
		names = append(names, t.name)
	}
	return strings.Join(names, ", ")
}

// roundRobin interleaves nearest-first candidate lists (one per taste cluster) into one list, taking from every
// taste in turn — the Supermix. Songs in skip, and repeats, are left out.
func roundRobin(lists [][]repositories.MixCandidate, skip map[int]bool) []repositories.MixCandidate {
	out := []repositories.MixCandidate{}
	seen := map[int]bool{}
	for i := 0; ; i++ {
		more := false
		for _, list := range lists {
			if i >= len(list) {
				continue
			}
			more = true
			c := list[i]
			if skip[c.SongID] || seen[c.SongID] {
				continue
			}
			seen[c.SongID] = true
			out = append(out, c)
		}
		if !more {
			return out
		}
	}
}
