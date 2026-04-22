package main

import (
    "log"
    "money-manager/config"
    "money-manager/database"
    "money-manager/routes"
)

func main() {
    // Load configuration
    config.LoadConfig()
    
    // Initialize database
    database.InitDB()
    defer database.DB.Close()
    
    // Setup router
    router := routes.SetupRouter()
    
    // Start server
    log.Println("Server starting on :8084")
    if err := router.Run(":8084"); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}