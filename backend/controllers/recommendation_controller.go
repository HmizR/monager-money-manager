package controllers

import (
    "fmt"
    "net/http"
    "time"
    "money-manager/database"
    "github.com/gin-gonic/gin"
)

func GenerateRecommendations(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Analyze spending patterns and generate recommendations
    recommendations := analyzeSpending(userID)
    
    // Save recommendations to database
    for _, rec := range recommendations {
        _, err := database.DB.Exec(
            "INSERT INTO recommendations (user_id, recommendation_text) VALUES (?, ?)",
            userID, rec,
        )
        if err != nil {
            fmt.Printf("Error saving recommendation: %v\n", err)
        }
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Recommendations generated", "recommendations": recommendations})
}

func GetRecommendations(c *gin.Context) {
    userID := c.GetInt("userID")
    
    rows, err := database.DB.Query(
        "SELECT id, recommendation_text, is_read, created_at FROM recommendations WHERE user_id = ? ORDER BY created_at DESC LIMIT 20",
        userID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recommendations"})
        return
    }
    defer rows.Close()
    
    var recommendations []map[string]interface{}
    for rows.Next() {
        var id int
        var text string
        var isRead bool
        var createdAt time.Time
        
        if err := rows.Scan(&id, &text, &isRead, &createdAt); err == nil {
            recommendations = append(recommendations, map[string]interface{}{
                "id":             id,
                "recommendation": text,
                "is_read":        isRead,
                "created_at":     createdAt,
            })
        }
    }
    
    c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

func MarkRecommendationAsRead(c *gin.Context) {
    userID := c.GetInt("userID")
    recID := c.Param("id")
    
    _, err := database.DB.Exec(
        "UPDATE recommendations SET is_read = TRUE WHERE id = ? AND user_id = ?",
        recID, userID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update recommendation"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Recommendation marked as read"})
}

func analyzeSpending(userID int) []string {
    var recommendations []string
    
    // Get total spending by category for current month
    currentMonth := time.Now().Month()
    currentYear := time.Now().Year()
    
    query := `SELECT category, SUM(amount) as total
              FROM transactions
              WHERE user_id = ? AND type = 'expense' 
                AND MONTH(transaction_date) = ? AND YEAR(transaction_date) = ?
              GROUP BY category`
    
    rows, err := database.DB.Query(query, userID, currentMonth, currentYear)
    if err != nil {
        return recommendations
    }
    defer rows.Close()
    
    categorySpending := make(map[string]float64)
    totalSpending := 0.0
    
    for rows.Next() {
        var category string
        var amount float64
        if err := rows.Scan(&category, &amount); err == nil {
            categorySpending[category] = amount
            totalSpending += amount
        }
    }
    
    // Generate recommendations based on spending patterns
    for category, amount := range categorySpending {
        if totalSpending > 0 {
            percentage := (amount / totalSpending) * 100
            if percentage > 40 && category != "Rent" && category != "Mortgage" {
                recommendations = append(recommendations,
                    fmt.Sprintf("Anda menghabiskan %.1f%% dari total pengeluaran untuk %s. Pertimbangkan untuk mengurangi pengeluaran di kategori ini.", percentage, category))
            }
        }
    }
    
    // Check if there are any budgets exceeded
    budgetQuery := `SELECT b.category, b.amount, COALESCE(SUM(t.amount), 0) as spent
                    FROM budgets b
                    LEFT JOIN transactions t ON t.user_id = b.user_id 
                        AND t.category = b.category 
                        AND t.type = 'expense'
                        AND MONTH(t.transaction_date) = b.month 
                        AND YEAR(t.transaction_date) = b.year
                    WHERE b.user_id = ? AND b.month = ? AND b.year = ?
                    GROUP BY b.id
                    HAVING spent > amount`
    
    budgetRows, err := database.DB.Query(budgetQuery, userID, currentMonth, currentYear)
    if err == nil {
        defer budgetRows.Close()
        for budgetRows.Next() {
            var category string
            var budgetAmount, spent float64
            if err := budgetRows.Scan(&category, &budgetAmount, &spent); err == nil {
                recommendations = append(recommendations,
                    fmt.Sprintf("⚠️ Peringatan: Anda telah melebihi budget untuk kategori %s. Budget: Rp %.0f, Terpakai: Rp %.0f",
                        category, budgetAmount, spent))
            }
        }
    }
    
    // Add general recommendations
    if len(recommendations) == 0 {
        recommendations = append(recommendations,
            "Bagus! Pengeluaran Anda terkontrol dengan baik. Pertahankan kebiasaan ini!",
            "Pertimbangkan untuk menginvestasikan 20% dari pendapatan Anda untuk masa depan.",
            "Buat dana darurat yang mencakup 3-6 bulan pengeluaran.")
    }
    
    return recommendations
}