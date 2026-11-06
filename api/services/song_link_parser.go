package services

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"be-songbanks-v1/api/models"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	minChordLines       = 3     // fewer chord lines than this is not a chord sheet
	minPairedLyricLines = 2     // chord lines directly followed by a lyric line
	minCollectionSongs  = 3     // song-title headings in one article that make it a list page
	maxCollectionSongs  = 50    // cap on links returned for a list page
	maxSongLinkContent  = 20000 // runes; a chord sheet is a few KB at most
	maxTrailingLyrics   = 16    // un-chorded lines kept after the last chord line (a final chorus)
)

// ParseSongPage recognises a chord sheet in a downloaded page. Each chord site lays its songs out differently,
// so extractors run from most to least specific; the first one that yields a chord sheet wins.
// pageURL is the final URL after redirects and resolves relative links on list pages.
func ParseSongPage(pageURL *url.URL, body []byte) models.SongLinkCheck {
	res := models.SongLinkCheck{URL: pageURL.String(), Source: sourceName(pageURL)}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		res.Status = models.SongLinkUnsupported
		res.Reason = "We couldn't read this page."
		return res
	}
	res.Title = songTitle(doc)
	res.Artist = songArtist(doc)
	res.Key = songKey(doc)

	// 1. Line-classed layouts (psalmnote): chords and lyrics are separate elements, not one text block.
	if sheet := lineClassedSheet(doc); sheet != "" {
		return supported(res, sheet)
	}
	// 2. <pre> chord sheets (jrchord, chordrohani and most WordPress/Blogger transposer themes).
	if sheet := preSheet(doc); sheet != "" {
		return supported(res, sheet)
	}
	// 3. Free-form article body (Blogger posts pasted from Word, aligned with &nbsp;).
	article, isArticle := articleRoot(doc)
	if isArticle {
		if songs := listedSongs(article, pageURL); len(songs) >= minCollectionSongs {
			res.Status = models.SongLinkCollection
			res.Songs = songs
			res.Reason = "This page lists several songs. Pick the one you want."
			return res
		}
	}
	if sheet := extractChordSheet(renderLines(article)); sheet != "" {
		return supported(res, sheet)
	}

	res.Status = models.SongLinkUnsupported
	res.Reason = "We couldn't find chords on this page."
	return res
}

func supported(res models.SongLinkCheck, sheet string) models.SongLinkCheck {
	res.Status = models.SongLinkSupported
	res.LyricsType = "lyrics_chords"
	res.Content = sheet
	return res
}

// ── Extractors ───────────────────────────────────────────────────────────────

// lineClassedSheet rebuilds psalmnote's layout: [song-part-tag] buttons, then .chords-line / .lyric rows.
func lineClassedSheet(doc *html.Node) string {
	root := findFirst(doc, func(n *html.Node) bool { return hasClass(n, "songviewer-content") })
	if root == nil {
		return ""
	}
	var lines []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case hasClass(n, "collapsed-state"): // the "[...]" placeholder shown when a part is folded
				return
			case hasClass(n, "song-part-tag"):
				if len(lines) > 0 {
					lines = append(lines, "")
				}
				lines = append(lines, strings.Trim(collapsedText(n), "[] ")+":")
				return
			case hasClass(n, "chords-line"), hasClass(n, "lyric"):
				lines = append(lines, rawText(n))
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	if countChordLines(lines) < minChordLines {
		return ""
	}
	return finishSheet(lines)
}

// preSheet picks the <pre> with the most chord lines.
func preSheet(doc *html.Node) string {
	best, bestCount := "", 0
	for _, pre := range findAll(doc, func(n *html.Node) bool { return n.DataAtom == atom.Pre }) {
		lines := strings.Split(rawText(pre), "\n")
		if n := countChordLines(lines); n > bestCount && isChordSheet(lines) {
			best, bestCount = finishSheet(lines), n
		}
	}
	return best
}

