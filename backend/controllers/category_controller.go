package controllers

import (
    "net/http"
    "money-manager/database"
    "github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Get unique categories from user's transactions
    query := `SELECT DISTINCT category FROM transactions WHERE user_id = ? ORDER BY category`
    
    rows, err := database.DB.Query(query, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
        return
    }
    defer rows.Close()
    
    categories := []string{}
    for rows.Next() {
        var category string
        if err := rows.Scan(&category); err == nil {
            categories = append(categories, category)
        }
    }
    
    // Add default categories if none exist
    if len(categories) == 0 {
        defaultCategories := []string{
            "Food & Dining",
            "Transportation",
            "Shopping",
            "Entertainment",
            "Bills & Utilities",
            "Healthcare",
            "Education",
            "Rent",
            "Salary",
            "Investment",
            "Other",
        }
        categories = defaultCategories
    }
    
    c.JSON(http.StatusOK, gin.H{"categories": categories})
}