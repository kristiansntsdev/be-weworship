// Package mocks provides function-field mock structs for all repository and
// platform interfaces. Every method delegates to a stored Fn field; if the
// field is nil the method returns zero values so tests only need to wire up the
// methods they actually care about.
//
// To switch to generated mocks (go.uber.org/mock), install mockgen and run:
//
//	go generate ./api/repositories/...
//	go generate ./api/platform/...
//	go generate ./api/providers/...
package mocks

import (
	"context"
	"database/sql"

	"be-songbanks-v1/api/models"
	"be-songbanks-v1/api/platform"
	"be-songbanks-v1/api/repositories"
)

// ── AuthRepo ──────────────────────────────────────────────────────────────────

type AuthRepo struct {
	FindByEmailFn            func(string) (*models.User, error)
	FindByIDFn               func(int) (*models.UserBasic, error)
	FindManyByIDsFn          func([]int) ([]models.UserBasic, error)
	FindOrCreateGoogleUserFn func(string, string, string) (*models.User, error)
	CreateLocalFn            func(string, string, string) (*models.User, error)
}

func (m *AuthRepo) FindByEmail(email string) (*models.User, error) {
	if m.FindByEmailFn != nil {
		return m.FindByEmailFn(email)
	}
	return nil, nil
}
func (m *AuthRepo) FindByID(id int) (*models.UserBasic, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(id)
	}
	return nil, nil
}
func (m *AuthRepo) FindManyByIDs(ids []int) ([]models.UserBasic, error) {
	if m.FindManyByIDsFn != nil {
		return m.FindManyByIDsFn(ids)
	}
	return nil, nil
}
func (m *AuthRepo) FindOrCreateGoogleUser(email, name, providerID string) (*models.User, error) {
	if m.FindOrCreateGoogleUserFn != nil {
		return m.FindOrCreateGoogleUserFn(email, name, providerID)
	}
	return nil, nil
}
func (m *AuthRepo) CreateLocal(name, email, hashedPassword string) (*models.User, error) {
	if m.CreateLocalFn != nil {
		return m.CreateLocalFn(name, email, hashedPassword)
	}
	return nil, nil
}

// ── SongRepo ──────────────────────────────────────────────────────────────────

type SongRepo struct {
	ListFn                    func(int, int, string, string, string, string, []int, *bool, *bool) ([]models.Song, int, error)
	GetByIDFn                 func(int) (*models.Song, error)
	GetBySlugFn               func(string) (*models.Song, error)
	CreateFn                  func(string, string, *string, *int, *string, *string, bool, *string, string) (int, error)
	UpdateFieldsFn            func(int, string, ...any) (int64, error)
	DeleteByIDFn              func(int) (int64, error)
	ClearSongTagsFn           func(int) error
	AssignSongTagFn           func(int, int) error
	ExistsByIDFn              func(int) (bool, error)
	ListArtistsRawFn          func() ([]string, error)
	CountFn                   func() (int, error)
	ListAllChordProFn         func() ([]models.Song, error)
	CreateSongRequestFn       func(int, string, string, string, string) (*models.SongRequest, error)
	ListSongRequestsFn        func(string, int, int) ([]models.SongRequest, int, error)
	UpdateSongRequestStatusFn func(int, string, string) error
	ClearSongRequestLyricsFn  func(int) error
	GetSongRequestByIDFn      func(int) (*models.SongRequest, bool, error)
	ListUserSongRequestsFn    func(int, int, int) ([]models.SongRequest, int, error)
	DeleteSongRequestFn       func(int, int) (bool, error)
	CreateSongReportFn        func(int, int, string, string, string) (*models.SongReport, error)
	ListSongReportsFn         func(string, int, int) ([]models.SongReport, int, error)
	UpdateSongReportStatusFn  func(int, string, string) error
	GetSongReportByIDFn       func(int) (*models.SongReport, bool, error)
	ListUserSongReportsFn     func(int, int, int) ([]models.SongReport, int, error)
	DeleteSongReportFn        func(int, int) (bool, error)
}

