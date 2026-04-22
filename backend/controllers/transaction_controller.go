package controllers

import (
    "net/http"
    "time"
    "money-manager/database"
    "money-manager/models"
    "github.com/gin-gonic/gin"
)

func CreateTransaction(c *gin.Context) {
    userID := c.GetInt("userID")
    var req models.TransactionRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
        return
    }

    transactionUUID := models.GenerateUUID()
    query := `INSERT INTO transactions (uuid, user_id, amount, type, category, description, transaction_date) 
              VALUES (?, ?, ?, ?, ?, ?, ?)`
    
    _, err = database.DB.Exec(query, transactionUUID, userID, req.Amount, req.Type, 
        req.Category, req.Description, transactionDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Transaction created successfully"})
}

func GetTransactions(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Parse query parameters for filtering
    startDate := c.Query("start_date")
    endDate := c.Query("end_date")
    category := c.Query("category")
    transactionType := c.Query("type")
    
    query := "SELECT uuid, amount, type, category, description, transaction_date, created_at FROM transactions WHERE user_id = ?"
    args := []interface{}{userID}
    
    if startDate != "" {
        query += " AND transaction_date >= ?"
        args = append(args, startDate)
    }
    if endDate != "" {
        query += " AND transaction_date <= ?"
        args = append(args, endDate)
    }
    if category != "" {
        query += " AND category = ?"
        args = append(args, category)
    }
    if transactionType != "" {
        query += " AND type = ?"
        args = append(args, transactionType)
    }
    
    query += " ORDER BY transaction_date DESC"
    
    rows, err := database.DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
        return
    }
    defer rows.Close()
    
    var transactions []models.Transaction
    for rows.Next() {
        var t models.Transaction
        err := rows.Scan(&t.UUID, &t.Amount, &t.Type, &t.Category, &t.Description, &t.TransactionDate, &t.CreatedAt)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning data"})
            return
        }
        transactions = append(transactions, t)
    }
    
    c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

func UpdateTransaction(c *gin.Context) {
    userID := c.GetInt("userID")
    transactionUUID := c.Param("uuid")
    
    var req models.TransactionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
        return
    }
    
    query := `UPDATE transactions SET amount = ?, type = ?, category = ?, description = ?, transaction_date = ? 
              WHERE uuid = ? AND user_id = ?`
    
    result, err := database.DB.Exec(query, req.Amount, req.Type, req.Category, req.Description, transactionDate, transactionUUID, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Transaction updated successfully"})
}

func DeleteTransaction(c *gin.Context) {
    userID := c.GetInt("userID")
    transactionUUID := c.Param("uuid")
    
    result, err := database.DB.Exec("DELETE FROM transactions WHERE uuid = ? AND user_id = ?", transactionUUID, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction"})
        return
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
}