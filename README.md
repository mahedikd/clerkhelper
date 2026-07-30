# Clerk Helper Go

A lightweight, high-performance Go package for easy Clerk authentication and authorization management in Echo-based applications.

## Features

- **Unified Middleware:** Simple `RequireAuth` middleware that supports role-based access control (RBAC).
- **Configurable Metadata:** Fetch any custom fields from Clerk's `public_metadata`.
- **High-Performance Caching:** Utilizes [Ristretto](https://github.com/dgraph-io/ristretto) for thread-safe, high-hit-ratio in-memory caching.
- **Concurrent Request Handling:** Uses `singleflight` to prevent "thundering herd" issues.
- **Easy Context Integration:** Access user metadata directly from the Echo context.
- **Customizable Cache:** Configure TTL and MaxCost for your specific needs.

## Installation

```bash
go get github.com/mahedikd/clerkhelper
```

## Setup & Initialization

Before using the middleware, you must initialize the package with your desired configuration.

### 1. Configure and Init

Call `clerkhelper.Init` in your `main.go`. The `role` field is always extracted by default.

```go
import "github.com/mahedikd/clerkhelper"
import "time"

func main() {
    // Initialize with custom settings
    clerkhelper.Init(clerkhelper.Config{
        MetadataKeys: []string{"t_id", "org_id"},
        CacheTTL:     10 * time.Minute, // Optional, default is 5 mins
        CacheLimit:   100_000,          // Optional, default is 100,000 users
    })

    // ... rest of setup
}
```

### 2. Full Initialization Example

```go
import (
    "context"
    "time"
    clerkSDK "github.com/clerk/clerk-sdk-go/v2"
    "github.com/clerk/clerk-sdk-go/v2/jwks"
    "github.com/mahedikd/clerkhelper"
)

func main() {
    // 1. Init Helper Config
    clerkhelper.Init(clerkhelper.Config{
        MetadataKeys: []string{"t_id"},
    })

    // 2. Set Clerk Key
    clerkSDK.SetKey("your_secret_key")

    // 3. Optional: Preload JWKS for performance
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _, _ = jwks.Get(ctx, &jwks.GetParams{})

    // 4. Recommended: Start cache cleanup
    stop := clerkhelper.StartCacheCleanup()
    defer stop()
}
```

## Middleware Usage

### Applying to Routes

```go
import (
    "github.com/labstack/echo/v4"
    clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
    "github.com/mahedikd/clerkhelper"
)

func Register(e *echo.Echo) {
    api := e.Group("/api")

    // Mandatory: Verify Clerk session from Authorization header
    api.Use(echo.WrapMiddleware(clerkhttp.WithHeaderAuthorization()))

    // RBAC: Allow only specific roles (e.g. ADMIN)
    admin := api.Group("/admin", clerkhelper.RequireAuth([]string{"ADMIN"}))
    admin.GET("/stats", handleStats)

    // Auth only: Allow any valid session (no role check)
    api.GET("/profile", handleProfile, clerkhelper.RequireAuth(nil))
}
```

### Accessing User Data in Handlers

```go
func handleProfile(c echo.Context) error {
    user, ok := clerkhelper.GetUserFromContext(c)
    if !ok {
        return c.JSON(401, "unauthorized")
    }

    // Role is extracted by default
    role := user.Role

    // Custom keys are accessed via the Extra map
    tenantID := user.Extra["t_id"]

    return c.JSON(200, user)
}
```

## Available Methods

| Method                 | Signature                               | Description                                            |
| :--------------------- | :-------------------------------------- | :----------------------------------------------------- |
| **Init**               | `Init(Config)`                          | Initializes the package with metadata keys to extract. |
| **RequireAuth**        | `RequireAuth([]string)`                 | Echo middleware for RBAC and session verification.     |
| **GetUserFromContext** | `GetUserFromContext(echo.Context)`      | Retrieves `ClerkUserData` from the Echo context.       |
| **GetUserData**        | `GetUserData(context.Context, string)`  | Fetches cached user data directly using a User ID.     |
| **ValidateClerkToken** | `ValidateClerkToken(ctx, token, roles)` | Manual token verification and role checking.           |
| **InvalidateCache**    | `InvalidateCache(string)`               | Evicts a user from the in-memory cache (next fetch hits Clerk API). Useful after metadata updates. |
| **StartCacheCleanup**  | `StartCacheCleanup()`                   | Returns a function to gracefully close the cache.      |

## Data Structures

### ClerkUserData

```go
type ClerkUserData struct {
    UserID string            // The Clerk User ID
    Role   string            // Extracted from "role" in public_metadata
    Extra  map[string]string // Custom fields defined during Init()
}
```

### Config

```go
type Config struct {
    MetadataKeys []string      // List of keys to pull from public_metadata
    CacheTTL     time.Duration // Time-to-live for cached users
    CacheLimit   int64         // Maximum number of users to cache
}
```

## License

MIT License - see the [LICENSE](LICENSE) file for details.
