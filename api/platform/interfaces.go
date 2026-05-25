package platform

//go:generate mockgen -source=interfaces.go -destination=../services/mocks/mock_platform.go -package=mocks

// SongCacheIface abstracts SongCache for testing.
type SongCacheIface interface {
	Enabled() bool
	Get(key string, out any) bool
	Set(key string, value any)
	InvalidateSongsList()
	InvalidateArtists()
}

// LiveCacheIface abstracts LiveCache for testing.
type LiveCacheIface interface {
	Enabled() bool
	StartSession(playlistID, leaderUserID int) error
	EndSession(playlistID int) error
	UpdateState(playlistID, songIndex int, scrollRatio float64) error
	GetState(playlistID int) (*LiveState, error)
}