func (m *SongRepo) List(page, limit int, search, baseChord, sortBy, sortOrder string, tagIDs []int, hasLink, chordPro *bool) ([]models.Song, int, error) {
	if m.ListFn != nil {
		return m.ListFn(page, limit, search, baseChord, sortBy, sortOrder, tagIDs, hasLink, chordPro)
	}
	return nil, 0, nil
}
func (m *SongRepo) GetByID(id int) (*models.Song, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}
	return nil, nil
}
func (m *SongRepo) GetBySlug(slug string) (*models.Song, error) {
	if m.GetBySlugFn != nil {
		return m.GetBySlugFn(slug)
	}
	return nil, nil
}
func (m *SongRepo) Create(title, artistJSON string, baseChord *string, bpm *int, lyrics *string, externalLinks *string, dmcaTakedown bool, dmcaStatusNotes *string, baseSlug string) (int, error) {
	if m.CreateFn != nil {
		return m.CreateFn(title, artistJSON, baseChord, bpm, lyrics, externalLinks, dmcaTakedown, dmcaStatusNotes, baseSlug)
	}
	return 0, nil
}
func (m *SongRepo) UpdateFields(id int, setExpr string, args ...any) (int64, error) {
	if m.UpdateFieldsFn != nil {
		return m.UpdateFieldsFn(id, setExpr, args...)
	}
	return 0, nil
}
func (m *SongRepo) DeleteByID(id int) (int64, error) {
	if m.DeleteByIDFn != nil {
		return m.DeleteByIDFn(id)
	}
	return 0, nil
}
func (m *SongRepo) ClearSongTags(songID int) error {
	if m.ClearSongTagsFn != nil {
		return m.ClearSongTagsFn(songID)
	}
	return nil
}
func (m *SongRepo) AssignSongTag(songID, tagID int) error {
	if m.AssignSongTagFn != nil {
		return m.AssignSongTagFn(songID, tagID)
	}
	return nil
}
func (m *SongRepo) ExistsByID(id int) (bool, error) {
	if m.ExistsByIDFn != nil {
		return m.ExistsByIDFn(id)
	}
	return false, nil
}
func (m *SongRepo) ListArtistsRaw() ([]string, error) {
	if m.ListArtistsRawFn != nil {
		return m.ListArtistsRawFn()
	}
	return nil, nil
}
func (m *SongRepo) Count() (int, error) {
	if m.CountFn != nil {
		return m.CountFn()
	}
	return 0, nil
}
func (m *SongRepo) ListAllChordPro() ([]models.Song, error) {
	if m.ListAllChordProFn != nil {
		return m.ListAllChordProFn()
	}
	return nil, nil
}
func (m *SongRepo) CreateSongRequest(userID int, songTitle, referenceLink, lyricsType, lyrics string) (*models.SongRequest, error) {
	if m.CreateSongRequestFn != nil {
		return m.CreateSongRequestFn(userID, songTitle, referenceLink, lyricsType, lyrics)
	}
	return nil, nil
}
func (m *SongRepo) ListSongRequests(status string, page, limit int) ([]models.SongRequest, int, error) {
	if m.ListSongRequestsFn != nil {
		return m.ListSongRequestsFn(status, page, limit)
	}
	return nil, 0, nil
}
func (m *SongRepo) UpdateSongRequestStatus(id int, status, adminNotes string) error {
	if m.UpdateSongRequestStatusFn != nil {
		return m.UpdateSongRequestStatusFn(id, status, adminNotes)
	}
	return nil
}
func (m *SongRepo) ClearSongRequestLyrics(id int) error {
	if m.ClearSongRequestLyricsFn != nil {
		return m.ClearSongRequestLyricsFn(id)
	}
	return nil
}
func (m *SongRepo) GetSongRequestByID(id int) (*models.SongRequest, bool, error) {
	if m.GetSongRequestByIDFn != nil {
		return m.GetSongRequestByIDFn(id)
	}
	return nil, false, nil
}
func (m *SongRepo) ListUserSongRequests(userID, page, limit int) ([]models.SongRequest, int, error) {
	if m.ListUserSongRequestsFn != nil {
		return m.ListUserSongRequestsFn(userID, page, limit)
	}
	return nil, 0, nil
}
func (m *SongRepo) DeleteSongRequest(id, userID int) (bool, error) {
	if m.DeleteSongRequestFn != nil {
		return m.DeleteSongRequestFn(id, userID)
	}
	return false, nil
}
func (m *SongRepo) CreateSongReport(userID, songID int, reportType, description, evidenceURL string) (*models.SongReport, error) {
	if m.CreateSongReportFn != nil {
		return m.CreateSongReportFn(userID, songID, reportType, description, evidenceURL)
	}
	return nil, nil
}
func (m *SongRepo) ListSongReports(status string, page, limit int) ([]models.SongReport, int, error) {
	if m.ListSongReportsFn != nil {
		return m.ListSongReportsFn(status, page, limit)
	}
	return nil, 0, nil
}
func (m *SongRepo) UpdateSongReportStatus(id int, status, adminNotes string) error {
	if m.UpdateSongReportStatusFn != nil {
		return m.UpdateSongReportStatusFn(id, status, adminNotes)
	}
	return nil
}
func (m *SongRepo) GetSongReportByID(id int) (*models.SongReport, bool, error) {
	if m.GetSongReportByIDFn != nil {
		return m.GetSongReportByIDFn(id)
	}
	return nil, false, nil
}
func (m *SongRepo) ListUserSongReports(userID, page, limit int) ([]models.SongReport, int, error) {
	if m.ListUserSongReportsFn != nil {
		return m.ListUserSongReportsFn(userID, page, limit)
	}
	return nil, 0, nil
}
func (m *SongRepo) DeleteSongReport(id, userID int) (bool, error) {
	if m.DeleteSongReportFn != nil {
		return m.DeleteSongReportFn(id, userID)
	}
	return false, nil
}

