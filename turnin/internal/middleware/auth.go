package middleware

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"

	"turnin/internal/apierr"
	"turnin/internal/auth"
	"turnin/internal/models"
)

const claimsKey = "claims"

func Authenticate(svc *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scheme, token, ok := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			return apierr.Unauthorized("missing_token", "Authorization: Bearer <token> required")
		}
		claims, err := svc.Parse(strings.TrimSpace(token))
		if err != nil {
			return apierr.Unauthorized("invalid_token", "Token is invalid or expired")
		}
		c.Locals(claimsKey, claims)
		return c.Next()
	}
}

// RequireRole is the RBAC gate. It must be chained after Authenticate.
func RequireRole(roles ...models.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := ClaimsFrom(c)
		if claims == nil || !slices.Contains(roles, claims.Role) {
			return apierr.Forbidden("insufficient_role", "Your role is not permitted to perform this action")
		}
		return c.Next()
	}
}

func ClaimsFrom(c *fiber.Ctx) *auth.Claims {
	cl, _ := c.Locals(claimsKey).(*auth.Claims)
	return cl
}
