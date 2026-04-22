package controllers

import (
    "database/sql"
    "net/http"
    "money-manager/database"
    "money-manager/models"
    "money-manager/utils"
    "github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
    var req models.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Check if user exists
    var exists bool
    query := "SELECT EXISTS(SELECT 1 FROM users WHERE username = ? OR email = ?)"
    err := database.DB.QueryRow(query, req.Username, req.Email).Scan(&exists)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }
    if exists {
        c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
        return
    }

    // Hash password
    hashedPassword, err := utils.HashPassword(req.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
        return
    }

    // Create user
    userUUID := models.GenerateUUID()
    result, err := database.DB.Exec(
        "INSERT INTO users (uuid, username, email, password_hash) VALUES (?, ?, ?, ?)",
        userUUID, req.Username, req.Email, hashedPassword,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
        return
    }

    userID, _ := result.LastInsertId()
    token, err := utils.GenerateToken(int(userID), req.Username)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "User registered successfully",
        "token":   token,
        "user": gin.H{
            "id":       userID,
            "username": req.Username,
            "email":    req.Email,
        },
    })
}

func Login(c *gin.Context) {
    var req models.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    query := "SELECT id, username, email, password_hash FROM users WHERE username = ?"
    err := database.DB.QueryRow(query, req.Username).Scan(
        &user.ID, &user.Username, &user.Email, &user.PasswordHash,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        }
        return
    }

    if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }

    token, err := utils.GenerateToken(user.ID, user.Username)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Login successful",
        "token":   token,
        "user": gin.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
        },
    })
}