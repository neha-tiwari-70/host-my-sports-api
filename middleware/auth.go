package middleware

import (
	"fmt"
	"net/http"
	"sports-events-api/crypto"
	"sports-events-api/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthenticationMiddleware checks if the user has a valid JWT token
func AuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			fmt.Println("Authorization header is missing")
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "msg": "Unauthenticated", "data": ""})
			c.Abort()
			return
		}

		// The token should be prefixed with "Bearer "
		tokenParts := strings.Split(tokenString, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			fmt.Println("Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "msg": "Invalid Token", "data": ""})
			c.Abort()
			return
		}

		// Extract the token part
		tokenString = tokenParts[1]

		// Verify the token and get claims
		claims, err := utils.VerifyToken(tokenString)
		if err != nil {
			fmt.Println("Error verifying token:", err)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "msg": "Invalid Token", "data": ""})
			c.Abort()
			return
		}

		// Type assert user_id to string
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			fmt.Println("user_id is not a string:", claims["user_id"])
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "msg": "Invalid Token", "data": ""})
			c.Abort()
			return
		}

		decId, err := crypto.NDecrypt(userIDStr)
		if err != nil {
			fmt.Println("Error decrypting user id:", err)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "msg": "Invalid Token", "data": err})
			c.Abort()
			return
		}
		// Token verified successfully, set the user information in context
		// fmt.Println("Token verified, claims:", claims)

		// Store user ID or other relevant data in context
		c.Set("user_id", decId) // Ensure "user_id" key matches what you store in your token claims
		// c.Set("userEmail", claims["email"])
		// Optionally set userId if used
		// c.Set("userId", claims.UserId)
		// Continue processing the request
		c.Next()
	}
}