// ── PlaylistRepo ──────────────────────────────────────────────────────────────

type PlaylistRepo struct {
	RecordShareEventFn         func(int)
	CountShareEventsThisWeekFn func() (int, error)
	CountShareableFn           func() (int, error)
	NameExistsForUserFn        func(int, string) (bool, error)
	CreateFn                   func(int, string, []int) (int, error)
	CountAccessibleFn          func(int) (int, error)
	ListAccessibleFn           func(int, int, int) ([]models.PlaylistListRow, error)
	GetByIDFn                  func(int) (*models.Playlist, error)
	UpdateNameFn               func(int, int, string) (int64, error)
	DeleteFn                   func(int, int) (int64, error)
	UpdateShareFn              func(int, string, string, int64) error
	FindByShareTokenFn         func(string) (int, int, sql.NullInt64, error)
	GetPreviewByShareTokenFn   func(string) (*repositories.PlaylistPreview, error)
	SetTeamIDFn                func(int, int64) error
	ClearShareAndTeamFn        func(int) error
	CanManageFn                func(int, int) (bool, error)
	SetSongsFn                 func(int, []int) error
	ExistsAndOwnerFn           func(int) (int, error)
	SetSongKeyFn               func(int, int, string) error
	GetSongKeysFn              func(int) (map[int]string, error)
	DeleteSongKeyFn            func(int, int) error
}

