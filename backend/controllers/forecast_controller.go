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
    
    // Get daily balance trend per currency (exclude transfers)
    query := `
        SELECT 
            COALESCE(currency_code, 'IDR') as currency_code,
            DATE(transaction_date) as date,
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as daily_income,
            COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as daily_expense
        FROM transactions
        WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
        GROUP BY COALESCE(currency_code, 'IDR'), DATE(transaction_date)
        ORDER BY currency_code ASC, date ASC`
    
    rows, err := database.DB.Query(query, userID, startDate, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance trend"})
        return
    }
    defer rows.Close()
    
    perCurrency := map[string][]BalanceTrend{}
    runningBalance := map[string]float64{}
    cumulativeIncome := map[string]float64{}
    cumulativeExpense := map[string]float64{}
    
    for rows.Next() {
        var code string
        var date time.Time
        var dailyIncome, dailyExpense float64
        
        if err := rows.Scan(&code, &date, &dailyIncome, &dailyExpense); err != nil {
            continue
        }
        
        cumulativeIncome[code] += dailyIncome
        cumulativeExpense[code] += dailyExpense
        runningBalance[code] = cumulativeIncome[code] - cumulativeExpense[code]
        
        perCurrency[code] = append(perCurrency[code], BalanceTrend{
            Date:          date.Format("2006-01-02"),
            Balance:       runningBalance[code],
            CumulativeIncome:  cumulativeIncome[code],
            CumulativeExpense: cumulativeExpense[code],
        })
    }
    
    c.JSON(http.StatusOK, gin.H{
        "per_currency": perCurrency,
        "start_date": startDate.Format("2006-01-02"),
        "end_date": endDate.Format("2006-01-02"),
    })
}

func GetBalanceForecast(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Get historical data for last 30 days to calculate averages
    endDate := time.Now()
    startDate := endDate.AddDate(0, 0, -30)
    
    // Forecast for next 30 days
    forecastDays := 30
    if days := c.Query("days"); days != "" {
        if d, err := time.ParseDuration(days + "h"); err == nil {
            forecastDays = int(d.Hours() / 24)
        }
    }

    // Average per currency using last 30 days (exclude transfers).
    avgQuery := `
        SELECT 
            COALESCE(currency_code, 'IDR') as currency_code,
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
            COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense,
            GREATEST(COUNT(DISTINCT DATE(transaction_date)), 1) as unique_days
        FROM transactions
        WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
        GROUP BY COALESCE(currency_code, 'IDR')
        ORDER BY currency_code ASC`

    rows, err := database.DB.Query(avgQuery, userID, startDate, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate averages"})
        return
    }
    defer rows.Close()

    type avgRow struct {
        code      string
        incTotal  float64
        expTotal  float64
        uniqueDays int
    }
    avgs := []avgRow{}
    for rows.Next() {
        var r avgRow
        if err := rows.Scan(&r.code, &r.incTotal, &r.expTotal, &r.uniqueDays); err == nil {
            avgs = append(avgs, r)
        }
    }

    // Current balance per currency (exclude transfers).
    balanceQuery := `
        SELECT 
            COALESCE(currency_code, 'IDR') as currency_code,
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as balance
        FROM transactions
        WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date <= ?
        GROUP BY COALESCE(currency_code, 'IDR')
        ORDER BY currency_code ASC`
    rows2, err := database.DB.Query(balanceQuery, userID, endDate)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate current balance"})
        return
    }
    defer rows2.Close()

    balances := map[string]float64{}
    for rows2.Next() {
        var code string
        var bal float64
        if err := rows2.Scan(&code, &bal); err == nil {
            balances[code] = bal
        }
    }
    
    perCurrency := map[string]BalanceForecast{}

    // Build forecasts for currencies that appear in balance or last-30d window.
    seen := map[string]bool{}
    for _, r := range avgs {
        seen[r.code] = true
    }
    for code := range balances {
        seen[code] = true
    }

    for code := range seen {
        // defaults
        incTotal := 0.0
        expTotal := 0.0
        uniqueDays := 1
        for _, r := range avgs {
            if r.code == code {
                incTotal = r.incTotal
                expTotal = r.expTotal
                uniqueDays = r.uniqueDays
                if uniqueDays <= 0 {
                    uniqueDays = 1
                }
                break
            }
        }

        dailyAvgIncome := incTotal / float64(uniqueDays)
        dailyAvgExpense := expTotal / float64(uniqueDays)
        currentBalance := balances[code]

        expectedIncome := dailyAvgIncome * float64(forecastDays)
        expectedExpense := dailyAvgExpense * float64(forecastDays)
        endingBalance := currentBalance + expectedIncome - expectedExpense

        recommendations := []string{}
        if endingBalance < 0 {
            recommendations = append(recommendations,
                "Peringatan: saldo diperkirakan akan negatif dalam 30 hari. Pertimbangkan untuk mengurangi pengeluaran atau meningkatkan pendapatan.")
        }
        if expectedExpense > expectedIncome {
            recommendations = append(recommendations,
                "Pengeluaran diperkirakan melebihi pendapatan. Buatlah rencana penghematan.")
        }
        if dailyAvgIncome > 0 && dailyAvgExpense > dailyAvgIncome*0.7 {
            recommendations = append(recommendations,
                "Pengeluaran Anda cukup tinggi (70%+ dari pendapatan). Coba tingkatkan tabungan.")
        }
        if currentBalance < dailyAvgExpense*30 {
            recommendations = append(recommendations,
                "Dana darurat Anda hanya cukup untuk kurang dari 30 hari. Targetkan 3-6 bulan pengeluaran.")
        }
        if len(recommendations) == 0 {
            recommendations = append(recommendations,
                "Keuangan Anda sehat! Pertahankan pola ini.",
                "Pertimbangkan untuk berinvestasi 20% dari pendapatan.")
        }

        perCurrency[code] = BalanceForecast{
            StartDate:           endDate.Format("2006-01-02"),
            EndDate:             endDate.AddDate(0, 0, forecastDays).Format("2006-01-02"),
            StartingBalance:     currentBalance,
            ExpectedIncome:      expectedIncome,
            ExpectedExpense:     expectedExpense,
            EndingBalance:       endingBalance,
            DailyAverageIncome:  dailyAvgIncome,
            DailyAverageExpense: dailyAvgExpense,
            Recommendations:     recommendations,
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "start_date":   endDate.Format("2006-01-02"),
        "end_date":     endDate.AddDate(0, 0, forecastDays).Format("2006-01-02"),
        "days":         forecastDays,
        "per_currency": perCurrency,
    })
}