// extractChordSheet cuts the song out of an article's text: from the first chord line (plus any section label
// right above it) to the last chord line, then the un-chorded lyric lines that follow it.
func extractChordSheet(lines []string) string {
	if !isChordSheet(lines) {
		return ""
	}
	var chordIdx []int
	for i, l := range lines {
		if isChordLine(l) {
			chordIdx = append(chordIdx, i)
		}
	}
	start, last := chordIdx[0], chordIdx[len(chordIdx)-1]
	for k := start - 1; k >= 0 && start-k <= 2; k-- {
		if isSectionLabel(lines[k]) {
			start = k
		} else if strings.TrimSpace(lines[k]) != "" {
			break
		}
	}
	end, blanks, kept := last, 0, 0
	for k := last + 1; k < len(lines) && kept < maxTrailingLyrics; k++ {
		t := strings.TrimSpace(lines[k])
		if t == "" {
			if blanks++; blanks >= 2 {
				break
			}
			continue
		}
		if !isTrailingLyric(t) {
			break
		}
		blanks, end, kept = 0, k, kept+1
	}
	return finishSheet(lines[start : end+1])
}

// listedSongs returns the song links of a list page: headings inside the article that link to another page.
func listedSongs(article *html.Node, pageURL *url.URL) []models.SongLinkItem {
	self := stripFragment(pageURL)
	seen := map[string]bool{self: true}
	var out []models.SongLinkItem
	for _, h := range findAll(article, isHeading) {
		a := findFirst(h, func(n *html.Node) bool { return n.DataAtom == atom.A && attr(n, "href") != "" })
		if a == nil {
			continue
		}
		u, err := pageURL.Parse(attr(a, "href"))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		key := stripFragment(u)
		title := cleanTitle(collapsedText(h))
		if seen[key] || title == "" {
			continue
		}
		seen[key] = true
		out = append(out, models.SongLinkItem{Title: title, URL: key})
		if len(out) == maxCollectionSongs {
			break
		}
	}
	return out
}

// ── Metadata ─────────────────────────────────────────────────────────────────

// Known title elements, most specific first. Add a site's selector here when its <h1> isn't the song title.
var titleClasses = []string{"song-detail-title", "songviewer-h1", "post-title-h", "post-title", "entry-title"}

func songTitle(doc *html.Node) string {
	for _, cls := range titleClasses {
		if n := findFirst(doc, func(n *html.Node) bool { return hasClass(n, cls) }); n != nil {
			if t := cleanTitle(collapsedText(n)); t != "" {
				return t
			}
		}
	}
	if m := findFirst(doc, func(n *html.Node) bool {
		return n.DataAtom == atom.Meta && attr(n, "property") == "og:title"
	}); m != nil {
		if t := cleanTitle(attr(m, "content")); t != "" {
			return t
		}
	}
	for _, a := range []atom.Atom{atom.H1, atom.Title} {
		if n := findFirst(doc, func(n *html.Node) bool { return n.DataAtom == a }); n != nil {
			if t := cleanTitle(collapsedText(n)); t != "" {
				return t
			}
		}
	}
	return ""
}

func songArtist(doc *html.Node) string {
	if n := findFirst(doc, func(n *html.Node) bool { return hasClass(n, "song-detail-artist") }); n != nil {
		return collapsedText(n)
	}
	if n := findFirst(doc, func(n *html.Node) bool { return attr(n, "data-artistname") != "" }); n != nil {
		return strings.TrimSpace(attr(n, "data-artistname"))
	}
	return ""
}

func songKey(doc *html.Node) string {
	if n := findFirst(doc, func(n *html.Node) bool { return n.DataAtom == atom.Pre && attr(n, "data-key") != "" }); n != nil {
		return strings.TrimSpace(attr(n, "data-key"))
	}
	if n := findFirst(doc, func(n *html.Node) bool { return attr(n, "data-cy") == "chord-base" }); n != nil {
		return collapsedText(n)
	}
	return ""
}

var (
	titleQuoted = regexp.MustCompile(`["“”]\s*([^"“”]{2,}?)\s*["“”]`)
	titlePrefix = regexp.MustCompile(`(?i)^(?:(?:lirik|chord|kunci gitar|lyrics?)(?:\s+(?:dan|&|and)\s+(?:lirik|chord|kunci gitar))?(?:\s+lagu)?(?:\s+rohani)?(?:\s+kristen)?\s*[:\-–]?\s+)`)
	titleSuffix = regexp.MustCompile(`(?i)\s*(?:[|–]\s.*|\s+-\s+(?:lirik|chord|kunci).*|\s+chords?(?:\s+by\s+.*)?)$`)
)

// cleanTitle turns "Lirik dan Chord Lagu Rohani “Janji Mu Sperti Fajar”" into "Janji Mu Sperti Fajar".
func cleanTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if m := titleQuoted.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	s = titlePrefix.ReplaceAllString(s, "")
	s = titleSuffix.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// ── Chord detection ──────────────────────────────────────────────────────────

