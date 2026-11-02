package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"turnin/internal/auth"
	"turnin/internal/config"
	"turnin/internal/handlers"
	"turnin/internal/middleware"
	"turnin/internal/models"
	"turnin/internal/worker"
)

type Deps struct {
	Config  *config.Config
	DB      *gorm.DB
	Auth    *auth.Service
	Scanner *worker.Scanner
}

func Register(app *fiber.App, d Deps) {
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} ${method} ${path} ${latency}\n",
		TimeFormat: "15:04:05.000",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: d.Config.AllowedOrigins,
		AllowHeaders: "Authorization,Content-Type,Idempotency-Key",
		AllowMethods: "GET,POST,PATCH,OPTIONS",
		MaxAge:       600,
	}))

	authH := handlers.NewAuthHandler(d.DB, d.Auth)
	asgH := handlers.NewAssignmentHandler(d.DB)
	subH := handlers.NewSubmissionHandler(d.DB, d.Scanner, d.Config.UploadDir, d.Config.MaxUploadBytes)

	authenticated := middleware.Authenticate(d.Auth)
	professorOnly := middleware.RequireRole(models.RoleProfessor)
	studentOnly := middleware.RequireRole(models.RoleStudent)

	app.Get("/healthz", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	api := app.Group("/api")

	api.Post("/auth/login", limiter.New(limiter.Config{Max: 10, Expiration: time.Minute}), authH.Login)
	api.Get("/auth/me", authenticated, authH.Me)

	api.Get("/assignments", authenticated, asgH.List)
	api.Post("/assignments", authenticated, professorOnly, asgH.Create)

	api.Get("/submissions", authenticated, subH.List)
	api.Post("/assignments/:id/submissions", authenticated, studentOnly, subH.Create)
	api.Patch("/submissions/:id/grade", authenticated, professorOnly, subH.Grade) // RBAC: professors only
	api.Get("/submissions/:id/file", authenticated, subH.Download)
}

