package fiber

import (
	"github.com/arya-analytics/delta/pkg/sec"
	"github.com/arya-analytics/delta/pkg/sec/password"
	"github.com/arya-analytics/delta/pkg/sec/token"
	"github.com/gofiber/fiber/v2"
)

func NewAuthenticationRouter(cfg Config, router fiber.Router) {
	router.Post("/login", Login(cfg))
	router.Post("/register", Register(cfg))
	protected := router.Group("/protected").Use(TokenMiddleware(cfg.Token))
	protected.Post("/change-password", AuthorizerMiddleware())
}

func Login(cfg Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var creds sec.Credentials
		if err := c.BodyParser(&creds); err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		if err := cfg.Authenticator.Authenticate(creds); err != nil {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		user, err := cfg.Users.RetrieveByUsername(creds.Username)
		if err != nil {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		c.Status(fiber.StatusOK)
		return generateTokenResponse(c, cfg.Token, user)
	}
}

func Register(cfg Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var creds sec.Credentials
		if err := c.BodyParser(&creds); err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		if err := cfg.Authenticator.Register(creds); err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		u := &sec.User{Username: creds.Username}
		if err := cfg.Users.Register(u); err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		c.Status(fiber.StatusCreated)
		return generateTokenResponse(c, cfg.Token, *u)
	}
}

type changePasswordRequest struct {
	creds       sec.Credentials
	NewPassword password.Raw `json:"newPassword"`
}

func ChangePassword(cfg Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cpr := changePasswordRequest{}
		if err := c.BodyParser(&cpr); err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		if err := cfg.Authenticator.UpdatePassword(cpr.creds, cpr.NewPassword); err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{"error": err.Error()})
		}
		c.Status(fiber.StatusOK)
		return nil
	}
}

func generateTokenResponse(
	c *fiber.Ctx,
	svc *token.Service,
	user sec.User,
) error {
	tk, err := svc.New(user.Key)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"user": user, "token": tk})
}
