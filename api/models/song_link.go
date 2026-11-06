package models

// Song link check statuses returned by POST /song-requests/check-link.
const (
	SongLinkSupported   = "supported"   // one song with chords was found; Content holds it
	SongLinkCollection  = "collection"  // the page lists several songs; Songs holds the links to pick from
	SongLinkUnsupported = "unsupported" // the page loaded but no chord sheet could be recognised
	SongLinkUnreachable = "unreachable" // the page could not be fetched
)

// SongLinkCheck is what a song request learns from a reference link before it is submitted.
type SongLinkCheck struct {
	Status     string         `json:"status"`
	URL        string         `json:"url"`
	Source     string         `json:"source"`
	Title      string         `json:"title,omitempty"`
	Artist     string         `json:"artist,omitempty"`
	Key        string         `json:"key,omitempty"`
	LyricsType string         `json:"lyrics_type,omitempty"`
	Content    string         `json:"content,omitempty"`
	Songs      []SongLinkItem `json:"songs,omitempty"`
	Reason     string         `json:"reason,omitempty"`
}

// SongLinkItem is one song listed on a collection page.
type SongLinkItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
