package handlers

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"turnin/internal/apierr"
	"turnin/internal/middleware"
	"turnin/internal/models"
	"turnin/internal/worker"
)

var allowedExt = map[string]struct{}{
	".pdf": {}, ".doc": {}, ".docx": {}, ".zip": {}, ".txt": {}, ".md": {},
	".py": {}, ".go": {}, ".java": {}, ".cpp": {}, ".ipynb": {},
}

var (
	unsafeChars  = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	storedPrefix = regexp.MustCompile(`^[0-9a-f-]{36}_`)
)

type SubmissionHandler struct {
	db        *gorm.DB
	scanner   *worker.Scanner
	uploadDir string
	maxBytes  int64
}

func NewSubmissionHandler(db *gorm.DB, scanner *worker.Scanner, uploadDir string, maxBytes int64) *SubmissionHandler {
	return &SubmissionHandler{db: db, scanner: scanner, uploadDir: uploadDir, maxBytes: maxBytes}
}

type gradeSubmissionRequest struct {
	Score *int `json:"score"`
}

// Grade handles PATCH /api/submissions/:id/grade (professor only).
// It validates course ownership (preventing cross-course IDOR), enforces
// 0 <= score <= assignment.max_score, and transitions Status to 'Graded'.
func (h *SubmissionHandler) Grade(c *fiber.Ctx) error {
	profID, err := middleware.ClaimsFrom(c).UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}

	subID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apierr.BadRequest("invalid_id", "Submission id must be a UUID")
	}

	var req gradeSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest("invalid_body", "Request body must be valid JSON")
	}
	if req.Score == nil {
		return apierr.BadRequest("missing_score", "score is required")
	}

	ctx := c.UserContext()
	var sub models.Submission
	if err := h.db.WithContext(ctx).
		Preload("Assignment.Course").
		Preload("Student").
		First(&sub, "id = ?", subID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierr.NotFound("submission_not_found", "Submission not found")
		}
		return err
	}

	// Ownership check: a professor may only grade submissions for courses they teach.
	if sub.Assignment == nil || sub.Assignment.Course == nil || sub.Assignment.Course.ProfessorID != profID {
		return apierr.Forbidden("not_course_owner", "You do not teach the course for this submission")
	}

	score := *req.Score
	if score < 0 || score > sub.Assignment.MaxScore {
		return apierr.Unprocessable(
			"score_out_of_range",
			"Score must be between 0 and the assignment max_score",
		)
	}

	sub.Score = &score
	sub.Status = models.StatusGraded

	if err := h.db.WithContext(ctx).
		Model(&sub).
		Updates(map[string]any{"score": score, "status": models.StatusGraded}).Error; err != nil {
		return err
	}

	log.Printf("[submissions] graded id=%s professor=%s score=%d/%d",
		sub.ID, profID, score, sub.Assignment.MaxScore)

	return c.JSON(fiber.Map{"data": sub})
}

