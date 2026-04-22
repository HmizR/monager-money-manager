package middleware

import (
    "net/http"
    "strings"
    "money-manager/utils"
    "github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        var tokenString string
        
        // Cek dari header Authorization dulu
        authHeader := c.GetHeader("Authorization")
        if authHeader != "" {
            parts := strings.Split(authHeader, " ")
            if len(parts) == 2 && parts[0] == "Bearer" {
                tokenString = parts[1]
            }
        }
        
        // Jika tidak ada di header, cek dari query parameter
        if tokenString == "" {
            tokenString = c.Query("token")
        }
        
        // Jika masih tidak ada, return unauthorized
        if tokenString == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header or token"})
            c.Abort()
            return
        }
        
        claims, err := utils.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        c.Set("userID", claims.UserID)
        c.Set("username", claims.Username)
        c.Next()
    }
}