// A chord symbol: root, accidental, quality, extensions, slash bass. C, F#m7, Bb/D, Dsus4, Cmaj7, Am7/G, E7#9.
var chordToken = regexp.MustCompile(`^[A-G](?:#|b)?(?:maj|min|m|M|dim|aug|sus|add|\+|°|ø)?\d{0,2}(?:(?:sus|add|maj|no|b|#)\d{0,2})*(?:/[A-G](?:#|b)?)?$`)

// Tokens that may sit on a chord line without making it a lyric line.
var chordFiller = regexp.MustCompile(`(?i)^(?:n\.?c\.?|x\d+|\d+x|/+|-+|\.+|%|\*|intro|interlude|outro|ending|coda|musik|music|int|instrumental)$`)

var sectionLabel = regexp.MustCompile(`(?i)^\W*(?:intro|verse|bait|ayat|reff?|refrain|chorus|pre-?chorus|pre-?reff?|bridge|interlude|outro|ending|coda|musik|int\.?|tag|hook|reff?rein)\b`)

func isChordLine(line string) bool {
	chords := 0
	for _, f := range strings.FieldsFunc(line, func(r rune) bool { return unicode.IsSpace(r) || r == '|' || r == ',' }) {
		f = strings.Trim(f, "()[]{}:.")
		switch {
		case f == "":
		case chordToken.MatchString(f):
			chords++
		case chordFiller.MatchString(f):
		default:
			return false
		}
	}
	return chords > 0
}

func isLyricLine(line string) bool {
	t := strings.TrimSpace(line)
	return utf8.RuneCountInString(t) >= 2 && !isChordLine(t) && strings.IndexFunc(t, unicode.IsLetter) >= 0
}

func isSectionLabel(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && utf8.RuneCountInString(t) <= 30 && (sectionLabel.MatchString(t) || strings.HasSuffix(t, ":"))
}

var boilerplate = regexp.MustCompile(`(?i)https?://|www\.|baca juga|lihat juga|download|share|bagikan|copyright|©|komentar|comment|posted by|diposting`)

func isTrailingLyric(t string) bool {
	return utf8.RuneCountInString(t) <= 70 && !boilerplate.MatchString(t)
}

func countChordLines(lines []string) int {
	n := 0
	for _, l := range lines {
		if isChordLine(l) {
			n++
		}
	}
	return n
}

// isChordSheet: enough chord lines, and enough of them sit directly above a lyric line.
// The pairing check keeps out pages that merely mention chord names or list a chord progression.
func isChordSheet(lines []string) bool {
	chords, paired := 0, 0
	for i, l := range lines {
		if !isChordLine(l) {
			continue
		}
		chords++
		if i+1 < len(lines) && isLyricLine(lines[i+1]) {
			paired++
		}
	}
	return chords >= minChordLines && paired >= minPairedLyricLines
}

// finishSheet normalises a chord sheet: no trailing spaces, common indent removed (relative chord alignment
// is kept), at most one blank line in a row, no blank edges, capped length.
func finishSheet(lines []string) string {
	cleaned := make([]string, 0, len(lines))
	indent := -1
	for _, l := range lines {
		l = strings.TrimRight(strings.ReplaceAll(strings.ReplaceAll(l, " ", " "), "\t", "    "), " \r")
		cleaned = append(cleaned, l)
		if l == "" {
			continue
		}
		if lead := len(l) - len(strings.TrimLeft(l, " ")); indent < 0 || lead < indent {
			indent = lead
		}
	}
	var out []string
	for _, l := range cleaned {
		if l == "" {
			if len(out) == 0 || out[len(out)-1] == "" {
				continue
			}
			out = append(out, "")
			continue
		}
		out = append(out, l[indent:])
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	sheet := strings.Join(out, "\n")
	if utf8.RuneCountInString(sheet) > maxSongLinkContent {
		sheet = string([]rune(sheet)[:maxSongLinkContent])
		if i := strings.LastIndexByte(sheet, '\n'); i > 0 {
			sheet = sheet[:i]
		}
	}
	return sheet
}

// ── Text rendering ───────────────────────────────────────────────────────────

// Article containers, most specific first. Only these are checked for list pages; <body> is too noisy.
var articleClasses = []string{"post-body", "entry-content", "post-content", "article-content", "single-content"}

func articleRoot(doc *html.Node) (*html.Node, bool) {
	if n := findFirst(doc, func(n *html.Node) bool { return attr(n, "itemprop") == "articleBody" }); n != nil {
		return n, true
	}
	for _, cls := range articleClasses {
		if n := findFirst(doc, func(n *html.Node) bool { return hasClass(n, cls) }); n != nil {
			return n, true
		}
	}
	for _, a := range []atom.Atom{atom.Article, atom.Main} {
		if n := findFirst(doc, func(n *html.Node) bool { return n.DataAtom == a }); n != nil {
			return n, true
		}
	}
	if n := findFirst(doc, func(n *html.Node) bool { return n.DataAtom == atom.Body }); n != nil {
		return n, false
	}
	return doc, false
}

var skipAtoms = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true, atom.Svg: true,
	atom.Iframe: true, atom.Form: true, atom.Button: true, atom.Select: true, atom.Textarea: true,
	atom.Nav: true, atom.Aside: true, atom.Footer: true, atom.Header: true,
}

