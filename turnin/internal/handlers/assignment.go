package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"turnin/internal/apierr"
	"turnin/internal/middleware"
	"turnin/internal/models"
)

type AssignmentHandler struct{ db *gorm.DB }

func NewAssignmentHandler(db *gorm.DB) *AssignmentHandler { return &AssignmentHandler{db: db} }

type assignmentView struct {
	models.Assignment
	SubmissionCount *int64             `json:"submission_count,omitempty"`
	Submission      *models.Submission `json:"submission"`
}

func (h *AssignmentHandler) List(c *fiber.Ctx) error {
	claims := middleware.ClaimsFrom(c)
	uid, err := claims.UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}
	ctx := c.UserContext()

	q := h.db.WithContext(ctx).Preload("Course").Order("deadline ASC")
	if claims.Role == models.RoleProfessor {
		owned := h.db.Model(&models.Course{}).Select("id").Where("professor_id = ?", uid)
		q = q.Where("course_id IN (?)", owned)
	}
	var assignments []models.Assignment
	if err := q.Find(&assignments).Error; err != nil {
		return err
	}

	views := make([]assignmentView, len(assignments))
	ids := make([]string, len(assignments))
	for i, a := range assignments {
		views[i] = assignmentView{Assignment: a}
		ids[i] = a.ID.String()
	}
	if len(ids) == 0 {
		return c.JSON(fiber.Map{"data": views})
	}

	switch claims.Role {
	case models.RoleProfessor:
		var rows []struct {
			AssignmentID string
			N            int64
		}
		if err := h.db.WithContext(ctx).Model(&models.Submission{}).
			Select("assignment_id, COUNT(*) AS n").
			Where("assignment_id IN ?", ids).
			Group("assignment_id").Scan(&rows).Error; err != nil {
			return err
		}
		counts := make(map[string]int64, len(rows))
		for _, r := range rows {
			counts[r.AssignmentID] = r.N
		}
		for i := range views {
			n := counts[views[i].ID.String()]
			views[i].SubmissionCount = &n
		}

	case models.RoleStudent:
		var subs []models.Submission
		if err := h.db.WithContext(ctx).
			Where("student_id = ? AND assignment_id IN ?", uid, ids).Find(&subs).Error; err != nil {
			return err
		}
		byAssignment := make(map[string]*models.Submission, len(subs))
		for i := range subs {
			byAssignment[subs[i].AssignmentID.String()] = &subs[i]
		}
		for i := range views {
			views[i].Submission = byAssignment[views[i].ID.String()]
		}
	}
	return c.JSON(fiber.Map{"data": views})
}

type createAssignmentRequest struct {
	CourseID    uuid.UUID `json:"course_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Deadline    time.Time `json:"deadline"`
	MaxScore    int       `json:"max_score"`
}

func (h *AssignmentHandler) Create(c *fiber.Ctx) error {
	uid, err := middleware.ClaimsFrom(c).UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}

	var req createAssignmentRequest
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest("invalid_body", "Request body must be valid JSON")
	}
	req.Title = strings.TrimSpace(req.Title)
	switch {
	case req.CourseID == uuid.Nil:
		return apierr.BadRequest("invalid_course", "course_id is required")
	case req.Title == "" || len(req.Title) > 200:
		return apierr.BadRequest("invalid_title", "title must be 1-200 characters")
	case req.MaxScore < 1 || req.MaxScore > 1000:
		return apierr.BadRequest("invalid_max_score", "max_score must be between 1 and 1000")
	case !req.Deadline.After(time.Now()):
		return apierr.BadRequest("invalid_deadline", "deadline must be in the future")
	}

	var owned int64
	if err := h.db.WithContext(c.UserContext()).Model(&models.Course{}).
		Where("id = ? AND professor_id = ?", req.CourseID, uid).Count(&owned).Error; err != nil {
		return err
	}
	if owned == 0 {
		return apierr.Forbidden("not_course_owner", "You do not teach this course")
	}

	a := models.Assignment{
		CourseID:    req.CourseID,
		Title:       req.Title,
		Description: strings.TrimSpace(req.Description),
		Deadline:    req.Deadline.UTC(),
		MaxScore:    req.MaxScore,
	}
	if err := h.db.WithContext(c.UserContext()).Create(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			return apierr.NotFound("course_not_found", "Course does not exist")
		}
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": a})
}
