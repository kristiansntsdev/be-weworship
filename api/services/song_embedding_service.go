package services

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"be-songbanks-v1/api/providers"
	"be-songbanks-v1/api/repositories"
	"be-songbanks-v1/api/utils"
)

const (
	// chunkerVersion is part of the content hash: bump it when BuildSongChunks changes so the backfill
	// re-embeds every song instead of skipping them as unchanged.
	chunkerVersion   = "2"
	maxChunksPerSong = 12
	// Stanzas shorter than this are merged with the next one. Many songs put a blank line between every lyric
	// line, and short generic lines ("Kaulah Tuhan") would otherwise sit close to almost every query.
	minChunkWords = 12
	// Chunks with fewer distinct words are dropped ("Yesus.. Yesus.. Yesus").
	minDistinctWords = 5
	// Cosine-distance cut-off for semantic matches. On bge-m3 a translation or typo'd version of a line sits
	// around 0.1–0.15 and unrelated text around 0.6, so 0.5 keeps related songs without flooding results.
	semanticMaxDistance    = 0.5
	semanticCandidateLimit = 30
	// Query vectors kept per instance so "Load more" and repeated searches skip the embedding call
	// (~4 KB each for bge-m3, so ~2 MB at most).
	queryVectorCacheSize = 500
)

var (
	reBlankLines   = regexp.MustCompile(`\n\s*\n`)
	reSectionLabel = regexp.MustCompile(`(?i)^[\[(]?\s*(intro|verse|vers|bait|chorus|reff?|refrein|bridge|pre-?chorus|outro|ending|interlude|instrumen(t|tal)?|music|tag|coda)\s*\d*\s*[:\])]?\s*$`)
)

// SongEmbeddingService keeps song_embeddings in sync and answers semantic (meaning-based) song queries.
type SongEmbeddingService struct {
	repo     repositories.EmbeddingRepoIface
	embedder providers.EmbedderIface
	queries  *vectorCache
}

// NewSongEmbeddingService returns nil when no embedder is configured, which disables semantic search.
func NewSongEmbeddingService(repo repositories.EmbeddingRepoIface, embedder providers.EmbedderIface) *SongEmbeddingService {
	if embedder == nil {
		return nil
	}
	return &SongEmbeddingService{repo: repo, embedder: embedder, queries: newVectorCache(queryVectorCacheSize)}
}

// vectorCache is a small LRU of query text → embedding. A query's vector depends only on its text and the model,
// so entries never go stale when songs change.
type vectorCache struct {
	mu    sync.Mutex
	max   int
	order *list.List // front = most recently used
	items map[string]*list.Element
}

type vectorEntry struct {
	key string
	vec []float32
}

func newVectorCache(max int) *vectorCache {
	return &vectorCache{max: max, order: list.New(), items: map[string]*list.Element{}}
}

func (c *vectorCache) get(key string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*vectorEntry).vec, true
}

func (c *vectorCache) put(key string, vec []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		el.Value.(*vectorEntry).vec = vec
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(&vectorEntry{key: key, vec: vec})
	if c.order.Len() > c.max {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*vectorEntry).key)
	}
}

func letterCount(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

func distinctWords(s string) int {
	seen := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(s)) {
		w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		if w != "" {
			seen[w] = true
		}
	}
	return len(seen)
}

// BuildSongChunks returns the texts embedded for a song: chunk 0 is title + artist + all lyrics (bge-m3 reads
// up to 8k tokens), the rest are its distinct stanzas so a remembered line lands close to its own stanza.
// Section labels ("Reff :", "Verse 1") and chord/notation leftovers are dropped, repeated choruses count once,
// and short stanzas are merged until a chunk has at least minChunkWords words.
func BuildSongChunks(title, artist, plainLyrics string) []string {
	var blocks []string
	seen := map[string]bool{}
	for _, block := range reBlankLines.Split(plainLyrics, -1) {
		var lines []string
		for _, line := range strings.Split(block, "\n") {
			l := strings.TrimSpace(line)
			if l == "" || reSectionLabel.MatchString(l) || letterCount(l) < 2 {
				continue
			}
			lines = append(lines, l)
		}
		text := strings.Join(lines, "\n")
		key := strings.ToLower(text)
		if text == "" || seen[key] {
			continue
		}
		seen[key] = true
		blocks = append(blocks, text)
	}

	var stanzas []string
	var buf []string
	flush := func() {
		text := strings.Join(buf, "\n")
		buf = nil
		if len(strings.Fields(text)) < minChunkWords && len(stanzas) > 0 {
			stanzas[len(stanzas)-1] += "\n" + text // short tail: keep it with the stanza before
			return
		}
		if distinctWords(text) >= minDistinctWords {
			stanzas = append(stanzas, text)
		}
	}
	for _, b := range blocks {
		buf = append(buf, b)
		if len(strings.Fields(strings.Join(buf, " "))) >= minChunkWords {
			flush()
		}
	}
	if len(buf) > 0 {
		flush()
	}

	whole := strings.TrimSpace(title)
	if a := strings.TrimSpace(artist); a != "" {
		whole += "\n" + a
	}
	if len(stanzas) > 0 {
		whole += "\n\n" + strings.Join(stanzas, "\n\n")
	}
	chunks := []string{whole}
	for _, st := range stanzas {
		if len(chunks) >= maxChunksPerSong {
			break
		}
		chunks = append(chunks, st)
	}
	return chunks
}

