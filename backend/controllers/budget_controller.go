package controllers

import (
    "net/http"
    "strconv"
    "time"
    "money-manager/database"
    "money-manager/models"
    "github.com/gin-gonic/gin"
)

type BudgetWithSpending struct {
    models.Budget
    Spent      float64 `json:"spent"`
    Remaining  float64 `json:"remaining"`
    Percentage float64 `json:"percentage"`
}

func CreateBudget(c *gin.Context) {
    userID := c.GetInt("userID")
    var req models.BudgetRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    budgetUUID := models.GenerateUUID()
    query := `INSERT INTO budgets (uuid, user_id, category, amount, month, year) 
              VALUES (?, ?, ?, ?, ?, ?)
              ON DUPLICATE KEY UPDATE amount = ?`
    
    _, err := database.DB.Exec(query, budgetUUID, userID, req.Category, req.Amount, req.Month, req.Year, req.Amount)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create/update budget"})
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{"message": "Budget saved successfully"})
}

func GetBudgets(c *gin.Context) {
    userID := c.GetInt("userID")
    month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
    year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
    
    query := `SELECT b.uuid, b.category, b.amount, b.month, b.year,
              COALESCE(SUM(t.amount), 0) as spent
              FROM budgets b
              LEFT JOIN transactions t ON t.user_id = b.user_id 
                  AND t.category = b.category 
                  AND t.type = 'expense'
                  AND MONTH(t.transaction_date) = b.month 
                  AND YEAR(t.transaction_date) = b.year
              WHERE b.user_id = ? AND b.month = ? AND b.year = ?
              GROUP BY b.id`
    
    rows, err := database.DB.Query(query, userID, month, year)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch budgets"})
        return
    }
    defer rows.Close()
    
    var budgets []BudgetWithSpending
    for rows.Next() {
        var b BudgetWithSpending
        err := rows.Scan(&b.UUID, &b.Category, &b.Amount, &b.Month, &b.Year, &b.Spent)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning data"})
            return
        }
        b.Remaining = b.Amount - b.Spent
        if b.Amount > 0 {
            b.Percentage = (b.Spent / b.Amount) * 100
        }
        budgets = append(budgets, b)
    }
    
    c.JSON(http.StatusOK, gin.H{"budgets": budgets})
}

func DeleteBudget(c *gin.Context) {
    userID := c.GetInt("userID")
    budgetUUID := c.Param("uuid")
    
    result, err := database.DB.Exec("DELETE FROM budgets WHERE uuid = ? AND user_id = ?", budgetUUID, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete budget"})
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
}

func UpdateBudget(c *gin.Context) {
    userID := c.GetInt("userID")
    budgetUUID := c.Param("uuid")
    
    var req models.BudgetRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // First, delete the old budget entry
    result, err := database.DB.Exec(
        "DELETE FROM budgets WHERE uuid = ? AND user_id = ?",
        budgetUUID, userID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove old budget"})
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
        return
    }
    
    // Then create new budget with updated category
    newUUID := models.GenerateUUID()
    _, err = database.DB.Exec(
        "INSERT INTO budgets (uuid, user_id, category, amount, month, year) VALUES (?, ?, ?, ?, ?, ?)",
        newUUID, userID, req.Category, req.Amount, req.Month, req.Year,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update budget"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Budget updated successfully"})
}