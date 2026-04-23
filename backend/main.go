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
	serverAddr := ":" + config.AppConfig.ServerPort
    log.Printf("Server starting on %s", serverAddr)
    if err := router.Run(serverAddr); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}