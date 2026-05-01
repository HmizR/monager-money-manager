package controllers

import (
    "database/sql"
    "net/http"
    "time"

    "money-manager/database"
    "money-manager/models"

    "github.com/gin-gonic/gin"
)

func CreateTransfer(c *gin.Context) {
    userID := c.GetInt("userID")
    var req models.TransferRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.FromAccountUUID == req.ToAccountUUID {
        c.JSON(http.StatusBadRequest, gin.H{"error": "from_account_uuid and to_account_uuid must be different"})
        return
    }

    txDate, err := time.Parse("2006-01-02", req.TransactionDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
        return
    }

    fromID, fromCurrency, err := getAccountByUUID(userID, req.FromAccountUUID)
    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "From account not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }

    toID, toCurrency, err := getAccountByUUID(userID, req.ToAccountUUID)
    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "To account not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }

    if fromCurrency != toCurrency {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Currency mismatch; multi-currency transfer not implemented yet"})
        return
    }

    transferUUID := models.GenerateUUID()

    dbtx, err := database.DB.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
        return
    }
    defer dbtx.Rollback()

    // Expense leg (from account)
    _, err = dbtx.Exec(
        `INSERT INTO transactions (uuid, user_id, account_id, currency_code, amount, type, category, description, transaction_date, is_transfer, transfer_uuid)
         VALUES (?, ?, ?, ?, ?, 'expense', 'transfer', ?, ?, TRUE, ?)`,
        models.GenerateUUID(), userID, fromID, fromCurrency, req.Amount, req.Description, txDate, transferUUID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer (debit)"})
        return
    }

    // Income leg (to account)
    _, err = dbtx.Exec(
        `INSERT INTO transactions (uuid, user_id, account_id, currency_code, amount, type, category, description, transaction_date, is_transfer, transfer_uuid)
         VALUES (?, ?, ?, ?, ?, 'income', 'transfer', ?, ?, TRUE, ?)`,
        models.GenerateUUID(), userID, toID, toCurrency, req.Amount, req.Description, txDate, transferUUID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer (credit)"})
        return
    }

    if err := dbtx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transfer"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"transfer_uuid": transferUUID})
}

func DeleteTransfer(c *gin.Context) {
    userID := c.GetInt("userID")
    transferUUID := c.Param("transfer_uuid")
    if transferUUID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "transfer_uuid is required"})
        return
    }

    res, err := database.DB.Exec(
        `DELETE FROM transactions
         WHERE user_id = ? AND transfer_uuid = ? AND is_transfer = TRUE`,
        userID, transferUUID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transfer"})
        return
    }

    rows, _ := res.RowsAffected()
    if rows == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Transfer deleted", "deleted_rows": rows})
}

