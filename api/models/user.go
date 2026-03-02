package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID         int            `db:"id"`
	Name       string         `db:"name"`
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
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}
