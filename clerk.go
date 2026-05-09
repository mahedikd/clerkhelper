package clerkhelper

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/labstack/echo/v4"
)

type ClerkUserData struct {
	UserID string            `json:"user_id"`
	Role   string            `json:"role"`
	Extra  map[string]string `json:"extra,omitempty"`
}

type Config struct {
	MetadataKeys []string
	CacheTTL     time.Duration
	CacheLimit   int64
}

type ctxKey struct{}

var (
	config      = Config{}
	userDataKey = ctxKey{}
)

func Init(cfg Config) {
	config = cfg
	userCache.ensureCache()
}

var sessionClaimsFromContext = clerk.SessionClaimsFromContext

func RequireAuth(allowedRoles []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := sessionClaimsFromContext(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			userData, err := userCache.GetUserData(c.Request().Context(), claims.Subject)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Failed to get user data"})
			}

			if len(allowedRoles) > 0 && !slices.Contains(allowedRoles, userData.Role) {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
			}

			ctx := context.WithValue(c.Request().Context(), userDataKey, userData)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func GetUserFromContext(c echo.Context) (*ClerkUserData, bool) {
	u, ok := c.Request().Context().Value(userDataKey).(*ClerkUserData)
	return u, ok
}

func GetUserData(ctx context.Context, userID string) (*ClerkUserData, error) {
	return userCache.GetUserData(ctx, userID)
}

func ValidateClerkToken(ctx context.Context, token string, roles []string) bool {
	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{Token: token})
	if err != nil {
		return false
	}

	u, err := userCache.GetUserData(ctx, claims.Subject)
	if err != nil {
		return false
	}

	if len(roles) > 0 {
		return slices.Contains(roles, u.Role)
	}
	return true
}
