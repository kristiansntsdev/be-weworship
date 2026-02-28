package middleware

import (
	"strings"

	"be-songbanks-v1/api/services"
	"be-songbanks-v1/api/types"
	"be-songbanks-v1/api/utils"
	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	auth *services.AuthService
}

func NewAuthMiddleware(auth *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{auth: auth}
}

func (m *AuthMiddleware) RequireAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	if token == "" {
		return utils.Fail(c, 401, "Access token required")
	}
	claims, err := m.auth.ParseToken(token)
	if err != nil {
		return utils.Fail(c, 401, "Invalid or expired token")
	}
	c.Locals("claims", claims)
	return c.Next()
}

func (m *AuthMiddleware) RequirePengurus(c *fiber.Ctx) error {
	claims := GetClaims(c)
	if claims == nil || claims.UserType != "pengurus" {
		return utils.Fail(c, 403, "Admin access required")
	}
	return c.Next()
}

func GetClaims(c *fiber.Ctx) *types.Claims {
	cl, _ := c.Locals("claims").(*types.Claims)
	return cl
}
