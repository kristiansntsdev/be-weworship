package utils

import "github.com/gofiber/fiber/v2"

func OK(c *fiber.Ctx, status int, message string, data any) error {
	resp := fiber.Map{"code": status, "message": message}
	if data != nil {
		resp["data"] = data
	}
	return c.Status(status).JSON(resp)
}

func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"code": status, "message": message})
}
