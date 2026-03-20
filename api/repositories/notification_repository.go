package repositories

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// NotificationRepository handles device-token persistence.
// Tokens older than 1 week are automatically pruned on every upsert.
type NotificationRepository struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// UpsertDeviceToken inserts or refreshes a device token for the given user.
// It also deletes any tokens for this user that are older than 1 week (TTL).
func (r *NotificationRepository) UpsertDeviceToken(userID int, token, platform string) error {
	// Prune stale tokens (> 1 week) for this user first.
	_, _ = r.db.Exec(
		`DELETE FROM device_tokens WHERE user_id = $1 AND "updatedAt" < $2`,
		userID, time.Now().Add(-7*24*time.Hour),
	)

	_, err := r.db.Exec(`
		INSERT INTO device_tokens (user_id, token, platform, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, token)
		DO UPDATE SET platform = EXCLUDED.platform, "updatedAt" = NOW()
	`, userID, token, platform)
	return err
}

// DeleteDeviceToken removes a specific token (called on logout).
func (r *NotificationRepository) DeleteDeviceToken(userID int, token string) error {
	_, err := r.db.Exec(
		`DELETE FROM device_tokens WHERE user_id = $1 AND token = $2`,
		userID, token,
	)
	return err
}

// GetTokensByUserIDs returns all non-expired FCM tokens for a set of user IDs.
func (r *NotificationRepository) GetTokensByUserIDs(userIDs []int) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	// Build $1, $2, … placeholders
	query, args, err := sqlx.In(
		`SELECT token FROM device_tokens WHERE user_id IN (?) AND "updatedAt" >= ?`,
		userIDs, time.Now().Add(-7*24*time.Hour),
	)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var tokens []string
	if err := r.db.Select(&tokens, query, args...); err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetAllTokens returns all non-expired device tokens across all users (for broadcast sends).
func (r *NotificationRepository) GetAllTokens() ([]string, error) {
	var tokens []string
	err := r.db.Select(&tokens,
		`SELECT token FROM device_tokens WHERE "updatedAt" >= $1`,
		time.Now().Add(-7*24*time.Hour),
	)
	return tokens, err
}


// UserID is nil for broadcast notifications (visible to all users).
type NotificationRow struct {
	ID        int     `db:"id"`
	UserID    *int    `db:"user_id"`
	Title     string  `db:"title"`
	Message   string  `db:"message"`
	GroupType string  `db:"group_type"`
	IsRead    bool    `db:"is_read"`
	Data      string  `db:"data"`
	CreatedAt string  `db:"createdAt"`
}

// SaveNotification persists a targeted notification to the inbox for a specific user.
func (r *NotificationRepository) SaveNotification(userID int, title, message, groupType, data string) error {
	_, err := r.db.Exec(`
		INSERT INTO notifications (user_id, title, message, group_type, is_read, data, "createdAt")
		VALUES ($1, $2, $3, $4, FALSE, $5, NOW())
	`, userID, title, message, groupType, data)
	return err
}

// SaveBroadcastNotification persists a broadcast notification (user_id = NULL) visible to all users.
func (r *NotificationRepository) SaveBroadcastNotification(title, message, groupType, data string) error {
	_, err := r.db.Exec(`
		INSERT INTO notifications (user_id, title, message, group_type, is_read, data, "createdAt")
		VALUES (NULL, $1, $2, $3, FALSE, $4, NOW())
	`, title, message, groupType, data)
	return err
}

// ListByUserID returns paginated notifications for a user (targeted + broadcasts), newest first.
func (r *NotificationRepository) ListByUserID(userID, page, limit int) ([]NotificationRow, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	var rows []NotificationRow
	err := r.db.Select(&rows, `
		SELECT id, user_id, title, message, group_type,
		       CASE WHEN user_id IS NULL THEN TRUE ELSE is_read END AS is_read,
		       COALESCE(data,'') AS data, "createdAt"
		FROM notifications
		WHERE user_id = $1 OR user_id IS NULL
		ORDER BY "createdAt" DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	return rows, err
}

// MarkRead marks a single targeted notification as read (broadcast rows are unaffected).
func (r *NotificationRepository) MarkRead(id, userID int) error {
	_, err := r.db.Exec(`UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// CountUnread returns the number of unread targeted notifications for a user.
// Broadcast rows are excluded — they never count toward the unread badge.
func (r *NotificationRepository) CountUnread(userID int) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID)
	return count, err
}
