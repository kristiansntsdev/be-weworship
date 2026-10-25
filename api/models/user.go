package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID         int            `db:"id"`
	Username   string         `db:"username"`
	Email      string         `db:"email"`
	Password   sql.NullString `db:"password"`
	AvatarURL  sql.NullString `db:"avatar_url"`
	Role       string         `db:"role"`
	Provider   string         `db:"provider"`
	ProviderID sql.NullString `db:"provider_id"`
	Verified   bool           `db:"verified"`
	Status     string         `db:"status"`
	CreatedAt  time.Time      `db:"createdAt"`
	UpdatedAt  time.Time      `db:"updatedAt"`
}

type UserDetail struct {
	UserID     int            `db:"user_id"`
	FullName   sql.NullString `db:"full_name"`
	Province   sql.NullString `db:"province"`
	City       sql.NullString `db:"city"`
	PostalCode sql.NullString `db:"postal_code"`
	CreatedAt  time.Time      `db:"createdAt"`
	UpdatedAt  time.Time      `db:"updatedAt"`
}

type UserBasic struct {
	ID       int    `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Email    string `db:"email" json:"email"`
}

// VerificationRequest is a user application to become an official (verified) account.
type VerificationRequest struct {
	ID          int            `db:"id"           json:"id"`
	UserID      int            `db:"user_id"      json:"user_id"`
	DisplayName string         `db:"display_name" json:"display_name"`
	ProofLink   string         `db:"proof_link"   json:"proof_link"`
	Note        sql.NullString `db:"note"         json:"-"`
	Status      string         `db:"status"       json:"status"`
	AdminNotes  sql.NullString `db:"admin_notes"  json:"-"`
	CreatedAt   time.Time      `db:"createdAt"    json:"createdAt"`
	UpdatedAt   time.Time      `db:"updatedAt"    json:"updatedAt"`
	// Joined from users for admin list (empty for user-facing reads).
	Username string `db:"username" json:"username,omitempty"`
	Email    string `db:"email"    json:"email,omitempty"`
}
