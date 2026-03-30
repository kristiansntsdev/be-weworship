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
		println("[DEBUG] RequireAuth: no token in header")
		return utils.Fail(c, 401, "Access token required")
	}
	claims, err := m.auth.ParseToken(token)
	if err != nil {
		println("[DEBUG] RequireAuth: ParseToken error:", err.Error())
		return utils.Fail(c, 401, "Invalid or expired token")
	}
	println("[DEBUG] RequireAuth: userId=", claims.UserID, "role=", claims.Role)
	c.Locals("claims", claims)
	return c.Next()
}

func (m *AuthMiddleware) RequireAdmin(c *fiber.Ctx) error {
	claims := GetClaims(c)
	if claims == nil || claims.Role != "admin" {
		return utils.Fail(c, 403, "Admin access required")
	}
	return c.Next()
}

func (m *AuthMiddleware) RequireMaintainer(c *fiber.Ctx) error {
	claims := GetClaims(c)
	if claims == nil {
		println("[DEBUG] RequireMaintainer: claims is nil")
		return utils.Fail(c, 403, "Maintainer access required")
	}
	println("[DEBUG] RequireMaintainer: userId=", claims.UserID, "role=", claims.Role)
	if claims.Role != "admin" && claims.Role != "maintainer" {
		println("[DEBUG] RequireMaintainer: role check failed, role=", claims.Role)
		return utils.Fail(c, 403, "Maintainer access required")
	}
	return c.Next()
}

func GetClaims(c *fiber.Ctx) *types.Claims {
	cl, _ := c.Locals("claims").(*types.Claims)
	return cl
}
