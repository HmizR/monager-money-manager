package routes

import (
    "money-manager/controllers"
    "money-manager/middleware"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
    router := gin.Default()
    
    // CORS configuration
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8085"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
    }))
    
    // Public routes
    auth := router.Group("/api/auth")
    {
        auth.POST("/register", controllers.Register)
        auth.POST("/login", controllers.Login)
    }
    
    // Protected routes
    api := router.Group("/api")
    api.Use(middleware.AuthMiddleware())
    {
        // Transaction routes
        api.POST("/transactions", controllers.CreateTransaction)
        api.GET("/transactions", controllers.GetTransactions)
        api.PUT("/transactions/:uuid", controllers.UpdateTransaction)
        api.DELETE("/transactions/:uuid", controllers.DeleteTransaction)
        
        // Statistics routes
        api.GET("/statistics", controllers.GetStatistics)
        api.GET("/charts", controllers.GetChartData)

		// Dalam protected routes section, setelah routes yang sudah ada
		api.GET("/balance/current", controllers.GetCurrentBalance)
		api.GET("/balance/trend", controllers.GetBalanceTrend)
		api.GET("/balance/forecast", controllers.GetBalanceForecast)
		api.GET("/balance/projection", controllers.GetMonthlyProjection)
        
        // Budget routes
        api.POST("/budgets", controllers.CreateBudget)
		api.PUT("/budgets/:uuid", controllers.UpdateBudget)
        api.GET("/budgets", controllers.GetBudgets)
        api.DELETE("/budgets/:uuid", controllers.DeleteBudget)
        
        // Recommendation routes
        api.POST("/recommendations/generate", controllers.GenerateRecommendations)
        api.GET("/recommendations", controllers.GetRecommendations)
        api.PUT("/recommendations/:id/read", controllers.MarkRecommendationAsRead)

		// Dalam protected routes section
		api.GET("/categories", controllers.GetCategories)
        
        // Export/Import routes
        api.GET("/export", controllers.ExportTransactions)
        api.POST("/import", controllers.ImportTransactions)
    }
    
    return router
}