package repositories

//go:generate mockgen -source=interfaces.go -destination=../services/mocks/mock_repos.go -package=mocks

import (
	"database/sql"

	"be-songbanks-v1/api/models"
)

// AuthRepoIface abstracts AuthRepository for testing.
type AuthRepoIface interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id int) (*models.UserBasic, error)
	FindManyByIDs(ids []int) ([]models.UserBasic, error)
	FindOrCreateGoogleUser(email, username, providerID string) (*models.User, error)
	CreateLocal(username, email, hashedPassword string) (*models.User, error)
}

// SongRepoIface abstracts SongRepository for testing.
type SongRepoIface interface {
	List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, hasLink, chordPro *bool) ([]models.Song, int, error)
	GetByID(id int) (*models.Song, error)
	GetBySlug(slug string) (*models.Song, error)
	Create(title, artistJSON string, baseChord *string, bpm *int, lyrics *string, externalLinks *string, dmcaTakedown bool, dmcaStatusNotes *string, baseSlug string) (int, error)
	UpdateFields(id int, setExpr string, args ...any) (int64, error)
	DeleteByID(id int) (int64, error)
	ClearSongTags(songID int) error
	AssignSongTag(songID, tagID int) error
	ExistsByID(id int) (bool, error)
	ListArtistsRaw() ([]string, error)
	Count() (int, error)
	ListAllChordPro() ([]models.Song, error)
	CreateSongRequest(userID int, songTitle, referenceLink, lyricsType, lyrics string) (*models.SongRequest, error)
	ListSongRequests(status string, page, limit int) ([]models.SongRequest, int, error)
	UpdateSongRequestStatus(id int, status, adminNotes string) error
	GetSongRequestByID(id int) (*models.SongRequest, bool, error)
	ListUserSongRequests(userID, page, limit int) ([]models.SongRequest, int, error)
	DeleteSongRequest(id, userID int) (bool, error)
}

// PlaylistRepoIface abstracts PlaylistRepository for testing.
type PlaylistRepoIface interface {
	RecordShareEvent(playlistID int)
	CountShareEventsThisWeek() (int, error)
	CountShareable() (int, error)
	NameExistsForUser(userID int, name string) (bool, error)
	Create(userID int, name string, songs []int) (int, error)
	CountAccessible(userID int) (int, error)
	ListAccessible(userID, page, limit int) ([]models.PlaylistListRow, error)
	GetByID(playlistID int) (*models.Playlist, error)
	UpdateName(playlistID, userID int, name string) (int64, error)
	Delete(playlistID, userID int) (int64, error)
	UpdateShare(playlistID int, link, token string, teamID int64) error
	FindByShareToken(shareToken string) (playlistID, ownerID int, teamID sql.NullInt64, err error)
	GetPreviewByShareToken(shareToken string) (*PlaylistPreview, error)
	SetTeamID(playlistID int, teamID int64) error
	ClearShareAndTeam(playlistID int) error
	CanManage(playlistID, userID int) (bool, error)
	SetSongs(playlistID int, songs []int) error
	ExistsAndOwner(playlistID int) (int, error)
	SetSongKey(playlistID, songID int, baseChord string) error
	GetSongKeys(playlistID int) (map[int]string, error)
	DeleteSongKey(playlistID, songID int) error
}

// TeamRepoIface abstracts TeamRepository for testing.
type TeamRepoIface interface {
	Create(playlistID, leadID int, members []int) (int64, error)
	GetByID(id int) (*models.PlaylistTeam, error)
	FindByPlaylistID(playlistID int) (*models.PlaylistTeam, error)
	ListByLeadID(leadID int) ([]models.PlaylistTeam, error)
	UpdateMembers(id int, members []int) error
	UpdateCoLeads(id int, coLeads []int) error
	Delete(id int) error
	ListMembers(playlistID int) ([]models.PlaylistMemberDetail, error)
	GetMember(playlistID, userID int) (*models.PlaylistMember, error)
	CountByRole(playlistID int, role string) (int, error)
	UpsertMember(playlistID, userID int, role string) error
}

// TagRepoIface abstracts TagRepository for testing.
type TagRepoIface interface {
	List(search string) ([]models.Tag, error)
	FindByName(name string) (*models.Tag, error)
	Create(name string) (int, error)
	GetTagsForSong(songID int) ([]models.Tag, error)
}

// UserRepoIface abstracts UserRepository for testing.
type UserRepoIface interface {
	Count(search string) (int, error)
	List(search string, page, limit int) ([]models.User, error)
	GetDetail(userID int) (*models.UserDetail, error)
	UpsertDetail(userID int, fullName, province, city, postalCode *string) error
	FindByID(userID int) (*models.User, error)
	UpdateAvatarURL(userID int, avatarURL string) error
	FindByEmail(email string) (*models.User, error)
	CreateDeletionRequest(email string, userID *int, ipAddress string) error
}

// NotificationRepoIface abstracts NotificationRepository for testing.
type NotificationRepoIface interface {
	UpsertDeviceToken(userID int, token, platform string) error
	DeleteDeviceToken(userID int, token string) error
	GetTokensByUserIDs(userIDs []int) ([]string, error)
	GetAllTokens() ([]string, error)
	SaveNotification(userID int, title, message, groupType, data string) error
	SaveBroadcastNotification(title, message, groupType, data string) error
	ListByUserID(userID, page, limit int) ([]NotificationRow, error)
	MarkRead(id, userID int) error
	CountUnread(userID int) (int, error)
}

// AnalyticsRepoIface abstracts AnalyticsRepository for testing.
type AnalyticsRepoIface interface {
	RecordSongEvent(songID, userID *int, eventType, platform string, durationMs *int)
	RecordSearchLog(userID *int, query string, filtersJSON *string, resultsCount int, platform string)
	RecordSession(userID *int, platform, appVersion, deviceOS string)
	RecordPerformance(userID *int, platform, metricType string, endpoint, screenName *string, durationMs, statusCode *int, appVersion, deviceOS string)
	TopSongs(days int, limit int) ([]TopSongRow, error)
	NewUsersPerDay(days int) ([]DailyCountRow, error)
	DAU(days int) ([]DailyCountRow, error)
	TopSearches(days, limit int) ([]TopSearchRow, error)
	SessionsByPlatform(days int) ([]PlatformBreakdownRow, error)
	PerformanceSummary(days int) ([]PerfSummaryRow, error)
	TotalUsers() (int, error)
	DAUToday() (int, error)
	MAU() (int, error)
}

// AuditRepoIface abstracts AuditRepository for testing.
type AuditRepoIface interface {
	Log(userID *int, userName, userEmail, action, entityType string, entityID *int, entityName string, changes any)
	List(action, entityType string, userID *int, page, limit int) ([]AuditLogRow, int, error)
}

// InvitationRepoIface abstracts InvitationRepository for testing.
type InvitationRepoIface interface {
	CreateInvitation(playlistID, inviterID, inviteeID int) (int, error)
	FindPendingByID(id int) (*models.PlaylistInvitation, error)
	FindPendingByPlaylistAndInvitee(playlistID, inviteeID int) (*models.PlaylistInvitation, error)
	UpdateStatus(id int, status string) error
}
