package types

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID int    `json:"userId"`
	Role   string `json:"role"` // "user" | "admin"
	Name   string `json:"name"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
