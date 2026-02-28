package types

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID    int    `json:"userId"`
	UserType  string `json:"userType"`
	Username  string `json:"username"`
	UserLevel string `json:"userlevel,omitempty"`
	jwt.RegisteredClaims
}