func (m *PlaylistRepo) RecordShareEvent(playlistID int) {
	if m.RecordShareEventFn != nil {
		m.RecordShareEventFn(playlistID)
	}
}
func (m *PlaylistRepo) CountShareEventsThisWeek() (int, error) {
	if m.CountShareEventsThisWeekFn != nil {
		return m.CountShareEventsThisWeekFn()
	}
	return 0, nil
}
func (m *PlaylistRepo) CountShareable() (int, error) {
	if m.CountShareableFn != nil {
		return m.CountShareableFn()
	}
	return 0, nil
}
func (m *PlaylistRepo) NameExistsForUser(userID int, name string) (bool, error) {
	if m.NameExistsForUserFn != nil {
		return m.NameExistsForUserFn(userID, name)
	}
	return false, nil
}
func (m *PlaylistRepo) Create(userID int, name string, songs []int) (int, error) {
	if m.CreateFn != nil {
		return m.CreateFn(userID, name, songs)
	}
	return 0, nil
}
func (m *PlaylistRepo) CountAccessible(userID int) (int, error) {
	if m.CountAccessibleFn != nil {
		return m.CountAccessibleFn(userID)
	}
	return 0, nil
}
func (m *PlaylistRepo) ListAccessible(userID, page, limit int) ([]models.PlaylistListRow, error) {
	if m.ListAccessibleFn != nil {
		return m.ListAccessibleFn(userID, page, limit)
	}
	return nil, nil
}
func (m *PlaylistRepo) GetByID(playlistID int) (*models.Playlist, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(playlistID)
	}
	return nil, nil
}
func (m *PlaylistRepo) UpdateName(playlistID, userID int, name string) (int64, error) {
	if m.UpdateNameFn != nil {
		return m.UpdateNameFn(playlistID, userID, name)
	}
	return 0, nil
}
func (m *PlaylistRepo) Delete(playlistID, userID int) (int64, error) {
	if m.DeleteFn != nil {
		return m.DeleteFn(playlistID, userID)
	}
	return 0, nil
}
func (m *PlaylistRepo) UpdateShare(playlistID int, link, token string, teamID int64) error {
	if m.UpdateShareFn != nil {
		return m.UpdateShareFn(playlistID, link, token, teamID)
	}
	return nil
}
func (m *PlaylistRepo) FindByShareToken(shareToken string) (int, int, sql.NullInt64, error) {
	if m.FindByShareTokenFn != nil {
		return m.FindByShareTokenFn(shareToken)
	}
	return 0, 0, sql.NullInt64{}, nil
}
func (m *PlaylistRepo) GetPreviewByShareToken(shareToken string) (*repositories.PlaylistPreview, error) {
	if m.GetPreviewByShareTokenFn != nil {
		return m.GetPreviewByShareTokenFn(shareToken)
	}
	return nil, nil
}
func (m *PlaylistRepo) SetTeamID(playlistID int, teamID int64) error {
	if m.SetTeamIDFn != nil {
		return m.SetTeamIDFn(playlistID, teamID)
	}
	return nil
}
func (m *PlaylistRepo) ClearShareAndTeam(playlistID int) error {
	if m.ClearShareAndTeamFn != nil {
		return m.ClearShareAndTeamFn(playlistID)
	}
	return nil
}
func (m *PlaylistRepo) CanManage(playlistID, userID int) (bool, error) {
	if m.CanManageFn != nil {
		return m.CanManageFn(playlistID, userID)
	}
	return false, nil
}
func (m *PlaylistRepo) SetSongs(playlistID int, songs []int) error {
	if m.SetSongsFn != nil {
		return m.SetSongsFn(playlistID, songs)
	}
	return nil
}
func (m *PlaylistRepo) ExistsAndOwner(playlistID int) (int, error) {
	if m.ExistsAndOwnerFn != nil {
		return m.ExistsAndOwnerFn(playlistID)
	}
	return 0, nil
}
func (m *PlaylistRepo) SetSongKey(playlistID, songID int, baseChord string) error {
	if m.SetSongKeyFn != nil {
		return m.SetSongKeyFn(playlistID, songID, baseChord)
	}
	return nil
}
func (m *PlaylistRepo) GetSongKeys(playlistID int) (map[int]string, error) {
	if m.GetSongKeysFn != nil {
		return m.GetSongKeysFn(playlistID)
	}
	return nil, nil
}
func (m *PlaylistRepo) DeleteSongKey(playlistID, songID int) error {
	if m.DeleteSongKeyFn != nil {
		return m.DeleteSongKeyFn(playlistID, songID)
	}
	return nil
}

// ── TeamRepo ──────────────────────────────────────────────────────────────────

type TeamRepo struct {
	CreateFn           func(int, int, []int) (int64, error)
	GetByIDFn          func(int) (*models.PlaylistTeam, error)
	FindByPlaylistIDFn func(int) (*models.PlaylistTeam, error)
	ListByLeadIDFn     func(int) ([]models.PlaylistTeam, error)
	UpdateMembersFn    func(int, []int) error
	UpdateCoLeadsFn    func(int, []int) error
	DeleteFn           func(int) error
	ListMembersFn      func(int) ([]models.PlaylistMemberDetail, error)
	GetMemberFn        func(int, int) (*models.PlaylistMember, error)
	CountByRoleFn      func(int, string) (int, error)
	UpsertMemberFn     func(int, int, string) error
}

