package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Role string

const (
	RoleStudent   Role = "student"
	RoleProfessor Role = "professor"
)

// Submission lifecycle. Status is intentionally a free-form string (per spec),
// so the scanner can write a result summary into it.
const (
	StatusPending = "Pending"
	StatusScanned = "Scanned: 0% Plagiarism"
	StatusGraded  = "Graded"
)

type User struct {
	ID           uuid.UUID `gorm:"type:text;primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	Email        string    `gorm:"not null;uniqueIndex" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         Role      `gorm:"type:text;not null;check:role IN ('student','professor')" json:"role"`
}

type Course struct {
	ID          uuid.UUID `gorm:"type:text;primaryKey" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	ProfessorID uuid.UUID `gorm:"type:text;not null;index" json:"professor_id"`
	Professor   *User     `gorm:"foreignKey:ProfessorID;constraint:OnDelete:RESTRICT" json:"professor,omitempty"`
}

type Assignment struct {
	ID          uuid.UUID `gorm:"type:text;primaryKey" json:"id"`
	CourseID    uuid.UUID `gorm:"type:text;not null;index" json:"course_id"`
	Course      *Course   `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"not null;default:''" json:"description"`
	Deadline    time.Time `gorm:"not null;index" json:"deadline"`
	MaxScore    int       `gorm:"not null;check:max_score > 0" json:"max_score"`
}

type Submission struct {
	ID uuid.UUID `gorm:"type:text;primaryKey" json:"id"`

	// (assignment_id, student_id) is unique: one submission per student per
	// assignment. The DB, not application code, is the source of truth.
	AssignmentID uuid.UUID   `gorm:"type:text;not null;uniqueIndex:ux_submission_student_assignment" json:"assignment_id"`
	Assignment   *Assignment `gorm:"foreignKey:AssignmentID;constraint:OnDelete:CASCADE" json:"assignment,omitempty"`
	StudentID    uuid.UUID   `gorm:"type:text;not null;uniqueIndex:ux_submission_student_assignment;index" json:"student_id"`
	Student      *User       `gorm:"foreignKey:StudentID;constraint:OnDelete:CASCADE" json:"student,omitempty"`

	FileURL     string    `gorm:"not null" json:"file_url"`
	Status      string    `gorm:"not null;default:Pending;index" json:"status"`
	Score       *int      `gorm:"check:score IS NULL OR score >= 0" json:"score"`
	SubmittedAt time.Time `gorm:"not null;index" json:"submitted_at"`
}

func (u *User) BeforeCreate(*gorm.DB) error       { return assignID(&u.ID) }
func (c *Course) BeforeCreate(*gorm.DB) error     { return assignID(&c.ID) }
func (a *Assignment) BeforeCreate(*gorm.DB) error { return assignID(&a.ID) }
func (s *Submission) BeforeCreate(*gorm.DB) error { return assignID(&s.ID) }

func assignID(id *uuid.UUID) error {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
	return nil
}

