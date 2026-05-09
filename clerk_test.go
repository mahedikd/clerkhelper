package clerkhelper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRequireAuth(t *testing.T) {
	e := echo.New()

	userID := "user_123"
	userData := &ClerkUserData{
		UserID: userID,
		Role:   "ADMIN",
		Extra:  map[string]string{"t_id": "tenant_123"},
	}

	userCache.ensureCache()
	userCache.cache.Set(userID, userData, 1)
	time.Sleep(10 * time.Millisecond)

	origSessionClaimsFromContext := sessionClaimsFromContext
	defer func() { sessionClaimsFromContext = origSessionClaimsFromContext }()

	t.Run("Authorized ADMIN", func(t *testing.T) {
		handler := RequireAuth([]string{"ADMIN"})(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		sessionClaimsFromContext = func(ctx context.Context) (*clerk.SessionClaims, bool) {
			claims := &clerk.SessionClaims{}
			claims.Subject = userID
			return claims, true
		}

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		u, ok := GetUserFromContext(c)
		assert.True(t, ok)
		assert.Equal(t, userData, u)
		assert.Equal(t, "tenant_123", u.Extra["t_id"])
	})

	t.Run("Forbidden Wrong Role", func(t *testing.T) {
		handler := RequireAuth([]string{"OWNER"})(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		sessionClaimsFromContext = func(ctx context.Context) (*clerk.SessionClaims, bool) {
			claims := &clerk.SessionClaims{}
			claims.Subject = userID
			return claims, true
		}

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestGetUserFromContext(t *testing.T) {
	e := echo.New()

	t.Run("User data present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expected := &ClerkUserData{UserID: "u1", Role: "ADMIN"}
		ctx := context.WithValue(req.Context(), userDataKey, expected)
		c.SetRequest(req.WithContext(ctx))

		user, ok := GetUserFromContext(c)
		assert.True(t, ok)
		assert.Equal(t, expected, user)
	})
}
