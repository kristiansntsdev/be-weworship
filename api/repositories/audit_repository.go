package repositories

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AuditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Log writes a single audit entry. Fire-and-forget — errors are silently dropped.
func (r *AuditRepository) Log(userID *int, userName, userEmail, action, entityType string, entityID *int, entityName string, changes any) {
	var changesJSON []byte
	if changes != nil {
		changesJSON, _ = json.Marshal(changes)
	}
	r.db.Exec(r.db.Rebind(`
		INSERT INTO audit_logs (user_id, user_name, user_email, action, entity_type, entity_id, entity_name, changes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		userID, userName, userEmail, action, entityType, entityID, entityName, changesJSON)
}

type AuditLogRow struct {
	ID         int     `db:"id"          json:"id"`
	UserID     *int    `db:"user_id"     json:"user_id"`
	UserName   string  `db:"user_name"   json:"user_name"`
	UserEmail  string  `db:"user_email"  json:"user_email"`
	Action     string  `db:"action"      json:"action"`
	EntityType string  `db:"entity_type" json:"entity_type"`
	EntityID   *int    `db:"entity_id"   json:"entity_id"`
	EntityName *string `db:"entity_name" json:"entity_name"`
	Changes    *string `db:"changes"     json:"changes"`
	CreatedAt  string  `db:"createdAt"   json:"createdAt"`
}

func (r *AuditRepository) List(action, entityType string, userID *int, page, limit int) ([]AuditLogRow, int, error) {
	where := "WHERE 1=1"
	args := []any{}

	if action != "" {
		args = append(args, action)
		where += fmt.Sprintf(" AND action = $%d", len(args))
	}
	if entityType != "" {
		args = append(args, entityType)
		where += fmt.Sprintf(" AND entity_type = $%d", len(args))
	}
	if userID != nil {
		args = append(args, *userID)
		where += fmt.Sprintf(" AND user_id = $%d", len(args))
	}

	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM audit_logs `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	rows := []AuditLogRow{}
	err = r.db.Select(&rows, fmt.Sprintf(`
		SELECT id, user_id, user_name, user_email, action, entity_type, entity_id, entity_name,
		       changes::text AS changes, "createdAt"::text AS "createdAt"
		FROM audit_logs %s
		ORDER BY "createdAt" DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)),
		args...)
	return rows, total, err
}
