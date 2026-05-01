package database

import (
    "database/sql"
    "fmt"
    "log"
    "money-manager/config"
    "strings"
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

        `CREATE TABLE IF NOT EXISTS accounts (
            id INT PRIMARY KEY AUTO_INCREMENT,
            uuid VARCHAR(36) UNIQUE NOT NULL,
            user_id INT NOT NULL,
            name VARCHAR(100) NOT NULL,
            type ENUM('cash','bank','ewallet') NOT NULL,
            currency_code CHAR(3) NOT NULL DEFAULT 'IDR',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            UNIQUE KEY unique_account_name (user_id, name),
            INDEX idx_user_accounts (user_id)
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

    ensureTransactionsSchema()
}

func ensureTransactionsSchema() {
    // Best-effort schema evolution without a migration tool.
    // Ignore "already exists" errors so upgrades are idempotent.
    alters := []string{
        `ALTER TABLE transactions ADD COLUMN account_id INT NULL`,
        `ALTER TABLE transactions ADD COLUMN currency_code CHAR(3) NULL`,
        `ALTER TABLE transactions ADD COLUMN is_transfer BOOLEAN NOT NULL DEFAULT FALSE`,
        `ALTER TABLE transactions ADD COLUMN transfer_uuid VARCHAR(36) NULL`,
        `ALTER TABLE transactions ADD INDEX idx_user_account_date (user_id, account_id, transaction_date)`,
        `ALTER TABLE transactions ADD INDEX idx_transfer_uuid (transfer_uuid)`,
        `ALTER TABLE transactions ADD CONSTRAINT fk_transactions_account FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL`,
    }

    for _, q := range alters {
        _, err := DB.Exec(q)
        if err == nil {
            continue
        }
        msg := err.Error()
        if strings.Contains(msg, "Duplicate column name") ||
            strings.Contains(msg, "Duplicate key name") ||
            strings.Contains(msg, "Duplicate foreign key constraint name") ||
            strings.Contains(msg, "already exists") ||
            strings.Contains(msg, "errno: 1061") || // duplicate key name
            strings.Contains(msg, "errno: 1060") { // duplicate column
            continue
        }
        log.Fatal("Failed to alter transactions schema:", err)
    }
}