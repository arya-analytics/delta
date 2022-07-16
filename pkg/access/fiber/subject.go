package fiber

import (
	"github.com/arya-analytics/delta/pkg/ontology"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
)

const subjectKey = "subject"

func SetSubject(c *fiber.Ctx, key ontology.Key) { c.Locals(subjectKey, key) }

// GetSubject retrieves the subject of a request (the entity attempting to perform
// an action on an object). Returns false if the subject is not set on the request.
func GetSubject(c *fiber.Ctx) (ontology.Key, error) {
	key, ok := c.Locals(subjectKey).(ontology.Key)
	if !ok {
		c.Status(fiber.StatusInternalServerError)
		return key, errors.New("[access] - subject not set on query. this is a bug.")
	}
	return key, nil
}