func (m *TeamRepo) Create(playlistID, leadID int, members []int) (int64, error) {
	if m.CreateFn != nil {
		return m.CreateFn(playlistID, leadID, members)
	}
	return 0, nil
}
func (m *TeamRepo) GetByID(id int) (*models.PlaylistTeam, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(id)
	}
	return nil, nil
}
func (m *TeamRepo) FindByPlaylistID(playlistID int) (*models.PlaylistTeam, error) {
	if m.FindByPlaylistIDFn != nil {
		return m.FindByPlaylistIDFn(playlistID)
	}
	return nil, nil
}
func (m *TeamRepo) ListByLeadID(leadID int) ([]models.PlaylistTeam, error) {
	if m.ListByLeadIDFn != nil {
		return m.ListByLeadIDFn(leadID)
	}
	return nil, nil
}
func (m *TeamRepo) UpdateMembers(id int, members []int) error {
	if m.UpdateMembersFn != nil {
		return m.UpdateMembersFn(id, members)
	}
	return nil
}
func (m *TeamRepo) UpdateCoLeads(id int, coLeads []int) error {
	if m.UpdateCoLeadsFn != nil {
		return m.UpdateCoLeadsFn(id, coLeads)
	}
	return nil
}
func (m *TeamRepo) Delete(id int) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(id)
	}
	return nil
}
func (m *TeamRepo) ListMembers(playlistID int) ([]models.PlaylistMemberDetail, error) {
	if m.ListMembersFn != nil {
		return m.ListMembersFn(playlistID)
	}
	return nil, nil
}
func (m *TeamRepo) GetMember(playlistID, userID int) (*models.PlaylistMember, error) {
	if m.GetMemberFn != nil {
		return m.GetMemberFn(playlistID, userID)
	}
	return nil, nil
}
func (m *TeamRepo) CountByRole(playlistID int, role string) (int, error) {
	if m.CountByRoleFn != nil {
		return m.CountByRoleFn(playlistID, role)
	}
	return 0, nil
}
func (m *TeamRepo) UpsertMember(playlistID, userID int, role string) error {
	if m.UpsertMemberFn != nil {
		return m.UpsertMemberFn(playlistID, userID, role)
	}
	return nil
}

// ── TagRepo ───────────────────────────────────────────────────────────────────

type TagRepo struct {
	ListFn           func(string) ([]models.Tag, error)
	FindByNameFn     func(string) (*models.Tag, error)
	CreateFn         func(string) (int, error)
	GetTagsForSongFn func(int) ([]models.Tag, error)
}

func (m *TagRepo) List(search string) ([]models.Tag, error) {
	if m.ListFn != nil {
		return m.ListFn(search)
	}
	return nil, nil
}
func (m *TagRepo) FindByName(name string) (*models.Tag, error) {
	if m.FindByNameFn != nil {
		return m.FindByNameFn(name)
	}
	return nil, nil
}
func (m *TagRepo) Create(name string) (int, error) {
	if m.CreateFn != nil {
		return m.CreateFn(name)
	}
	return 0, nil
}
func (m *TagRepo) GetTagsForSong(songID int) ([]models.Tag, error) {
	if m.GetTagsForSongFn != nil {
		return m.GetTagsForSongFn(songID)
	}
	return nil, nil
}

// ── UserRepo ──────────────────────────────────────────────────────────────────

type UserRepo struct {
	CountFn                 func(string) (int, error)
	ListFn                  func(string, int, int) ([]models.User, error)
	GetDetailFn             func(int) (*models.UserDetail, error)
	UpsertDetailFn          func(int, *string, *string, *string, *string) error
	FindByIDFn              func(int) (*models.User, error)
	UpdateRoleFn            func(int, string) (*models.User, error)
	UpdateAvatarURLFn       func(int, string) error
	FindByEmailFn           func(string) (*models.User, error)
	CreateDeletionRequestFn func(string, *int, string) error
}

