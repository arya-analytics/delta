package fiber

import (
	"github.com/arya-analytics/delta/pkg/resource"
	"github.com/arya-analytics/x/query"
	"github.com/gofiber/fiber/v2"
)

type Service struct{ reader resource.Reader }

func (s *Service) BindTo(parent fiber.Router) {
	router := parent.Group("/resource")
	router.Get("/", s.root)
	router.Get("/:key", s.children)
	router.Get("/:key/children", s.children)
	router.Get("/:key/parents", s.children)
}

func (s *Service) root(c *fiber.Ctx) error {
	root, err := s.reader.GetResource(resource.RootKey)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(root)
}

func (s *Service) get(c *fiber.Ctx) error {
	key, err := s.parseKey(c)
	if err != nil {
		return err
	}
	res, err := s.reader.GetResource(key)
	if err != query.NotFound {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{"error": err.Error()})
	} else if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(res)
}

func (s *Service) children(c *fiber.Ctx) error {
	key, err := s.parseKey(c)
	if err != nil {
		return err
	}
	children, err := s.reader.GetChildResources(key)
	if err != query.NotFound {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{"error": err.Error()})
	} else if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(children)
}

func (s *Service) parents(c *fiber.Ctx) error {
	key, err := s.parseKey(c)
	if err != nil {
		return err
	}
	parents, err := s.reader.GetParentResources(key)
	if err != query.NotFound {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{"error": err.Error()})
	} else if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(parents)
}

func (s *Service) parseKey(c *fiber.Ctx) (resource.Key, error) {
	var key resource.Key
	if err := c.BodyParser(&key); err != nil {
		c.Status(fiber.StatusBadRequest)
		return key, c.JSON(fiber.Map{"error": err.Error()})
	}
	return key, nil
}
