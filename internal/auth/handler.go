package auth

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

var Store *session.Store

func Init() {
	Store = session.New(session.Config{
		Expiration: 24 * time.Hour,
	})
}

func CheckCredentials(username, password string) bool {
	// 1. Try to get from Environment Variables (Set by install.sh)
	envUser := os.Getenv("CONTROLP_USER")
	envPass := os.Getenv("CONTROLP_PASS")

	// 2. Fallback defaults (if not set)
	if envUser == "" {
		envUser = "admin"
	}
	if envPass == "" {
		envPass = "password123"
	}

	return username == envUser && password == envPass
}

func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := Store.Get(c)
		if err != nil {
			return c.Redirect("/login")
		}
		if sess.Get("authenticated") != true {
			return c.Redirect("/login")
		}
		return c.Next()
	}
}