func (m *UserRepo) Count(search string) (int, error) {
	if m.CountFn != nil {
		return m.CountFn(search)
	}
	return 0, nil
}
func (m *UserRepo) List(search string, page, limit int) ([]models.User, error) {
	if m.ListFn != nil {
		return m.ListFn(search, page, limit)
	}
	return nil, nil
}
func (m *UserRepo) GetDetail(userID int) (*models.UserDetail, error) {
	if m.GetDetailFn != nil {
		return m.GetDetailFn(userID)
	}
	return nil, nil
}
func (m *UserRepo) UpsertDetail(userID int, fullName, province, city, postalCode *string) error {
	if m.UpsertDetailFn != nil {
		return m.UpsertDetailFn(userID, fullName, province, city, postalCode)
	}
	return nil
}
func (m *UserRepo) FindByID(userID int) (*models.User, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(userID)
	}
	return nil, nil
}
func (m *UserRepo) UpdateRole(userID int, role string) (*models.User, error) {
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(userID, role)
	}
	return nil, nil
}
func (m *UserRepo) UpdateAvatarURL(userID int, avatarURL string) error {
	if m.UpdateAvatarURLFn != nil {
		return m.UpdateAvatarURLFn(userID, avatarURL)
	}
	return nil
}
func (m *UserRepo) FindByEmail(email string) (*models.User, error) {
	if m.FindByEmailFn != nil {
		return m.FindByEmailFn(email)
	}
	return nil, nil
}
func (m *UserRepo) CreateDeletionRequest(email string, userID *int, ipAddress string) error {
	if m.CreateDeletionRequestFn != nil {
		return m.CreateDeletionRequestFn(email, userID, ipAddress)
	}
	return nil
}

// ── NotificationRepo ──────────────────────────────────────────────────────────

type NotificationRepo struct {
	UpsertDeviceTokenFn         func(int, string, string) error
	DeleteDeviceTokenFn         func(int, string) error
	GetTokensByUserIDsFn        func([]int) ([]string, error)
	GetAllTokensFn              func() ([]string, error)
	SaveNotificationFn          func(int, string, string, string, string) error
	SaveBroadcastNotificationFn func(string, string, string, string) error
	ListByUserIDFn              func(int, int, int) ([]repositories.NotificationRow, error)
	MarkReadFn                  func(int, int) error
	CountUnreadFn               func(int) (int, error)
}

func (m *NotificationRepo) UpsertDeviceToken(userID int, token, platform string) error {
	if m.UpsertDeviceTokenFn != nil {
		return m.UpsertDeviceTokenFn(userID, token, platform)
	}
	return nil
}
func (m *NotificationRepo) DeleteDeviceToken(userID int, token string) error {
	if m.DeleteDeviceTokenFn != nil {
		return m.DeleteDeviceTokenFn(userID, token)
	}
	return nil
}
func (m *NotificationRepo) GetTokensByUserIDs(userIDs []int) ([]string, error) {
	if m.GetTokensByUserIDsFn != nil {
		return m.GetTokensByUserIDsFn(userIDs)
	}
	return nil, nil
}
func (m *NotificationRepo) GetAllTokens() ([]string, error) {
	if m.GetAllTokensFn != nil {
		return m.GetAllTokensFn()
	}
	return nil, nil
}
func (m *NotificationRepo) SaveNotification(userID int, title, message, groupType, data string) error {
	if m.SaveNotificationFn != nil {
		return m.SaveNotificationFn(userID, title, message, groupType, data)
	}
	return nil
}
func (m *NotificationRepo) SaveBroadcastNotification(title, message, groupType, data string) error {
	if m.SaveBroadcastNotificationFn != nil {
		return m.SaveBroadcastNotificationFn(title, message, groupType, data)
	}
	return nil
}
func (m *NotificationRepo) ListByUserID(userID, page, limit int) ([]repositories.NotificationRow, error) {
	if m.ListByUserIDFn != nil {
		return m.ListByUserIDFn(userID, page, limit)
	}
	return nil, nil
}
func (m *NotificationRepo) MarkRead(id, userID int) error {
	if m.MarkReadFn != nil {
		return m.MarkReadFn(id, userID)
	}
	return nil
}
func (m *NotificationRepo) CountUnread(userID int) (int, error) {
	if m.CountUnreadFn != nil {
		return m.CountUnreadFn(userID)
	}
	return 0, nil
}

