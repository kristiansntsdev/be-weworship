package utils

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func OK(c *fiber.Ctx, status int, message string, data any) error {
	resp := fiber.Map{"code": status, "message": message}
	if data != nil {
		resp["data"] = data
	}
	return c.Status(status).JSON(resp)
}

func Fail(c *fiber.Ctx, status int, message string) error {
	logFailure(c, status, message, nil)
	return c.Status(status).JSON(fiber.Map{"code": status, "message": message})
}

func FailErr(c *fiber.Ctx, status int, message string, err error) error {
	logFailure(c, status, message, err)
	return c.Status(status).JSON(fiber.Map{"code": status, "message": message})
}

func logFailure(c *fiber.Ctx, status int, message string, err error) {
	if err != nil {
		log.Printf("[HTTP] Error on %s %s status=%d message=%q cause=%v ip=%s ua=%q",
			c.Method(), c.OriginalURL(), status, message, err, c.IP(), c.Get("User-Agent"))
		return
	}
	log.Printf("[HTTP] Error response on %s %s status=%d message=%q ip=%s ua=%q",
		c.Method(), c.OriginalURL(), status, message, c.IP(), c.Get("User-Agent"))
}
