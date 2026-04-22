package database

import (
    "database/sql"
    "fmt"
    "log"
    "money-manager/config"
    _ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB() {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        config.AppConfig.DBUser,
        config.AppConfig.DBPassword,
        config.AppConfig.DBHost,
        config.AppConfig.DBPort,
        config.AppConfig.DBName,
    )

    var err error
    DB, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    if err = DB.Ping(); err != nil {
        log.Fatal("Failed to ping database:", err)
    }

    createTables()
}

func createTables() {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS users (
            id INT PRIMARY KEY AUTO_INCREMENT,
            uuid VARCHAR(36) UNIQUE NOT NULL,
            username VARCHAR(50) UNIQUE NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )`,
        
        `CREATE TABLE IF NOT EXISTS transactions (
            id INT PRIMARY KEY AUTO_INCREMENT,
            uuid VARCHAR(36) UNIQUE NOT NULL,
            user_id INT NOT NULL,
            amount DECIMAL(15,2) NOT NULL,
            type ENUM('income', 'expense') NOT NULL,
            category VARCHAR(50) NOT NULL,
            description TEXT,
            transaction_date DATE NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            INDEX idx_user_date (user_id, transaction_date),
            INDEX idx_user_category (user_id, category)
        )`,
        
        `CREATE TABLE IF NOT EXISTS budgets (
            id INT PRIMARY KEY AUTO_INCREMENT,
            uuid VARCHAR(36) UNIQUE NOT NULL,
            user_id INT NOT NULL,
            category VARCHAR(50) NOT NULL,
            amount DECIMAL(15,2) NOT NULL,
            month INT NOT NULL,
            year INT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            UNIQUE KEY unique_budget (user_id, category, month, year)
        )`,
        
        `CREATE TABLE IF NOT EXISTS recommendations (
            id INT PRIMARY KEY AUTO_INCREMENT,
            user_id INT NOT NULL,
            recommendation_text TEXT NOT NULL,
            is_read BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        )`,
    }

    for _, query := range queries {
        _, err := DB.Exec(query)
        if err != nil {
            log.Fatal("Failed to create table:", err)
        }
    }
}