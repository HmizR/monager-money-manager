package controllers

import (
	"database/sql"
	"money-manager/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Statistics struct {
	TotalIncome      float64            `json:"total_income"`
	TotalExpense     float64            `json:"total_expense"`
	Balance          float64            `json:"balance"`
	CategorySpending map[string]float64 `json:"category_spending"`
	MonthlyTrend     []MonthlyData      `json:"monthly_trend"`
}

type MonthlyData struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

func GetCurrentBalance(c *gin.Context) {
	userID := c.GetInt("userID")

	var balance float64
	query := `SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) 
              FROM transactions WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL)`

	database.DB.QueryRow(query, userID).Scan(&balance)

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func GetStatistics(c *gin.Context) {
	userID := c.GetInt("userID")

	// Get range parameter
	rangeType := c.DefaultQuery("range", "30D") // 7D, 30D, 12W, 6M, 1Y

	var startDate, endDate time.Time
	endDate = time.Now()

	switch rangeType {
	case "7D":
		startDate = endDate.AddDate(0, 0, -7)
	case "30D":
		startDate = endDate.AddDate(0, 0, -30)
	case "12W":
		startDate = endDate.AddDate(0, 0, -84) // 12 * 7 hari
	case "6M":
		startDate = endDate.AddDate(0, -6, 0)
	case "1Y":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(0, 0, -30)
	}

	// Override dengan custom date jika ada
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

	stats := Statistics{
		CategorySpending: make(map[string]float64),
		MonthlyTrend:     []MonthlyData{},
	}

	// Get total income and expense
	query := `SELECT 
                SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as total_income,
                SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as total_expense
              FROM transactions 
              WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?`

	err := database.DB.QueryRow(query, userID, startDate, endDate).Scan(&stats.TotalIncome, &stats.TotalExpense)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate totals"})
		return
	}

	stats.Balance = stats.TotalIncome - stats.TotalExpense

	// Get category spending
	categoryQuery := `SELECT category, SUM(amount) as total 
                      FROM transactions 
                      WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND type = 'expense' AND transaction_date BETWEEN ? AND ?
                      GROUP BY category`

	rows, err := database.DB.Query(categoryQuery, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category data"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var total float64
		if err := rows.Scan(&category, &total); err == nil {
			stats.CategorySpending[category] = total
		}
	}

	// Get monthly trend
	monthlyQuery := `SELECT 
                        DATE_FORMAT(transaction_date, '%Y-%m') as month,
                        SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as income,
                        SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as expense
                     FROM transactions 
                     WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
                     GROUP BY DATE_FORMAT(transaction_date, '%Y-%m')
                     ORDER BY month ASC`

	rows, err = database.DB.Query(monthlyQuery, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monthly data"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var monthData MonthlyData
		if err := rows.Scan(&monthData.Month, &monthData.Income, &monthData.Expense); err == nil {
			stats.MonthlyTrend = append(stats.MonthlyTrend, monthData)
		}
	}

	c.JSON(http.StatusOK, stats)
}

// In GetChartData function
func GetChartData(c *gin.Context) {
	userID := c.GetInt("userID")
	chartType := c.Query("type")
	rangeType := c.DefaultQuery("range", "30D") // Tambah ini

	switch chartType {
	case "category":
		getCategoryChartData(c, userID, rangeType)
	case "trend":
		getTrendChartDataWithRange(c, userID, rangeType) // Modified
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chart type"})
	}
}

func getCategoryChartData(c *gin.Context, userID int, rangeType string) {
	endDate := time.Now()
	var startDate time.Time // VARIABLE INI TIDAK DIGUNAKAN - HAPUS ATAU GUNAKAN

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

	query := `SELECT category, SUM(amount) as total 
              FROM transactions 
              WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND type = 'expense' AND transaction_date BETWEEN ? AND ?
              GROUP BY category`

	rows, err := database.DB.Query(query, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chart data"})
		return
	}
	defer rows.Close()

	labels := []string{}
	data := []float64{}

	for rows.Next() {
		var category string
		var total float64
		if err := rows.Scan(&category, &total); err == nil {
			labels = append(labels, category)
			data = append(data, total)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"labels": labels,
		"data":   data,
	})
}

func getTrendChartDataWithRange(c *gin.Context, userID int, rangeType string) {
	endDate := time.Now()
	var startDate time.Time // VARIABLE INI TIDAK DIGUNAKAN - HAPUS ATAU GUNAKAN

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

	var query string // VARIABLE INI JUGA TIDAK DIGUNAKAN

	// Jika query tidak pernah digunakan, hapus deklarasi ini
	// Atau gunakan query tersebut untuk database query

	// Contoh jika ingin menggunakan query:
	var rows *sql.Rows
	var err error

	if rangeType == "7D" || rangeType == "30D" || rangeType == "12W" {
		query = `SELECT 
                    DATE(transaction_date) as period,
                    SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as income,
                    SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as expense
                 FROM transactions 
                 WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
                 GROUP BY DATE(transaction_date)
                 ORDER BY period ASC`

		rows, err = database.DB.Query(query, userID, startDate, endDate)
	} else {
		query = `SELECT 
                    DATE_FORMAT(transaction_date, '%Y-%m') as period,
                    SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) as income,
                    SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) as expense
                 FROM transactions 
                 WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
                 GROUP BY DATE_FORMAT(transaction_date, '%Y-%m')
                 ORDER BY period ASC`

		rows, err = database.DB.Query(query, userID, startDate, endDate)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trend data"})
		return
	}
	defer rows.Close()

	periods := []string{}
	incomeData := []float64{}
	expenseData := []float64{}

	for rows.Next() {
		var period string
		var income, expense float64
		if err := rows.Scan(&period, &income, &expense); err == nil {
			periods = append(periods, period)
			incomeData = append(incomeData, income)
			expenseData = append(expenseData, expense)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"months":  periods,
		"income":  incomeData,
		"expense": expenseData,
	})
}