func GetMonthlyProjection(c *gin.Context) {
    userID := c.GetInt("userID")
    
    // Get current month and year
    now := time.Now()
    currentMonth := now.Month()
    currentYear := now.Year()
    
    // Get actual spending this month so far per currency (exclude transfers)
    actualQuery := `
        SELECT 
            COALESCE(currency_code, 'IDR') as currency_code,
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
            COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
        FROM transactions
        WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND MONTH(transaction_date) = ? AND YEAR(transaction_date) = ?
        GROUP BY COALESCE(currency_code, 'IDR')
        ORDER BY currency_code ASC`
    
    rowsAct, err := database.DB.Query(actualQuery, userID, currentMonth, currentYear)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch actual totals"})
        return
    }
    defer rowsAct.Close()

    actualIncomeBy := map[string]float64{}
    actualExpenseBy := map[string]float64{}
    for rowsAct.Next() {
        var code string
        var inc, exp float64
        if err := rowsAct.Scan(&code, &inc, &exp); err == nil {
            actualIncomeBy[code] = inc
            actualExpenseBy[code] = exp
        }
    }
    
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

    // Get starting balance (end of last month) per currency (exclude transfers)
    startQuery := `
        SELECT 
            COALESCE(currency_code, 'IDR') as currency_code,
            COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as balance
        FROM transactions
        WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date < ?
        GROUP BY COALESCE(currency_code, 'IDR')
        ORDER BY currency_code ASC`
    monthStart := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.UTC)
    rowsStart, err := database.DB.Query(startQuery, userID, monthStart)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate starting balance"})
        return
    }
    defer rowsStart.Close()

    startingBy := map[string]float64{}
    for rowsStart.Next() {
        var code string
        var bal float64
        if err := rowsStart.Scan(&code, &bal); err == nil {
            startingBy[code] = bal
        }
    }

    // Union currencies
    seen := map[string]bool{}
    for code := range startingBy {
        seen[code] = true
    }
    for code := range actualIncomeBy {
        seen[code] = true
    }
    for code := range actualExpenseBy {
        seen[code] = true
    }

    perCurrency := map[string]map[string]interface{}{}
    for code := range seen {
        actualIncome := actualIncomeBy[code]
        actualExpense := actualExpenseBy[code]

        projectedIncome := actualIncome
        projectedExpense := actualExpense
        if currentDay > 0 {
            dailyAvgIncome := actualIncome / float64(currentDay)
            dailyAvgExpense := actualExpense / float64(currentDay)
            projectedIncome = actualIncome + (dailyAvgIncome * float64(remainingDays))
            projectedExpense = actualExpense + (dailyAvgExpense * float64(remainingDays))
        }

        startingBalance := startingBy[code]

        perCurrency[code] = map[string]interface{}{
            "starting_balance":         startingBalance,
            "actual_income":            actualIncome,
            "actual_expense":           actualExpense,
            "projected_income":         projectedIncome,
            "projected_expense":        projectedExpense,
            "projected_ending_balance": startingBalance + projectedIncome - projectedExpense,
            // budgets table currently has no currency dimension; avoid misleading values.
            "budget_vs_actual":         nil,
        }
    }

    projection := map[string]interface{}{
        "month":         currentMonth.String(),
        "year":          currentYear,
        "days_remaining": remainingDays,
        "budgets":       budgets,
        "per_currency":  perCurrency,
    }

    c.JSON(http.StatusOK, projection)
}