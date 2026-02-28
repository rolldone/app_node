package auth

import (
	"fmt"

	authjwt "go_framework/internal/auth"

	"github.com/gin-gonic/gin"
)

// AdminClaimsMiddleware parses the Authorization header (Bearer token), extracts
// admin claims and injects them into the Gin context as `admin_id` and
// `admin_level`. For backward compatibility it also sets `user_id`.
// Parse errors are logged at debug level without revealing token contents.
func AdminClaimsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth != "" {
			var token string
			if n, _ := fmt.Sscanf(auth, "Bearer %s", &token); n == 1 {
				if claims, err := authjwt.ParseAccessTokenClaims(token); err == nil {
					// canonical keys
					c.Set("admin_id", claims.AdminID)
					c.Set("admin_level", claims.Level)
					// backward compatibility
					c.Set("user_id", claims.AdminID)
				} else {
					// debug-only log
					if gin.IsDebugging() {
						fmt.Printf("[debug] auth: failed to parse access token: %v\n", err)
					}
				}
			}
		}
		c.Next()
	}
}

// MemberClaimsMiddleware extracts customer/member claims and injects them as
// `customer_id` and `user_id` for compatibility; also sets `user_role` if present.
func MemberClaimsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth != "" {
			var token string
			if n, _ := fmt.Sscanf(auth, "Bearer %s", &token); n == 1 {
				if claims, err := authjwt.ParseAccessTokenClaims(token); err == nil {
					// claims.Level may indicate "customer" or roles
					c.Set("customer_id", claims.AdminID)
					c.Set("user_id", claims.AdminID)
					c.Set("user_role", claims.Level)
				} else {
					if gin.IsDebugging() {
						fmt.Printf("[debug] auth: failed to parse access token (member): %v\n", err)
					}
				}
			}
		}
		c.Next()
	}
}