// Create handles POST /api/assignments/:id/submissions (multipart, field "file").
//
// It persists the submission as 'Pending', hands it to the background scanner,
// and returns 201 right away. The client is never blocked on the scan.
func (h *SubmissionHandler) Create(c *fiber.Ctx) error {
	studentID, err := middleware.ClaimsFrom(c).UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}
	assignmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apierr.BadRequest("invalid_id", "Assignment id must be a UUID")
	}
	ctx := c.UserContext()

	var assignment models.Assignment
	if err := h.db.WithContext(ctx).First(&assignment, "id = ?", assignmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierr.NotFound("assignment_not_found", "Assignment does not exist")
		}
		return err
	}
	if !time.Now().Before(assignment.Deadline) {
		return apierr.Unprocessable("deadline_passed", "The deadline for this assignment has passed")
	}

	var existing int64
	if err := h.db.WithContext(ctx).Model(&models.Submission{}).
		Where("assignment_id = ? AND student_id = ?", assignmentID, studentID).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return apierr.Conflict("already_submitted", "You have already submitted this assignment")
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return apierr.BadRequest("file_required", "A file must be attached in the 'file' form field")
	}
	if fh.Size == 0 || fh.Size > h.maxBytes {
		return apierr.BadRequest("invalid_file_size", "File must be between 1 byte and 10 MB")
	}
	if _, ok := allowedExt[strings.ToLower(filepath.Ext(fh.Filename))]; !ok {
		return apierr.BadRequest("unsupported_file_type", "Unsupported file type")
	}

	name := storageName(fh.Filename)
	dst := filepath.Join(h.uploadDir, name)
	if err := c.SaveFile(fh, dst); err != nil {
		return err
	}

	sub := models.Submission{
		AssignmentID: assignmentID,
		StudentID:    studentID,
		FileURL:      "/uploads/" + name,
		Status:       models.StatusPending,
		SubmittedAt:  time.Now().UTC(),
	}
	if err := h.db.WithContext(ctx).Create(&sub).Error; err != nil {
		_ = os.Remove(dst) // don't orphan the blob if the row didn't land
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return apierr.Conflict("already_submitted", "You have already submitted this assignment")
		}
		return err
	}

	log.Printf("[submissions] created id=%s assignment=%s student=%s file=%s", sub.ID, assignmentID, studentID, name)
	h.scanner.Dispatch(sub.ID) // fire-and-forget; see worker.Scanner

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": sub})
}

// List is role-scoped: students see their own submissions, professors see
// every submission across the courses they teach.
func (h *SubmissionHandler) List(c *fiber.Ctx) error {
	claims := middleware.ClaimsFrom(c)
	uid, err := claims.UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}

	q := h.db.WithContext(c.UserContext()).Preload("Assignment.Course").Order("submitted_at DESC")
	if claims.Role == models.RoleProfessor {
		courses := h.db.Model(&models.Course{}).Select("id").Where("professor_id = ?", uid)
		assignments := h.db.Model(&models.Assignment{}).Select("id").Where("course_id IN (?)", courses)
		q = q.Preload("Student").Where("assignment_id IN (?)", assignments)
	} else {
		q = q.Where("student_id = ?", uid)
	}

	var subs []models.Submission
	if err := q.Find(&subs).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": subs})
}

// Download streams the stored file to the owning student or the course professor.
// Unauthorised callers get 404 (not 403) so submission ids can't be probed.
func (h *SubmissionHandler) Download(c *fiber.Ctx) error {
	claims := middleware.ClaimsFrom(c)
	uid, err := claims.UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apierr.BadRequest("invalid_id", "Submission id must be a UUID")
	}

	var sub models.Submission
	if err := h.db.WithContext(c.UserContext()).Preload("Assignment.Course").First(&sub, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierr.NotFound("submission_not_found", "Submission not found")
		}
		return err
	}

	allowed := false
	switch claims.Role {
	case models.RoleStudent:
		allowed = sub.StudentID == uid
	case models.RoleProfessor:
		allowed = sub.Assignment != nil && sub.Assignment.Course != nil && sub.Assignment.Course.ProfessorID == uid
	}
	if !allowed {
		return apierr.NotFound("submission_not_found", "Submission not found")
	}

	// filepath.Base neutralises any traversal attempt in the stored value.
	path := filepath.Join(h.uploadDir, filepath.Base(sub.FileURL))
	if _, err := os.Stat(path); err != nil {
		return apierr.NotFound("file_missing", "The stored file is no longer available")
	}
	return c.Download(path, storedPrefix.ReplaceAllString(filepath.Base(sub.FileURL), ""))
}

// storageName never trusts the client filename: UUID prefix for uniqueness,
// sanitised stem for human-readability, allow-listed extension.
func storageName(original string) string {
	base := filepath.Base(original)
	ext := strings.ToLower(filepath.Ext(base))
	stem := unsafeChars.ReplaceAllString(strings.TrimSuffix(base, filepath.Ext(base)), "_")
	if len(stem) > 64 {
		stem = stem[:64]
	}
	return uuid.NewString() + "_" + stem + ext
}
