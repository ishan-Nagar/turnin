package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"turnin/internal/apierr"
	"turnin/internal/auth"
	"turnin/internal/middleware"
	"turnin/internal/models"
)

type AuthHandler struct {
	db   *gorm.DB
	auth *auth.Service
}

func NewAuthHandler(db *gorm.DB, svc *auth.Service) *AuthHandler {
	return &AuthHandler{db: db, auth: svc}
}

// dummyHash lets us burn the same bcrypt time for unknown emails, so response
// latency doesn't reveal which accounts exist.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcrypt.DefaultCost)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest("invalid_body", "Request body must be valid JSON")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		return apierr.BadRequest("missing_fields", "Email and password are required")
	}

	var user models.User
	lookupErr := h.db.WithContext(c.UserContext()).Where("email = ?", email).First(&user).Error
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return lookupErr
	}

	hash := dummyHash
	if lookupErr == nil {
		hash = []byte(user.PasswordHash)
	}
	pwErr := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	if lookupErr != nil || pwErr != nil {
		return apierr.Unauthorized("invalid_credentials", "Invalid email or password")
	}

	token, err := h.auth.Issue(&user)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"token": token, "user": user}})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	uid, err := middleware.ClaimsFrom(c).UserID()
	if err != nil {
		return apierr.Unauthorized("invalid_token", "Malformed subject claim")
	}
	var user models.User
	if err := h.db.WithContext(c.UserContext()).First(&user, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierr.Unauthorized("user_gone", "Account no longer exists")
		}
		return err
	}
	return c.JSON(fiber.Map{"data": user})
}

