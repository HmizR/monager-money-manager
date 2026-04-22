package controllers

import (
    "net/http"
    "time"
    "money-manager/database"
    "github.com/gin-gonic/gin"
)

type BalanceTrend struct {
    Date          string  `json:"date"`
    Balance       float64 `json:"balance"`
    CumulativeIncome  float64 `json:"cumulative_income"`
    CumulativeExpense float64 `json:"cumulative_expense"`
}

type BalanceForecast struct {
    StartDate       string  `json:"start_date"`
    EndDate         string  `json:"end_date"`
    StartingBalance float64 `json:"starting_balance"`
    ExpectedIncome  float64 `json:"expected_income"`
    ExpectedExpense float64 `json:"expected_expense"`
    EndingBalance   float64 `json:"ending_balance"`
    DailyAverageIncome float64 `json:"daily_average_income"`
    DailyAverageExpense float64 `json:"daily_average_expense"`
    Recommendations []string `json:"recommendations"`
}

func GetBalanceTrend(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Get range parameter
    rangeType := c.DefaultQuery("range", "30D")
    endDate := time.Now()
    
    var startDate time.Time
    switch rangeType {
    case "7D":
        startDate = endDate.AddDate(0, 0, -7)
    case "30D":
        startDate = endDate.AddDate(0, 0, -30)
    case "12W":
        startDate = endDate.AddDate(0, 0, -84)
    case "6M":
        startDate = endDate.AddDate(0, -6, 0)
    case "1Y":
        startDate = endDate.AddDate(-1, 0, 0)
    default:
        startDate = endDate.AddDate(0, 0, -30)
    }
    
    if start := c.Query("start_date"); start != "" {
        if parsed, err := time.Parse("2006-01-02", start); err == nil {
            startDate = parsed
        }
    }
    if end := c.Query("end_date"); end != "" {
        if parsed, err := time.Parse("2006-01-02", end); err == nil {
            endDate = parsed
        }
    }
    
    // Get daily balance trend
    query := `
        SELECT 
            DATE(transaction_date) as date,
            SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as daily_income,
            SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as daily_expense
        FROM transactions
        WHERE user_id = ? AND transaction_date BETWEEN ? AND ?
        GROUP BY DATE(transaction_date)
        ORDER BY date ASC`
    
    rows, err := database.DB.Query(query, userID, startDate, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance trend"})
        return
    }
    defer rows.Close()
    
    var trends []BalanceTrend
    runningBalance := 0.0
    cumulativeIncome := 0.0
    cumulativeExpense := 0.0
    
    for rows.Next() {
        var date time.Time
        var dailyIncome, dailyExpense float64
        
        if err := rows.Scan(&date, &dailyIncome, &dailyExpense); err != nil {
            continue
        }
        
        cumulativeIncome += dailyIncome
        cumulativeExpense += dailyExpense
        runningBalance = cumulativeIncome - cumulativeExpense
        
        trends = append(trends, BalanceTrend{
            Date:          date.Format("2006-01-02"),
            Balance:       runningBalance,
            CumulativeIncome:  cumulativeIncome,
            CumulativeExpense: cumulativeExpense,
        })
    }
    
    c.JSON(http.StatusOK, gin.H{
        "trends": trends,
        "start_date": startDate.Format("2006-01-02"),
        "end_date": endDate.Format("2006-01-02"),
    })
}

