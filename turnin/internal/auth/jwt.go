package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"turnin/internal/models"
)

const issuer = "turnin"

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Name string      `json:"name"`
	Role models.Role `json:"role"`
	jwt.RegisteredClaims
}

func (c *Claims) UserID() (uuid.UUID, error) { return uuid.Parse(c.Subject) }

type Service struct {
	secret []byte
	ttl    time.Duration
}

func NewService(secret string, ttl time.Duration) *Service {
	return &Service{secret: []byte(secret), ttl: ttl}
}

func (s *Service) Issue(u *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		Name: u.Name,
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// Parse pins the algorithm (defends against alg=none / key-confusion), and
// requires both issuer and expiry.
func (s *Service) Parse(raw string) (*Claims, error) {
	var claims Claims
	tok, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !tok.Valid {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

