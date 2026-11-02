// Package apierr gives handlers a typed error vocabulary and a single place
// that turns errors into a consistent JSON envelope.
package apierr

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, msg string) *Error    { return New(fiber.StatusBadRequest, code, msg) }
func Unauthorized(code, msg string) *Error  { return New(fiber.StatusUnauthorized, code, msg) }
func Forbidden(code, msg string) *Error     { return New(fiber.StatusForbidden, code, msg) }
func NotFound(code, msg string) *Error      { return New(fiber.StatusNotFound, code, msg) }
func Conflict(code, msg string) *Error      { return New(fiber.StatusConflict, code, msg) }
func Unprocessable(code, msg string) *Error { return New(fiber.StatusUnprocessableEntity, code, msg) }

// Handler is installed as fiber.Config.ErrorHandler. Unknown errors are logged
// server-side and never leaked to clients.
func Handler(c *fiber.Ctx, err error) error {
	var ae *Error
	if !errors.As(err, &ae) {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			ae = New(fe.Code, "http_error", fe.Message)
		} else {
			log.Printf("[error] %s %s: %v", c.Method(), c.Path(), err)
			ae = New(fiber.StatusInternalServerError, "internal", "Internal server error")
		}
	}
	return c.Status(ae.Status).JSON(fiber.Map{
		"error": fiber.Map{"code": ae.Code, "message": ae.Message},
	})
}

