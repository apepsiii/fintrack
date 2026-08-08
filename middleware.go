package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT token and adds user info to context
func AuthMiddleware() gin.HandlerFunc {
	authService := NewAuthService()

	return func(c *gin.Context) {
		// Try to get token from cookie first
		tokenString, err := c.Cookie("auth_token")
		
		// If no cookie, try Authorization header
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		}

		// Validate token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			isSecure := os.Getenv("GIN_MODE") == "release"
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     "auth_token",
				Value:    "",
				MaxAge:   -1,
				Path:     "/",
				HttpOnly: true,
				Secure:   isSecure,
				SameSite: http.SameSiteStrictMode,
			})
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)

		// Role separation: admin tidak bisa akses route user
		var isAdmin int
		db.QueryRow("SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?", claims.UserID).Scan(&isAdmin)
		c.Set("is_admin", isAdmin == 1)

		if isAdmin == 1 {
			path := c.Request.URL.Path
			// Admin hanya boleh akses /backoffice/* dan /logout
			if !strings.HasPrefix(path, "/backoffice") && path != "/logout" {
				c.Redirect(http.StatusFound, "/backoffice/")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// OptionalAuthMiddleware validates token if present but doesn't require it
func OptionalAuthMiddleware() gin.HandlerFunc {
	authService := NewAuthService()

	return func(c *gin.Context) {
		tokenString, err := c.Cookie("auth_token")
		
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}
		}

		if tokenString != "" {
			claims, err := authService.ValidateToken(tokenString)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Name)
			}
		}

		c.Next()
	}
}

// GetCurrentUser retrieves current user from context
func GetCurrentUser(c *gin.Context) (*User, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, nil
	}

	return GetUserByID(userID.(int))
}

// GetCurrentUserID gets just the user ID from context
func GetCurrentUserID(c *gin.Context) int {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(int)
}
