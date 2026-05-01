package controllers

import (
    "database/sql"
    "net/http"
    "strings"

    "money-manager/database"
    "money-manager/models"

    "github.com/gin-gonic/gin"
)

func CreateAccount(c *gin.Context) {
    userID := c.GetInt("userID")
    var req models.AccountRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    currency := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
    accountUUID := models.GenerateUUID()

    _, err := database.DB.Exec(
        `INSERT INTO accounts (uuid, user_id, name, type, currency_code) VALUES (?, ?, ?, ?, ?)`,
        accountUUID, userID, req.Name, req.Type, currency,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"uuid": accountUUID})
}

func GetAccounts(c *gin.Context) {
    userID := c.GetInt("userID")

    rows, err := database.DB.Query(
        `SELECT uuid, name, type, currency_code, created_at, updated_at
         FROM accounts
         WHERE user_id = ?
         ORDER BY created_at ASC`,
        userID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch accounts"})
        return
    }
    defer rows.Close()

    accounts := []models.Account{}
    for rows.Next() {
        var a models.Account
        if err := rows.Scan(&a.UUID, &a.Name, &a.Type, &a.CurrencyCode, &a.CreatedAt, &a.UpdatedAt); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning data"})
            return
        }
        accounts = append(accounts, a)
    }

    c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

func DeleteAccount(c *gin.Context) {
    userID := c.GetInt("userID")
    accountUUID := c.Param("uuid")

    // Prevent deletion if account has any transactions.
    var count int
    err := database.DB.QueryRow(
        `SELECT COUNT(1)
         FROM transactions t
         JOIN accounts a ON a.id = t.account_id
         WHERE a.user_id = ? AND a.uuid = ?`,
        userID, accountUUID,
    ).Scan(&count)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }
    if count > 0 {
        c.JSON(http.StatusConflict, gin.H{"error": "Account has transactions; cannot delete"})
        return
    }

    res, err := database.DB.Exec(`DELETE FROM accounts WHERE user_id = ? AND uuid = ?`, userID, accountUUID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
        return
    }
    rows, _ := res.RowsAffected()
    if rows == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

func getAccountByUUID(userID int, accountUUID string) (id int, currencyCode string, err error) {
    err = database.DB.QueryRow(
        `SELECT id, currency_code FROM accounts WHERE user_id = ? AND uuid = ?`,
        userID, accountUUID,
    ).Scan(&id, &currencyCode)
    if err == sql.ErrNoRows {
        return 0, "", sql.ErrNoRows
    }
    return id, currencyCode, err
}

func getDefaultAccount(userID int) (id int, uuid string, currencyCode string, err error) {
    err = database.DB.QueryRow(
        `SELECT id, uuid, currency_code FROM accounts WHERE user_id = ? ORDER BY created_at ASC LIMIT 1`,
        userID,
    ).Scan(&id, &uuid, &currencyCode)
    return id, uuid, currencyCode, err
}

