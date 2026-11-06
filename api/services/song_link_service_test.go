package services

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixtures mirror the markup of real chord sites; the lyrics are made up.

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

// ── ParseSongPage: one fixture per site layout ────────────────────────────────

func TestParseSongPage_PreWithDataKey(t *testing.T) {
	// jrchord / chordrohani: a <pre data-key> chord-over-lyrics block.
	page := `<html><head><title>Ignored | Site</title></head><body>
		<h1 class="song-detail-title">Langit Baru</h1><div class="song-detail-artist">Tim Pujian</div>
		<div class="song p-4"><div class="song-title">Related Song</div></div>
		<pre data-key="G">
Intro : G  D  Em  C

Bait :
G          D
  Langit baru bumi baru
Em         C
  Semua kita nyanyikan
G          D/F#     Em
  Hari ini penuh syukur
C          D        G
  Bersama umat-Nya
</pre></body></html>`

	res := ParseSongPage(mustURL(t, "https://www.jrchord.com/langit-baru"), []byte(page))

	assert.Equal(t, models.SongLinkSupported, res.Status)
	assert.Equal(t, "jrchord.com", res.Source)
	assert.Equal(t, "Langit Baru", res.Title)
	assert.Equal(t, "Tim Pujian", res.Artist)
	assert.Equal(t, "G", res.Key)
	assert.Equal(t, "lyrics_chords", res.LyricsType)
	assert.True(t, strings.HasPrefix(res.Content, "Intro : G  D  Em  C\n\nBait :\nG          D\n  Langit baru bumi baru"), res.Content)
}

func TestParseSongPage_LineClassedLayout(t *testing.T) {
	// psalmnote: part-tag buttons, then separate .chords-line / .lyric rows; folded parts show "[...]".
	page := `<html><body>
		<h1 class="songviewer-h1">Kasih Yang Teguh <svg><path d="M0"/></svg></h1>
		<button class="btn-cta" data-songtitle="Kasih Yang Teguh" data-artistname="Paduan Suara">Share</button>
		<div class="songviewer-content">
		  <div>Key: <span data-cy="chord-base">D</span></div>
		  <div><button class="song-part-tag"><span>[</span>Verse<span>]</span></button>
		    <div><section class="chords-line">  D        G</section></div>
		    <div><section class="lyric">Kasih-Mu teguh selamanya</section></div>
		    <div><section class="chords-line">     A     D</section></div>
		    <div><section class="lyric">Tak pernah berubah</section></div>
		    <span class="collapsed-state d-none">[...]</span>
		  </div>
		  <div><button class="song-part-tag"><span>[</span>Chorus<span>]</span></button>
		    <div><section class="chords-line">G     A</section></div>
		    <div><section class="lyric">Haleluya</section></div>
		  </div>
		</div></body></html>`

	res := ParseSongPage(mustURL(t, "https://www.psalmnote.com/song/kasih"), []byte(page))

	require.Equal(t, models.SongLinkSupported, res.Status)
	assert.Equal(t, "Kasih Yang Teguh", res.Title)
	assert.Equal(t, "Paduan Suara", res.Artist)
	assert.Equal(t, "D", res.Key)
	assert.Equal(t, "Verse:\n  D        G\nKasih-Mu teguh selamanya\n     A     D\nTak pernah berubah\n\nChorus:\nG     A\nHaleluya", res.Content)
}

func TestParseSongPage_WordPastedBloggerPost(t *testing.T) {
	// Blogger post pasted from Word: one <div> per line, chords aligned with runs of &nbsp;,
	// ordinary source newlines inside the markup must not add spaces.
	line := func(s string) string { return `<div style="margin:0"><b><span>` + s + "</span></b></div>\n" }
	body := `<p>Selamat datang di blog kami, berikut chord lagunya.</p>` +
		line("Reff:") +
		line("C &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;\n<span>&nbsp;&nbsp;</span>G") +
		line("Hatiku memuji") +
		line("Am&nbsp;&nbsp;&nbsp;&nbsp; F") +
		line("Namamu selalu") +
		line("C&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; G&nbsp;&nbsp;&nbsp; C") +
		line("Sampai akhir zaman") +
		line("Haleluya amin") +
		`<div><br></div><div><br></div><p>Baca juga: lagu lainnya https://example.com</p>`
	page := `<html><head><meta property="og:title" content="x"></head><body>
		<h3 class="post-title entry-title">Lirik dan Chord Lagu Rohani " Hatiku Memuji "</h3>
		<div class="post-body entry-content">` + body + `</div></body></html>`

	res := ParseSongPage(mustURL(t, "https://contoh.blogspot.com/2020/01/hatiku.html"), []byte(page))

	require.Equal(t, models.SongLinkSupported, res.Status)
	assert.Equal(t, "Hatiku Memuji", res.Title)
	// "C ␣ 5×nbsp ⏎ 2×nbsp G": a browser keeps both collapsible spaces because they sit next to &nbsp;, not each other.
	assert.Equal(t, "Reff:\nC         G\nHatiku memuji\nAm     F\nNamamu selalu\nC      G    C\nSampai akhir zaman\nHaleluya amin", res.Content)
}

