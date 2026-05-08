package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Sriniketh24/rollout/internal/models"
)

var (
	ErrInvalidToken  = errors.New("invalid or expired token")
	ErrInvalidAPIKey = errors.New("invalid API key")
	ErrForbidden     = errors.New("insufficient permissions")
)

type contextKey string

const userContextKey contextKey = "user"

type UserStore interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByAPIKey(ctx context.Context, apiKey string) (*models.User, error)
	GetUserProjectRole(ctx context.Context, userID, projectID string) (models.Role, error)
}

type Auth struct {
	store      UserStore
	jwtSecret  []byte
	tokenExpiry time.Duration
}

func New(store UserStore, jwtSecret string, tokenExpiry time.Duration) *Auth {
	return &Auth{
		store:       store,
		jwtSecret:   []byte(jwtSecret),
		tokenExpiry: tokenExpiry,
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (a *Auth) GenerateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *Auth) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return a.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (a *Auth) AuthenticateJWT(ctx context.Context, tokenStr string) (*models.User, error) {
	claims, err := a.ValidateToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return a.store.GetUserByID(ctx, claims.UserID)
}

func (a *Auth) AuthenticateAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	user, err := a.store.GetUserByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	return user, nil
}

func (a *Auth) Authorize(ctx context.Context, userID, projectID string, required models.Role) error {
	role, err := a.store.GetUserProjectRole(ctx, userID, projectID)
	if err != nil {
		return ErrForbidden
	}
	if !hasPermission(role, required) {
		return ErrForbidden
	}
	return nil
}

func hasPermission(actual, required models.Role) bool {
	levels := map[models.Role]int{
		models.RoleViewer: 1,
		models.RoleEditor: 2,
		models.RoleAdmin:  3,
	}
	return levels[actual] >= levels[required]
}

func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "rol_" + hex.EncodeToString(bytes), nil
}

func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func SetUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func GetUser(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok
}
