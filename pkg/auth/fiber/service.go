package fiber

import (
	"github.com/arya-analytics/delta/pkg/access"
	fiberaccess "github.com/arya-analytics/delta/pkg/access/fiber"
	"github.com/arya-analytics/delta/pkg/auth"
	"github.com/arya-analytics/delta/pkg/auth/token"
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/arya-analytics/delta/pkg/password"
	"github.com/arya-analytics/delta/pkg/user"
	"github.com/arya-analytics/x/gorp"
	"github.com/gofiber/fiber/v2"
)

type Service struct {
	User     *user.Service
	Token    *token.Service
	DB       *gorp.DB
	Auth     auth.Authenticator
	Enforcer access.Enforcer
}

func (s *Service) BindTo(parent fiber.Router) {
	router := parent.Group("/auth")
	router.Post("/login", s.login)
	router.Post("/register", s.register)
	protected := parent.Group("/protected")
	protected.Use(TokenMiddleware(s.Token))
	protected.Use(fiberaccess.StaticMiddleware(
		ontology.RouteKey("/auth/protected"),
		access.ActionIrrelivant,
		s.Enforcer,
	))
	protected.Post("/change-password", s.changePassword)
	protected.Post("/change-username", s.changeUsername)
}

func (s *Service) login(c *fiber.Ctx) error {
	var creds auth.InsecureCredentials
	if err := c.BodyParser(&creds); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := s.Auth.Authenticate(creds); err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	u, err := s.User.RetrieveByUsername(creds.Username)
	if err != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	c.Status(fiber.StatusOK)
	return s.tokenResponse(c, u)
}

type registrationRequest struct {
	auth.InsecureCredentials
}

func (s *Service) register(c *fiber.Ctx) error {
	var req registrationRequest
	txn := s.DB.BeginTxn()
	if err := c.BodyParser(&req); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := s.Auth.Register(txn, req.InsecureCredentials); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	u := &user.User{Username: req.Username}
	if err := s.User.Create(txn, u); err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := txn.Commit(); err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return s.tokenResponse(c, *u)
}

type changePasswordRequest struct {
	auth.InsecureCredentials
	NewPassword password.Raw `json:"newPassword"`
}

func (s *Service) changePassword(c *fiber.Ctx) error {
	var cpr changePasswordRequest
	txn := s.DB.BeginTxn()
	if err := c.BodyParser(&cpr); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := s.Auth.UpdatePassword(
		txn,
		cpr.InsecureCredentials,
		cpr.NewPassword,
	); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := txn.Commit(); err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	c.Status(fiber.StatusNoContent)
	return nil
}

type changeUserNameRequest struct {
	auth.InsecureCredentials
	NewUsername string `json:"username"`
}

func (s *Service) changeUsername(c *fiber.Ctx) error {
	var cpr changeUserNameRequest
	txn := s.DB.BeginTxn()
	if err := c.BodyParser(&cpr); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := s.Auth.UpdateUsername(
		txn,
		cpr.InsecureCredentials,
		cpr.NewUsername,
	); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	if err := txn.Commit(); err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	c.Status(fiber.StatusNoContent)
	return nil
}

func (s *Service) tokenResponse(c *fiber.Ctx, u user.User) error {
	tk, err := s.Token.New(u.Key)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"user": u, "Token": tk})
}
