package main
import "turnin/internal/routes"


import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"turnin/internal/apierr"
	"turnin/internal/auth"
	"turnin/internal/config"
	"turnin/internal/database"
	"turnin/internal/worker"
)

func main() {
	seed := flag.Bool("seed", false, "apply the embedded idempotent seed data before serving")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o750); err != nil {
		log.Fatalf("uploads dir: %v", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if *seed {
		if err := database.Seed(db); err != nil {
			log.Fatalf("seed: %v", err)
		}
		log.Println("[seed] applied")
	}

	scanner := worker.NewScanner(db, cfg.ScanDuration)
	if n, err := scanner.Recover(); err != nil {
		log.Printf("[scanner] recovery failed: %v", err)
	} else if n > 0 {
		log.Printf("[scanner] recovered %d stranded submission(s)", n)
	}

	app := fiber.New(fiber.Config{
		AppName:      "turnin",
		ErrorHandler: apierr.Handler,
		BodyLimit:    int(cfg.MaxUploadBytes) + (1 << 20), // file + multipart overhead
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	})
	routes.Register(app, routes.Deps{
		Config: cfg, DB: db, Auth: auth.NewService(cfg.JWTSecret, cfg.JWTTTL), Scanner: scanner,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- app.Listen(cfg.Addr) }()

	select {
	case err := <-serveErr:
		log.Fatalf("server: %v", err)
	case <-ctx.Done():
		log.Println("[shutdown] signal received, draining")
	}

	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("[shutdown] http: %v", err)
	}
	drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := scanner.Wait(drain); err != nil {
		log.Println("[shutdown] scans still in flight; they remain 'Pending' and will be recovered on next boot")
	}
}