// ── AnalyticsRepo ─────────────────────────────────────────────────────────────

type AnalyticsRepo struct {
	RecordSongEventFn    func(*int, *int, string, string, *int)
	RecordSearchLogFn    func(*int, string, *string, int, string)
	RecordSessionFn      func(*int, string, string, string) error
	RecordPerformanceFn  func(*int, string, string, *string, *string, *int, *int, string, string) error
	TopSongsFn           func(int, int) ([]repositories.TopSongRow, error)
	NewUsersPerDayFn     func(int) ([]repositories.DailyCountRow, error)
	DAUFn                func(int) ([]repositories.DailyCountRow, error)
	TopSearchesFn        func(int, int) ([]repositories.TopSearchRow, error)
	SessionsByPlatformFn func(int) ([]repositories.PlatformBreakdownRow, error)
	PerformanceSummaryFn func(int) ([]repositories.PerfSummaryRow, error)
	TotalUsersFn         func() (int, error)
	DAUTodayFn           func() (int, error)
	MAUFn                func() (int, error)
}

func (m *AnalyticsRepo) RecordSongEvent(songID, userID *int, eventType, platform string, durationMs *int) {
	if m.RecordSongEventFn != nil {
		m.RecordSongEventFn(songID, userID, eventType, platform, durationMs)
	}
}
func (m *AnalyticsRepo) RecordSearchLog(userID *int, query string, filtersJSON *string, resultsCount int, platform string) {
	if m.RecordSearchLogFn != nil {
		m.RecordSearchLogFn(userID, query, filtersJSON, resultsCount, platform)
	}
}
func (m *AnalyticsRepo) RecordSession(userID *int, platform, appVersion, deviceOS string) error {
	if m.RecordSessionFn != nil {
		return m.RecordSessionFn(userID, platform, appVersion, deviceOS)
	}
	return nil
}
func (m *AnalyticsRepo) RecordPerformance(userID *int, platform, metricType string, endpoint, screenName *string, durationMs, statusCode *int, appVersion, deviceOS string) error {
	if m.RecordPerformanceFn != nil {
		return m.RecordPerformanceFn(userID, platform, metricType, endpoint, screenName, durationMs, statusCode, appVersion, deviceOS)
	}
	return nil
}
func (m *AnalyticsRepo) TopSongs(days, limit int) ([]repositories.TopSongRow, error) {
	if m.TopSongsFn != nil {
		return m.TopSongsFn(days, limit)
	}
	return nil, nil
}
func (m *AnalyticsRepo) NewUsersPerDay(days int) ([]repositories.DailyCountRow, error) {
	if m.NewUsersPerDayFn != nil {
		return m.NewUsersPerDayFn(days)
	}
	return nil, nil
}
func (m *AnalyticsRepo) DAU(days int) ([]repositories.DailyCountRow, error) {
	if m.DAUFn != nil {
		return m.DAUFn(days)
	}
	return nil, nil
}
func (m *AnalyticsRepo) TopSearches(days, limit int) ([]repositories.TopSearchRow, error) {
	if m.TopSearchesFn != nil {
		return m.TopSearchesFn(days, limit)
	}
	return nil, nil
}
func (m *AnalyticsRepo) SessionsByPlatform(days int) ([]repositories.PlatformBreakdownRow, error) {
	if m.SessionsByPlatformFn != nil {
		return m.SessionsByPlatformFn(days)
	}
	return nil, nil
}
func (m *AnalyticsRepo) PerformanceSummary(days int) ([]repositories.PerfSummaryRow, error) {
	if m.PerformanceSummaryFn != nil {
		return m.PerformanceSummaryFn(days)
	}
	return nil, nil
}
func (m *AnalyticsRepo) TotalUsers() (int, error) {
	if m.TotalUsersFn != nil {
		return m.TotalUsersFn()
	}
	return 0, nil
}
func (m *AnalyticsRepo) DAUToday() (int, error) {
	if m.DAUTodayFn != nil {
		return m.DAUTodayFn()
	}
	return 0, nil
}
func (m *AnalyticsRepo) MAU() (int, error) {
	if m.MAUFn != nil {
		return m.MAUFn()
	}
	return 0, nil
}

