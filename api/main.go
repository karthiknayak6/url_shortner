package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/karthiknayak6/url-shortner/routes"
)

func setUpRoutes(app *fiber.App) {
	app.Get("/", renderHome )
	app.Get("/:url", routes.ResolveURL)
	app.Post("/api/v1", routes.ShortenURL)
}

func main() {
	
	err := godotenv.Load()

	if err != nil {
		fmt.Println(err)
	}

	app := fiber.New()

	app.Use(logger.New())

	setUpRoutes(app)

	log.Fatal(app.Listen(os.Getenv("APP_PORT")))

}

func renderHome(c *fiber.Ctx) error {
	return c.Render("./views/index.html", fiber.Map{
        "hello": "world",
    });
}