func songContentHash(model, title, artist, lyrics string) string {
	sum := sha256.Sum256([]byte(chunkerVersion + "\x00" + model + "\x00" + title + "\x00" + artist + "\x00" + lyrics))
	return hex.EncodeToString(sum[:])
}

func embeddingText(src *repositories.EmbeddingSource) (artist, lyrics string) {
	artist = strings.Join(utils.ParseArtists(src.Artist.String), ", ")
	lyrics = src.PlainLyrics.String
	if strings.TrimSpace(lyrics) == "" {
		lyrics = utils.ExtractPlainLyrics(src.LyricsAndChord.String)
	}
	return artist, lyrics
}

// index embeds one song unless its text is unchanged since it was last embedded (or force is set).
func (s *SongEmbeddingService) index(ctx context.Context, src *repositories.EmbeddingSource, prevHash string, force bool) (bool, error) {
	artist, lyrics := embeddingText(src)
	hash := songContentHash(s.embedder.Model(), src.Title, artist, lyrics)
	if !force && hash == prevHash {
		return false, nil
	}
	texts := BuildSongChunks(src.Title, artist, lyrics)
	vectors, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return false, err
	}
	chunks := make([]repositories.SongChunk, len(texts))
	for i, text := range texts {
		chunks[i] = repositories.SongChunk{Index: i, Text: text, Vector: vectors[i]}
	}
	if err := s.repo.ReplaceSongChunks(src.ID, s.embedder.Model(), hash, chunks); err != nil {
		return false, err
	}
	return true, nil
}

// IndexSong (re)embeds one song if its title, artist or lyrics changed. Deleted songs are a no-op
// (their rows go with ON DELETE CASCADE).
func (s *SongEmbeddingService) IndexSong(ctx context.Context, songID int) (bool, error) {
	src, err := s.repo.Source(songID)
	if err != nil || src == nil {
		return false, err
	}
	hashes, err := s.repo.StoredHashes(s.embedder.Model())
	if err != nil {
		return false, err
	}
	return s.index(ctx, src, hashes[songID], false)
}

// BackfillResult summarises a Backfill run.
type BackfillResult struct {
	Embedded int
	Skipped  int
	Failed   map[int]error
}

// Backfill embeds every song whose text changed since its last embedding (all songs with force).
// A failing song is recorded and skipped so one bad row does not stop the run; re-running retries it.
func (s *SongEmbeddingService) Backfill(ctx context.Context, force bool, progress func(done, total int)) (BackfillResult, error) {
	result := BackfillResult{Failed: map[int]error{}}
	sources, err := s.repo.AllSources()
	if err != nil {
		return result, err
	}
	hashes, err := s.repo.StoredHashes(s.embedder.Model())
	if err != nil {
		return result, err
	}
	for i := range sources {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		embedded, err := s.index(ctx, &sources[i], hashes[sources[i].ID], force)
		switch {
		case err != nil:
			result.Failed[sources[i].ID] = fmt.Errorf("%s: %w", sources[i].Title, err)
		case embedded:
			result.Embedded++
		default:
			result.Skipped++
		}
		if progress != nil {
			progress(i+1, len(sources))
		}
	}
	return result, nil
}

// SemanticMatches returns songs whose meaning is close to the query, nearest first.
func (s *SongEmbeddingService) SemanticMatches(ctx context.Context, query string) ([]repositories.SongDistance, error) {
	query = strings.Join(strings.Fields(query), " ")
	vec, ok := s.queries.get(query)
	if !ok {
		vectors, err := s.embedder.Embed(ctx, []string{query})
		if err != nil {
			return nil, err
		}
		vec = vectors[0]
		s.queries.put(query, vec)
	}
	return s.repo.NearestSongs(vec, s.embedder.Model(), semanticMaxDistance, semanticCandidateLimit)
}
