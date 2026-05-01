package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID           int       `json:"id"`
    UUID         string    `json:"uuid"`
    Username     string    `json:"username"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type Transaction struct {
    ID              int       `json:"id"`
    UUID            string    `json:"uuid"`
    UserID          int       `json:"user_id"`
    AccountUUID     string    `json:"account_uuid,omitempty"`
    CurrencyCode    string    `json:"currency_code,omitempty"`
    TransferUUID    string    `json:"transfer_uuid,omitempty"`
    Amount          float64   `json:"amount"`
    Type            string    `json:"type"`
    Category        string    `json:"category"`
    Description     string    `json:"description"`
    TransactionDate time.Time `json:"transaction_date"`
    CreatedAt       time.Time `json:"created_at"`
}

type Account struct {
    ID           int       `json:"-"`
    UUID         string    `json:"uuid"`
    UserID       int       `json:"-"`
    Name         string    `json:"name"`
    Type         string    `json:"type"`
    CurrencyCode string    `json:"currency_code"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type Budget struct {
    ID       int     `json:"id"`
    UUID     string  `json:"uuid"`
    UserID   int     `json:"user_id"`
    Category string  `json:"category"`
    Amount   float64 `json:"amount"`
    Month    int     `json:"month"`
    Year     int     `json:"year"`
}

type Recommendation struct {
    ID                 int       `json:"id"`
    UserID             int       `json:"user_id"`
    RecommendationText string    `json:"recommendation_text"`
    IsRead             bool      `json:"is_read"`
    CreatedAt          time.Time `json:"created_at"`
}

type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
    Username string `json:"username" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}

type TransactionRequest struct {
    AccountUUID     string  `json:"account_uuid"`
    Amount          float64 `json:"amount" binding:"required"`
    Type            string  `json:"type" binding:"required,oneof=income expense"`
    Category        string  `json:"category" binding:"required"`
    Description     string  `json:"description"`
    TransactionDate string  `json:"transaction_date" binding:"required"`
}

type AccountRequest struct {
    Name         string `json:"name" binding:"required"`
    Type         string `json:"type" binding:"required,oneof=cash bank ewallet"`
    CurrencyCode string `json:"currency_code" binding:"required,len=3"`
}

type TransferRequest struct {
    FromAccountUUID string  `json:"from_account_uuid" binding:"required"`
    ToAccountUUID   string  `json:"to_account_uuid" binding:"required"`
    Amount          float64 `json:"amount" binding:"required,gt=0"`
    Description     string  `json:"description"`
    TransactionDate string  `json:"transaction_date" binding:"required"`
}

type BudgetRequest struct {
    Category string  `json:"category" binding:"required"`
    Amount   float64 `json:"amount" binding:"required"`
    Month    int     `json:"month" binding:"required,min=1,max=12"`
    Year     int     `json:"year" binding:"required,min=2020,max=2030"`
}

func GenerateUUID() string {
    return uuid.New().String()
}