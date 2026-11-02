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


// 5. Professor Grading, Bounds Validation, IDOR & Student Visibility Test
func TestProfessorGradingLifecycle(t *testing.T) {
	env := setupTestEnv(t, 20*time.Millisecond)
	stud1Tok, _ := env.authSvc.Issue(&env.stud1)
	prof1Tok, _ := env.authSvc.Issue(&env.prof1)
	prof2Tok, _ := env.authSvc.Issue(&env.prof2)

	// Student 1 submits work on openAsg (MaxScore = 100)
	upReq := buildMultipartReq(
		t,
		"/api/assignments/"+env.openAsg.ID.String()+"/submissions",
		stud1Tok,
		"rbtree.cpp",
		[]byte("#include <iostream>\nint main() { return 0; }\n"),
	)
	upRes, _ := env.app.Test(upReq, -1)
	var created struct {
		Data models.Submission `json:"data"`
	}
	raw, _ := io.ReadAll(upRes.Body)
	_ = json.Unmarshal(raw, &created)
	gradeURL := "/api/submissions/" + created.Data.ID.String() + "/grade"

	// 1. Student attempting to grade their own submission -> 403 Forbidden
	studGradeReq := httptest.NewRequest(http.MethodPatch, gradeURL, bytes.NewBufferString(`{"score":100}`))
	studGradeReq.Header.Set("Content-Type", "application/json")
	studGradeReq.Header.Set("Authorization", "Bearer "+stud1Tok)
	if r, _ := env.app.Test(studGradeReq, -1); r.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for student grading, got %d", r.StatusCode)
	}

	// 2. Non-owning Professor 2 attempting to grade -> 403 Forbidden
	prof2GradeReq := httptest.NewRequest(http.MethodPatch, gradeURL, bytes.NewBufferString(`{"score":90}`))
	prof2GradeReq.Header.Set("Content-Type", "application/json")
	prof2GradeReq.Header.Set("Authorization", "Bearer "+prof2Tok)
	if r, _ := env.app.Test(prof2GradeReq, -1); r.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-course professor grading, got %d", r.StatusCode)
	}

	// 3. Score > MaxScore (105 > 100) -> 422 Unprocessable Entity
	overGradeReq := httptest.NewRequest(http.MethodPatch, gradeURL, bytes.NewBufferString(`{"score":105}`))
	overGradeReq.Header.Set("Content-Type", "application/json")
	overGradeReq.Header.Set("Authorization", "Bearer "+prof1Tok)
	if r, _ := env.app.Test(overGradeReq, -1); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for score > max_score, got %d", r.StatusCode)
	}

	// 4. Owning Professor 1 grades with 92 -> 200 OK, status = "Graded", score = 92
	okGradeReq := httptest.NewRequest(http.MethodPatch, gradeURL, bytes.NewBufferString(`{"score":92}`))
	okGradeReq.Header.Set("Content-Type", "application/json")
	okGradeReq.Header.Set("Authorization", "Bearer "+prof1Tok)
	okRes, _ := env.app.Test(okGradeReq, -1)
	if okRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on valid grade, got %d", okRes.StatusCode)
	}

	// 5. Wait for background scanner to finish and verify it did NOT clobber "Graded"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = env.scanner.Wait(ctx)

	// 6. Student fetches GET /api/assignments and sees status="Graded" and score=92
	asgReq := httptest.NewRequest(http.MethodGet, "/api/assignments", nil)
	asgReq.Header.Set("Authorization", "Bearer "+stud1Tok)
	asgRes, _ := env.app.Test(asgReq, -1)
	var asgList struct {
		Data []struct {
			ID         uuid.UUID          `json:"id"`
			Submission *models.Submission `json:"submission"`
		} `json:"data"`
	}
	asgBytes, _ := io.ReadAll(asgRes.Body)
	_ = json.Unmarshal(asgBytes, &asgList)

	for _, a := range asgList.Data {
		if a.ID == env.openAsg.ID {
			if a.Submission == nil || a.Submission.Status != models.StatusGraded || a.Submission.Score == nil || *a.Submission.Score != 92 {
				t.Fatalf("student view mismatch after grading: %+v", a.Submission)
			}
		}
	}
}