func GetBalanceForecast(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Get historical data for last 30 days to calculate averages
    endDate := time.Now()
    startDate := endDate.AddDate(0, 0, -30)
    
    // Calculate average daily income and expense
    var totalIncome, totalExpense float64
    var uniqueDays int
    
    incomeQuery := `
        SELECT DATE(transaction_date), SUM(amount)
        FROM transactions
        WHERE user_id = ? AND type = 'income' AND transaction_date BETWEEN ? AND ?
        GROUP BY DATE(transaction_date)`
    
    rows, err := database.DB.Query(incomeQuery, userID, startDate, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate income average"})
        return
    }
    defer rows.Close()
    
    incomeDays := 0
    for rows.Next() {
        var date time.Time
        var amount float64
        if err := rows.Scan(&date, &amount); err == nil {
            totalIncome += amount
            incomeDays++
        }
    }
    
    expenseQuery := `
        SELECT DATE(transaction_date), SUM(amount)
        FROM transactions
        WHERE user_id = ? AND type = 'expense' AND transaction_date BETWEEN ? AND ?
        GROUP BY DATE(transaction_date)`
    
    rows, err = database.DB.Query(expenseQuery, userID, startDate, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate expense average"})
        return
    }
    defer rows.Close()
    
    expenseDays := 0
    for rows.Next() {
        var date time.Time
        var amount float64
        if err := rows.Scan(&date, &amount); err == nil {
            totalExpense += amount
            expenseDays++
        }
    }
    
    // Get unique days count
    uniqueDaysQuery := `SELECT COUNT(DISTINCT DATE(transaction_date)) FROM transactions WHERE user_id = ? AND transaction_date BETWEEN ? AND ?`
    database.DB.QueryRow(uniqueDaysQuery, userID, startDate, endDate).Scan(&uniqueDays)
    if uniqueDays == 0 {
        uniqueDays = 1
    }
    
    dailyAvgIncome := totalIncome / float64(uniqueDays)
    dailyAvgExpense := totalExpense / float64(uniqueDays)
    
    // Get current balance
    var currentBalance float64
    balanceQuery := `
        SELECT SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END)
        FROM transactions
        WHERE user_id = ? AND transaction_date <= ?`
    database.DB.QueryRow(balanceQuery, userID, endDate).Scan(&currentBalance)
    
    // Forecast for next 30 days
    forecastDays := 30
    if days := c.Query("days"); days != "" {
        if d, err := time.ParseDuration(days + "h"); err == nil {
            forecastDays = int(d.Hours() / 24)
        }
    }
    
    expectedIncome := dailyAvgIncome * float64(forecastDays)
    expectedExpense := dailyAvgExpense * float64(forecastDays)
    endingBalance := currentBalance + expectedIncome - expectedExpense
    
    // Generate recommendations based on forecast
    recommendations := []string{}
    if endingBalance < 0 {
        recommendations = append(recommendations, 
            "⚠️ Peringatan: Balance diperkirakan akan negatif dalam 30 hari. Pertimbangkan untuk mengurangi pengeluaran atau meningkatkan pendapatan.")
    }
    
    if expectedExpense > expectedIncome {
        recommendations = append(recommendations,
            "📉 Pengeluaran diperkirakan melebihi pendapatan. Buatlah rencana penghematan.")
    }
    
    if dailyAvgExpense > dailyAvgIncome*0.7 {
        recommendations = append(recommendations,
            "💡 Pengeluaran Anda cukup tinggi (70%+ dari pendapatan). Coba tingkatkan tabungan.")
    }
    
    if currentBalance < dailyAvgExpense*30 {
        recommendations = append(recommendations,
            "🏦 Dana darurat Anda hanya cukup untuk kurang dari 30 hari. Targetkan 3-6 bulan pengeluaran.")
    }
    
    if len(recommendations) == 0 {
        recommendations = append(recommendations,
            "✅ Keuangan Anda sehat! Pertahankan pola ini.",
            "🎯 Pertimbangkan untuk berinvestasi 20% dari pendapatan.")
    }
    
    forecast := BalanceForecast{
        StartDate:          endDate.Format("2006-01-02"),
        EndDate:            endDate.AddDate(0, 0, forecastDays).Format("2006-01-02"),
        StartingBalance:    currentBalance,
        ExpectedIncome:     expectedIncome,
        ExpectedExpense:    expectedExpense,
        EndingBalance:      endingBalance,
        DailyAverageIncome: dailyAvgIncome,
        DailyAverageExpense: dailyAvgExpense,
        Recommendations:    recommendations,
    }
    
    c.JSON(http.StatusOK, forecast)
}

func GetMonthlyProjection(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Get current month and year
    now := time.Now()
    currentMonth := now.Month()
    currentYear := now.Year()
    
    // Get actual spending this month so far
    var actualIncome, actualExpense float64
    actualQuery := `
        SELECT 
            SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as income,
            SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as expense
        FROM transactions
        WHERE user_id = ? AND MONTH(transaction_date) = ? AND YEAR(transaction_date) = ?`
    
    database.DB.QueryRow(actualQuery, userID, currentMonth, currentYear).Scan(&actualIncome, &actualExpense)
    
    // Get budgeted amounts
    budgetQuery := `
        SELECT category, amount
        FROM budgets
        WHERE user_id = ? AND month = ? AND year = ?`
    
    rows, err := database.DB.Query(budgetQuery, userID, currentMonth, currentYear)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch budgets"})
        return
    }
    defer rows.Close()
    
    totalBudget := 0.0
    budgets := []map[string]interface{}{}
    for rows.Next() {
        var category string
        var amount float64
        if err := rows.Scan(&category, &amount); err == nil {
            totalBudget += amount
            budgets = append(budgets, map[string]interface{}{
                "category": category,
                "budget":   amount,
            })
        }
    }
    
    // Calculate projected end of month balance
    daysInMonth := time.Date(currentYear, currentMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
    currentDay := now.Day()
    remainingDays := daysInMonth - currentDay
    
    var projectedIncome, projectedExpense float64
    if currentDay > 0 {
        dailyAvgIncome := actualIncome / float64(currentDay)
        dailyAvgExpense := actualExpense / float64(currentDay)
        projectedIncome = actualIncome + (dailyAvgIncome * float64(remainingDays))
        projectedExpense = actualExpense + (dailyAvgExpense * float64(remainingDays))
    } else {
        projectedIncome = actualIncome
        projectedExpense = actualExpense
    }
    
    // Get starting balance (end of last month)
    var startingBalance float64
    startQuery := `
        SELECT SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END)
        FROM transactions
        WHERE user_id = ? AND transaction_date < ?`
    monthStart := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
    database.DB.QueryRow(startQuery, userID, monthStart).Scan(&startingBalance)
    
    projection := map[string]interface{}{
        "month":                currentMonth.String(),
        "year":                 currentYear,
        "starting_balance":     startingBalance,
        "actual_income":        actualIncome,
        "actual_expense":       actualExpense,
        "projected_income":     projectedIncome,
        "projected_expense":    projectedExpense,
        "projected_ending_balance": startingBalance + projectedIncome - projectedExpense,
        "budget_vs_actual":     totalBudget - actualExpense,
        "days_remaining":       remainingDays,
        "budgets":              budgets,
    }
    
    c.JSON(http.StatusOK, projection)
}