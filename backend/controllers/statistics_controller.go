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

	query := `SELECT COALESCE(currency_code, 'IDR') as currency_code,
	                 COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as balance
	          FROM transactions
	          WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL)
	          GROUP BY COALESCE(currency_code, 'IDR')
	          ORDER BY currency_code ASC`

	rows, err := database.DB.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance"})
		return
	}
	defer rows.Close()

	perCurrency := map[string]float64{}
	for rows.Next() {
		var code string
		var bal float64
		if err := rows.Scan(&code, &bal); err == nil {
			perCurrency[code] = bal
		}
	}

	c.JSON(http.StatusOK, gin.H{"per_currency": perCurrency})
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

	// per_currency response
	perCurrency := map[string]*Statistics{}

	// Totals grouped by currency
	totalsQuery := `SELECT 
	                  COALESCE(currency_code, 'IDR') as currency_code,
	                  COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
	                  COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense
	                FROM transactions
	                WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
	                GROUP BY COALESCE(currency_code, 'IDR')
	                ORDER BY currency_code ASC`

	rows, err := database.DB.Query(totalsQuery, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate totals"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var code string
		var inc, exp float64
		if err := rows.Scan(&code, &inc, &exp); err != nil && err != sql.ErrNoRows {
			continue
		}
		perCurrency[code] = &Statistics{
			TotalIncome:      inc,
			TotalExpense:     exp,
			Balance:          inc - exp,
			CategorySpending: map[string]float64{},
			MonthlyTrend:     []MonthlyData{},
		}
	}

	// Category spending grouped by currency+category
	categoryQuery := `SELECT COALESCE(currency_code, 'IDR') as currency_code, category, COALESCE(SUM(amount), 0) as total
	                  FROM transactions
	                  WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND type = 'expense' AND transaction_date BETWEEN ? AND ?
	                  GROUP BY COALESCE(currency_code, 'IDR'), category
	                  ORDER BY currency_code ASC, total DESC`

	rows2, err := database.DB.Query(categoryQuery, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get category data"})
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var code, category string
		var total float64
		if err := rows2.Scan(&code, &category, &total); err == nil {
			s, ok := perCurrency[code]
			if !ok {
				s = &Statistics{CategorySpending: map[string]float64{}, MonthlyTrend: []MonthlyData{}}
				perCurrency[code] = s
			}
			if s.CategorySpending == nil {
				s.CategorySpending = map[string]float64{}
			}
			s.CategorySpending[category] = total
		}
	}

	// Monthly trend grouped by currency+month
	monthlyQuery := `SELECT 
	                   COALESCE(currency_code, 'IDR') as currency_code,
	                   DATE_FORMAT(transaction_date, '%Y-%m') as month,
	                   COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
	                   COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
	                 FROM transactions 
	                 WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
	                 GROUP BY COALESCE(currency_code, 'IDR'), DATE_FORMAT(transaction_date, '%Y-%m')
	                 ORDER BY currency_code ASC, month ASC`

	rows3, err := database.DB.Query(monthlyQuery, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monthly data"})
		return
	}
	defer rows3.Close()

	for rows3.Next() {
		var code string
		var md MonthlyData
		if err := rows3.Scan(&code, &md.Month, &md.Income, &md.Expense); err == nil {
			s, ok := perCurrency[code]
			if !ok {
				s = &Statistics{CategorySpending: map[string]float64{}, MonthlyTrend: []MonthlyData{}}
				perCurrency[code] = s
			}
			s.MonthlyTrend = append(s.MonthlyTrend, md)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"per_currency": perCurrency,
		"start_date":   startDate.Format("2006-01-02"),
		"end_date":     endDate.Format("2006-01-02"),
		"range":        rangeType,
	})
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

	query := `SELECT COALESCE(currency_code, 'IDR') as currency_code, category, COALESCE(SUM(amount), 0) as total 
              FROM transactions 
              WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND type = 'expense' AND transaction_date BETWEEN ? AND ?
              GROUP BY COALESCE(currency_code, 'IDR'), category
              ORDER BY currency_code ASC, total DESC`

	rows, err := database.DB.Query(query, userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chart data"})
		return
	}
	defer rows.Close()

	perCurrency := map[string]gin.H{}
	labelsBy := map[string][]string{}
	dataBy := map[string][]float64{}

	for rows.Next() {
		var code, category string
		var total float64
		if err := rows.Scan(&code, &category, &total); err == nil {
			labelsBy[code] = append(labelsBy[code], category)
			dataBy[code] = append(dataBy[code], total)
		}
	}

	for code, labels := range labelsBy {
		perCurrency[code] = gin.H{"labels": labels, "data": dataBy[code]}
	}

	c.JSON(http.StatusOK, gin.H{"per_currency": perCurrency})
}

func getTrendChartDataWithRange(c *gin.Context, userID int, rangeType string) {
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

	var query string
	var rows *sql.Rows
	var err error

	if rangeType == "7D" || rangeType == "30D" || rangeType == "12W" {
		query = `SELECT 
		            COALESCE(currency_code, 'IDR') as currency_code,
                    DATE(transaction_date) as period,
                    COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
                    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
                 FROM transactions 
                 WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
                 GROUP BY COALESCE(currency_code, 'IDR'), DATE(transaction_date)
                 ORDER BY currency_code ASC, period ASC`

		rows, err = database.DB.Query(query, userID, startDate, endDate)
	} else {
		query = `SELECT 
		            COALESCE(currency_code, 'IDR') as currency_code,
                    DATE_FORMAT(transaction_date, '%Y-%m') as period,
                    COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
                    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
                 FROM transactions 
                 WHERE user_id = ? AND (is_transfer = FALSE OR is_transfer IS NULL) AND transaction_date BETWEEN ? AND ?
                 GROUP BY COALESCE(currency_code, 'IDR'), DATE_FORMAT(transaction_date, '%Y-%m')
                 ORDER BY currency_code ASC, period ASC`

		rows, err = database.DB.Query(query, userID, startDate, endDate)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trend data"})
		return
	}
	defer rows.Close()

	periodsBy := map[string][]string{}
	incomeBy := map[string][]float64{}
	expenseBy := map[string][]float64{}

	for rows.Next() {
		var code, period string
		var income, expense float64
		if err := rows.Scan(&code, &period, &income, &expense); err == nil {
			periodsBy[code] = append(periodsBy[code], period)
			incomeBy[code] = append(incomeBy[code], income)
			expenseBy[code] = append(expenseBy[code], expense)
		}
	}

	perCurrency := map[string]gin.H{}
	for code, periods := range periodsBy {
		perCurrency[code] = gin.H{
			"months":  periods,
			"income":  incomeBy[code],
			"expense": expenseBy[code],
		}
	}

	c.JSON(http.StatusOK, gin.H{"per_currency": perCurrency})
}
