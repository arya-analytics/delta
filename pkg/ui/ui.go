package main

import (
	"embed"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"net/http"
)

//go:embed dist/* dist/_app/immutable/assets/* dist/_app/immutable/pages/* dist/_app/immutable/chunks/*
var dist embed.FS

type Service struct{}

func (s *Service) BindTo(f fiber.Router) {
	f.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(dist),
		PathPrefix: "dist",
		Browse:     true,
	}))
}

func main() {
	app := fiber.New()
	s := &Service{}
	s.BindTo(app)
	app.Listen(":3000")
}