// ── AuditRepo ─────────────────────────────────────────────────────────────────

type AuditRepo struct {
	LogFn  func(*int, string, string, string, string, *int, string, any)
	ListFn func(string, string, *int, int, int) ([]repositories.AuditLogRow, int, error)
}

func (m *AuditRepo) Log(userID *int, userName, userEmail, action, entityType string, entityID *int, entityName string, changes any) {
	if m.LogFn != nil {
		m.LogFn(userID, userName, userEmail, action, entityType, entityID, entityName, changes)
	}
}
func (m *AuditRepo) List(action, entityType string, userID *int, page, limit int) ([]repositories.AuditLogRow, int, error) {
	if m.ListFn != nil {
		return m.ListFn(action, entityType, userID, page, limit)
	}
	return nil, 0, nil
}

// ── SongCache ─────────────────────────────────────────────────────────────────

type SongCache struct {
	EnabledFn             func() bool
	GetFn                 func(string, any) bool
	SetFn                 func(string, any)
	InvalidateSongsListFn func()
	InvalidateArtistsFn   func()
}

func (m *SongCache) Enabled() bool {
	if m.EnabledFn != nil {
		return m.EnabledFn()
	}
	return false
}
func (m *SongCache) Get(key string, out any) bool {
	if m.GetFn != nil {
		return m.GetFn(key, out)
	}
	return false
}
func (m *SongCache) Set(key string, value any) {
	if m.SetFn != nil {
		m.SetFn(key, value)
	}
}
func (m *SongCache) InvalidateSongsList() {
	if m.InvalidateSongsListFn != nil {
		m.InvalidateSongsListFn()
	}
}
func (m *SongCache) InvalidateArtists() {
	if m.InvalidateArtistsFn != nil {
		m.InvalidateArtistsFn()
	}
}

// ── LiveCache ─────────────────────────────────────────────────────────────────

type LiveCache struct {
	EnabledFn      func() bool
	StartSessionFn func(int, int) error
	EndSessionFn   func(int) error
	UpdateStateFn  func(int, int, float64) error
	TouchContentFn func(int) error
	GetStateFn     func(int) (*platform.LiveState, error)
}

func (m *LiveCache) Enabled() bool {
	if m.EnabledFn != nil {
		return m.EnabledFn()
	}
	return false
}
func (m *LiveCache) StartSession(playlistID, leaderUserID int) error {
	if m.StartSessionFn != nil {
		return m.StartSessionFn(playlistID, leaderUserID)
	}
	return nil
}
func (m *LiveCache) EndSession(playlistID int) error {
	if m.EndSessionFn != nil {
		return m.EndSessionFn(playlistID)
	}
	return nil
}
func (m *LiveCache) UpdateState(playlistID, songIndex int, scrollRatio float64) error {
	if m.UpdateStateFn != nil {
		return m.UpdateStateFn(playlistID, songIndex, scrollRatio)
	}
	return nil
}
func (m *LiveCache) TouchContent(playlistID int) error {
	if m.TouchContentFn != nil {
		return m.TouchContentFn(playlistID)
	}
	return nil
}
func (m *LiveCache) GetState(playlistID int) (*platform.LiveState, error) {
	if m.GetStateFn != nil {
		return m.GetStateFn(playlistID)
	}
	return nil, nil
}

// ── PushProvider ──────────────────────────────────────────────────────────────

type PushProvider struct {
	SendFn func(context.Context, []string, string, string, map[string]string) error
}

func (m *PushProvider) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if m.SendFn != nil {
		return m.SendFn(ctx, tokens, title, body, data)
	}
	return nil
}
