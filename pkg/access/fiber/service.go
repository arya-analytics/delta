package fiber

import (
	"github.com/arya-analytics/delta/pkg/access/rbac"
	"github.com/arya-analytics/x/gorp"
	"github.com/gofiber/fiber/v2"
)

type Service struct {
	DB         *gorp.DB
	Legislator *rbac.Legislator
}

func (s *Service) BindTo(parent fiber.Router) {
	router := parent.Group("/rbac")
	router.Post("/policy", s.createPolicy)
}

func (s *Service) createPolicy(c *fiber.Ctx) error {
	var p rbac.Policy
	if err := c.BodyParser(&p); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	txn := s.DB.BeginTxn()
	if err := s.Legislator.Create(txn, p); err != nil {
		c.Status(fiber.StatusBadRequest)
		return err
	}
	if err := txn.Commit(); err != nil {
		c.Status(fiber.StatusInternalServerError)
		return err
	}
	c.Status(fiber.StatusCreated)
	return nil
}
