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