func TestParseSongPage_CollectionPage(t *testing.T) {
	page := `<html><body>
		<h3 class="post-title">Kumpulan Lirik dan Chord Lagu Rohani</h3>
		<div class="post-body">
		  <h3><a href="https://contoh.blogspot.com/a.html">Lirik dan Chord "Lagu Satu"</a></h3>
		  <div>C G</div><div>Kata pertama</div>
		  <h3><a href="/b.html">Chord Pujian Dua</a></h3>
		  <h3><a href="https://contoh.blogspot.com/c.html#top">Lagu Tiga</a></h3>
		  <h3><a href="https://contoh.blogspot.com/a.html">Lagu Satu (again)</a></h3>
		  <h3><a href="https://contoh.blogspot.com/kumpulan.html">This page</a></h3>
		</div></body></html>`

	res := ParseSongPage(mustURL(t, "https://contoh.blogspot.com/kumpulan.html"), []byte(page))

	require.Equal(t, models.SongLinkCollection, res.Status)
	assert.Equal(t, []models.SongLinkItem{
		{Title: "Lagu Satu", URL: "https://contoh.blogspot.com/a.html"},
		{Title: "Pujian Dua", URL: "https://contoh.blogspot.com/b.html"},
		{Title: "Lagu Tiga", URL: "https://contoh.blogspot.com/c.html"},
	}, res.Songs)
	assert.Empty(t, res.Content)
}

func TestParseSongPage_Unsupported(t *testing.T) {
	cases := map[string]string{
		"plain article": `<article><h1>Renungan</h1><p>Tuhan adalah gembalaku.</p><p>A B C dari iman.</p></article>`,
		// Chord names without lyrics under them are a theory lesson, not a song.
		"progression only": `<article><pre>C G Am F
C G F C
Am F C G
Dm G C</pre></article>`,
		"lyrics without chords": `<article><pre>Bapa sungguh baik
KasihMu kekal
Selamanya</pre></article>`,
	}
	for name, page := range cases {
		t.Run(name, func(t *testing.T) {
			res := ParseSongPage(mustURL(t, "https://example.com/x"), []byte(page))
			assert.Equal(t, models.SongLinkUnsupported, res.Status)
			assert.NotEmpty(t, res.Reason)
			assert.Empty(t, res.Content)
		})
	}
}

// ── Building blocks ───────────────────────────────────────────────────────────

func TestIsChordLine(t *testing.T) {
	chordLines := []string{
		"C", "  D#       F", "Bb       Bb/D", "F#m7b5  E7#9  Cmaj7", "Dsus4  D", "Am7/G  C2  Gsus",
		"Intro : G  D  Em  C", "| C . . . | G . . . |", "N.C.   C   x2", "(Am)  F",
	}
	for _, l := range chordLines {
		assert.True(t, isChordLine(l), "expected chord line: %q", l)
	}
	lyricLines := []string{
		"", "Above All Powers", "I Am", "Dia baik", "Reff : C G", "A Mighty Fortress", "4/4", "Tempo : 72",
	}
	for _, l := range lyricLines {
		assert.False(t, isChordLine(l), "expected non-chord line: %q", l)
	}
}

func TestCleanTitle(t *testing.T) {
	cases := map[string]string{
		`Lirik dan Chord Lagu Rohani " Janji Mu Sperti Fajar"`: "Janji Mu Sperti Fajar",
		"Chord Tiap Langkahku":                                 "Tiap Langkahku",
		"Kunci Gitar Bapa Engkau Sungguh Baik":                 "Bapa Engkau Sungguh Baik",
		"Engkaulah Perisaiku | Chord Rohani":                   "Engkaulah Perisaiku",
		"Above All Chords by Michael W. Smith":                 "Above All",
		"  Langit   Baru  ":                                    "Langit Baru",
	}
	for in, want := range cases {
		assert.Equal(t, want, cleanTitle(in), in)
	}
}

// ── SongLinkService: fetch outcomes ───────────────────────────────────────────

type fakeFetcher struct {
	page *providers.FetchedPage
	err  error
}

func (f fakeFetcher) Fetch(context.Context, string) (*providers.FetchedPage, error) {
	return f.page, f.err
}

func TestSongLinkService_CheckLink(t *testing.T) {
	t.Run("invalid link is an error", func(t *testing.T) {
		for _, raw := range []string{"", "not a url", "ftp://x.com/a", "javascript:alert(1)", "http://127.0.0.1/", "https://x.com:8443/a", "https://u:p@x.com/"} {
			_, err := NewSongLinkService(fakeFetcher{}).CheckLink(context.Background(), raw)
			assert.ErrorIs(t, err, ErrInvalidSongLink, raw)
		}
	})

	t.Run("fetch failures become results", func(t *testing.T) {
		cases := []struct {
			err    error
			status string
		}{
			{providers.ErrNotHTML, models.SongLinkUnsupported},
			{providers.ErrBlockedHost, models.SongLinkUnreachable},
			{errors.New("page returned HTTP 404"), models.SongLinkUnreachable},
		}
		for _, c := range cases {
			res, err := NewSongLinkService(fakeFetcher{err: c.err}).CheckLink(context.Background(), "https://www.chords.example/x")
			require.NoError(t, err)
			assert.Equal(t, c.status, res.Status, c.err.Error())
			assert.Equal(t, "chords.example", res.Source)
			assert.NotEmpty(t, res.Reason)
		}
	})

	t.Run("parses the final URL after redirects", func(t *testing.T) {
		final := mustURL(t, "https://www.jrchord.com/song")
		html := `<pre data-key="C">C   G
Satu dua
Am  F
Tiga empat
C   G
Lima enam</pre>`
		res, err := NewSongLinkService(fakeFetcher{page: &providers.FetchedPage{URL: final, HTML: []byte(html)}}).
			CheckLink(context.Background(), "https://short.link/abc")
		require.NoError(t, err)
		assert.Equal(t, models.SongLinkSupported, res.Status)
		assert.Equal(t, "https://www.jrchord.com/song", res.URL)
		assert.Equal(t, "jrchord.com", res.Source)
	})
}
