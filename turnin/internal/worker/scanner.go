// Package worker hosts background jobs that must outlive the HTTP request that
// triggered them.
package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"turnin/internal/models"
)

// Scanner simulates a heavy plagiarism scan. Design notes:
//   - Dispatch is non-blocking, so the HTTP handler can return 201 immediately.
//   - A WaitGroup lets the process drain in-flight scans on graceful shutdown.
//   - The status update is a compare-and-set (WHERE status = 'Pending'), so a
//     scan can never clobber a status written by someone else (e.g. a grade).
//   - Recover() re-dispatches rows left 'Pending' by a crash or hard stop, so
//     in-process goroutines never leave work permanently orphaned.
type Scanner struct {
	db       *gorm.DB
	duration time.Duration
	wg       sync.WaitGroup
}

func NewScanner(db *gorm.DB, duration time.Duration) *Scanner {
	return &Scanner{db: db, duration: duration}
}

// Dispatch spawns the background goroutine and returns immediately.
func (s *Scanner) Dispatch(id uuid.UUID) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[scanner] submission=%s status=panic err=%v", id, r)
			}
		}()
		s.scan(id)
	}()
}

func (s *Scanner) scan(id uuid.UUID) {
	started := time.Now()
	log.Printf("[scanner] submission=%s status=started simulated_work=%s", id, s.duration)

	time.Sleep(s.duration) // stand-in for fingerprinting + corpus comparison

	res := s.db.Model(&models.Submission{}).
		Where("id = ? AND status = ?", id, models.StatusPending).
		Update("status", models.StatusScanned)

	switch {
	case res.Error != nil:
		log.Printf("[scanner] submission=%s status=failed err=%v", id, res.Error)
	case res.RowsAffected == 0:
		log.Printf("[scanner] submission=%s status=skipped reason=no-longer-pending", id)
	default:
		log.Printf("[scanner] submission=%s status=completed result=%q elapsed=%s",
			id, models.StatusScanned, time.Since(started).Round(time.Millisecond))
	}
}

// Recover re-queues submissions stranded in 'Pending' (e.g. after a crash).
func (s *Scanner) Recover() (int, error) {
	var stranded []models.Submission
	if err := s.db.Select("id").Where("status = ?", models.StatusPending).Find(&stranded).Error; err != nil {
		return 0, err
	}
	for _, sub := range stranded {
		s.Dispatch(sub.ID)
	}
	return len(stranded), nil
}

// Wait blocks until in-flight scans finish or ctx expires. Anything left
// unfinished stays 'Pending' and is picked up by Recover on next boot.
func (s *Scanner) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