var blockAtoms = map[atom.Atom]bool{
	atom.Address: true, atom.Article: true, atom.Blockquote: true, atom.Dd: true, atom.Div: true,
	atom.Dl: true, atom.Dt: true, atom.Figure: true, atom.H1: true, atom.H2: true, atom.H3: true,
	atom.H4: true, atom.H5: true, atom.H6: true, atom.Hr: true, atom.Li: true, atom.Main: true,
	atom.Ol: true, atom.P: true, atom.Pre: true, atom.Section: true, atom.Table: true, atom.Tr: true,
	atom.Ul: true,
}

// renderLines lays out text the way a browser shows it: runs of ordinary whitespace collapse to one space and
// vanish at line starts, while &nbsp; is kept — Word-pasted chord blogs align chords with runs of &nbsp;.
func renderLines(root *html.Node) []string {
	var (
		lines   []string
		cur     strings.Builder
		pending bool
	)
	flush := func() {
		lines = append(lines, cur.String())
		cur.Reset()
		pending = false
	}
	breakBlock := func() {
		if cur.Len() > 0 {
			flush()
		}
		pending = false
	}
	var walk func(n *html.Node, inPre bool)
	walk = func(n *html.Node, inPre bool) {
		switch n.Type {
		case html.TextNode:
			for _, r := range n.Data {
				switch {
				case inPre && r == '\n':
					flush()
				case inPre:
					cur.WriteRune(r)
				case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f':
					pending = true
				default:
					if pending && cur.Len() > 0 {
						cur.WriteByte(' ')
					}
					pending = false
					if r == ' ' {
						r = ' '
					}
					cur.WriteRune(r)
				}
			}
			return
		case html.ElementNode:
			if skipAtoms[n.DataAtom] || hasClass(n, "d-none") {
				return
			}
			if n.DataAtom == atom.Br {
				flush()
				return
			}
		}
		block := n.Type == html.ElementNode && blockAtoms[n.DataAtom]
		if block {
			breakBlock()
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inPre || n.DataAtom == atom.Pre)
		}
		if block {
			breakBlock()
		}
	}
	walk(root, false)
	if cur.Len() > 0 {
		flush()
	}
	return lines
}

// ── DOM helpers ──────────────────────────────────────────────────────────────

func findFirst(n *html.Node, match func(*html.Node) bool) *html.Node {
	if n.Type == html.ElementNode && match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, match); found != nil {
			return found
		}
	}
	return nil
}

func findAll(n *html.Node, match func(*html.Node) bool) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && match(n) {
			out = append(out, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, cls string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == cls {
			return true
		}
	}
	return false
}

func isHeading(n *html.Node) bool {
	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4:
		return true
	}
	return false
}

// rawText concatenates text exactly as written (whitespace kept), which is what <pre> and psalmnote rows need.
// skipAtoms applies to descendants only: the root itself may be a <button> (psalmnote's part labels).
func rawText(root *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			b.WriteString(n.Data)
		case n.Type == html.ElementNode && n.DataAtom == atom.Br:
			b.WriteByte('\n')
		case n.Type == html.ElementNode && skipAtoms[n.DataAtom]:
		default:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
	}
	if root.Type == html.TextNode {
		walk(root)
	} else {
		for c := root.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	return strings.ReplaceAll(b.String(), " ", " ")
}

func collapsedText(n *html.Node) string {
	return strings.Join(strings.Fields(rawText(n)), " ")
}

func sourceName(u *url.URL) string {
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

func stripFragment(u *url.URL) string {
	c := *u
	c.Fragment = ""
	c.RawFragment = ""
	return c.String()
}
