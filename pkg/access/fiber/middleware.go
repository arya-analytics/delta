package fiber

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/resource"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
)

// StaticMiddleware is a middleware whose action and object access parameters can
// be described at runtime as opposed to request time.
func StaticMiddleware(
	object resource.Key,
	action access.Action,
	enforcer access.Enforcer,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key, err := GetSubject(c)
		if err != nil {
			return err
		}
		err = enforcer.Enforce(access.Request{
			Subject: key,
			Object:  object,
			Action:  action,
		})
		if errors.Is(err, access.Forbidden) {
			c.Status(fiber.StatusForbidden)
			return err
		}
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return err
		}
		return c.Next()
	}
}
