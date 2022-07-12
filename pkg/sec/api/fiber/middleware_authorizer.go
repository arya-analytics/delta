package fiber

import (
	"github.com/arya-analytics/delta/pkg/sec"
	"github.com/gofiber/fiber/v2"
)

func AuthorizerMiddleware(
	enforcer sec.AccessEnforcer,
	resource sec.Resource,
	action sec.Action,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return enforcer.Enforce(getUserSubject(c), resource, action)
	}
}
