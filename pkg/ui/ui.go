package ui

import (
	"embed"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"net/http"
)

//go:embed dist/*
var dist embed.FS

type Service struct{}

func (s *Service) BindTo(f fiber.Router) {
	f.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(dist),
		PathPrefix: "dist",
		Browse:     true,
	}))
}
