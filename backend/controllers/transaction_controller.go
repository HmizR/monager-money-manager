package controllers

import (
    "database/sql"
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

    var accountID int
    var currencyCode string
    if req.AccountUUID != "" {
        accountID, currencyCode, err = getAccountByUUID(userID, req.AccountUUID)
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
            return
        }
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
            return
        }
    } else {
        accountID, _, currencyCode, err = getDefaultAccount(userID)
        if err == sql.ErrNoRows {
            c.JSON(http.StatusBadRequest, gin.H{"error": "No accounts found; create an account first"})
            return
        }
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
            return
        }
    }

    transactionUUID := models.GenerateUUID()
    query := `INSERT INTO transactions (uuid, user_id, account_id, currency_code, amount, type, category, description, transaction_date) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
    
    _, err = database.DB.Exec(query, transactionUUID, userID, accountID, currencyCode, req.Amount, req.Type, 
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
    accountUUID := c.Query("account_uuid")
    
    query := `SELECT t.uuid, a.uuid, t.currency_code, t.transfer_uuid, t.amount, t.type, t.category, t.description, t.transaction_date, t.created_at
              FROM transactions t
              LEFT JOIN accounts a ON a.id = t.account_id
              WHERE t.user_id = ?`
    args := []interface{}{userID}
    
    if startDate != "" {
        query += " AND t.transaction_date >= ?"
        args = append(args, startDate)
    }
    if endDate != "" {
        query += " AND t.transaction_date <= ?"
        args = append(args, endDate)
    }
    if category != "" {
        query += " AND t.category = ?"
        args = append(args, category)
    }
    if transactionType != "" {
        query += " AND t.type = ?"
        args = append(args, transactionType)
    }
    if accountUUID != "" {
        query += " AND a.uuid = ?"
        args = append(args, accountUUID)
    }
    
    query += " ORDER BY t.transaction_date DESC"
    
    rows, err := database.DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
        return
    }
    defer rows.Close()
    
    var transactions []models.Transaction
    rowIdx := 0
    for rows.Next() {
        var t models.Transaction
        var accountUUIDNS sql.NullString
        var currencyCodeNS sql.NullString
        var transferUUIDNS sql.NullString
        err := rows.Scan(&t.UUID, &accountUUIDNS, &currencyCodeNS, &transferUUIDNS, &t.Amount, &t.Type, &t.Category, &t.Description, &t.TransactionDate, &t.CreatedAt)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning data"})
            return
        }
        if accountUUIDNS.Valid {
            t.AccountUUID = accountUUIDNS.String
        } else {
            t.AccountUUID = ""
        }
        if currencyCodeNS.Valid {
            t.CurrencyCode = currencyCodeNS.String
        } else {
            t.CurrencyCode = ""
        }
        if transferUUIDNS.Valid {
            t.TransferUUID = transferUUIDNS.String
        } else {
            t.TransferUUID = ""
        }
        transactions = append(transactions, t)
        rowIdx++
    }
    if rerr := rows.Err(); rerr != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
        return
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

    // Optional: move transaction to another account
    var accountID any = nil
    var currencyCode any = nil
    if req.AccountUUID != "" {
        id, cur, err := getAccountByUUID(userID, req.AccountUUID)
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
            return
        }
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
            return
        }
        accountID = id
        currencyCode = cur
    }
    
    query := `UPDATE transactions
              SET amount = ?, type = ?, category = ?, description = ?, transaction_date = ?,
                  account_id = COALESCE(?, account_id),
                  currency_code = COALESCE(?, currency_code)
              WHERE uuid = ? AND user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL)`
    
    result, err := database.DB.Exec(query, req.Amount, req.Type, req.Category, req.Description, transactionDate, accountID, currencyCode, transactionUUID, userID)
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
    
    result, err := database.DB.Exec("DELETE FROM transactions WHERE uuid = ? AND user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL)", transactionUUID, userID)
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