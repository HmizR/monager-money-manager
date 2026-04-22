package controllers

import (
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"
    "money-manager/database"
    "money-manager/models"
    "github.com/gin-gonic/gin"
)

func ExportTransactions(c *gin.Context) {
    userID := c.GetInt("userID")
    format := c.DefaultQuery("format", "csv")
    
    rows, err := database.DB.Query(`
        SELECT amount, type, category, description, transaction_date 
        FROM transactions 
        WHERE user_id = ? 
        ORDER BY transaction_date DESC`, userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
        return
    }
    defer rows.Close()
    
    if format == "json" {
        var transactions []map[string]interface{}
        for rows.Next() {
            var amount float64
            var transType, category, description string
            var date time.Time
            
            rows.Scan(&amount, &transType, &category, &description, &date)
            transactions = append(transactions, map[string]interface{}{
                "amount":      amount,
                "type":        transType,
                "category":    category,
                "description": description,
                "date":        date.Format("2006-01-02"),
            })
        }
        
        c.Header("Content-Type", "application/json")
        c.Header("Content-Disposition", "attachment; filename=transactions.json")
        c.JSON(http.StatusOK, transactions)
        return
    }
    
    // Default CSV export
    c.Header("Content-Type", "text/csv")
    c.Header("Content-Disposition", "attachment; filename=transactions.csv")
    
    writer := csv.NewWriter(c.Writer)
    writer.Write([]string{"Amount", "Type", "Category", "Description", "Date"})
    
    for rows.Next() {
        var amount float64
        var transType, category, description string
        var date time.Time
        
        rows.Scan(&amount, &transType, &category, &description, &date)
        writer.Write([]string{
            strconv.FormatFloat(amount, 'f', 2, 64),
            transType,
            category,
            description,
            date.Format("2006-01-02"),
        })
    }
    
    writer.Flush()
}

func ImportTransactions(c *gin.Context) {
    userID := c.GetInt("userID")
    
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }
    
    src, err := file.Open()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
        return
    }
    defer src.Close()
    
    // Check file extension
    filename := file.Filename
    var importedCount int
    
    if len(filename) > 5 && filename[len(filename)-5:] == ".json" {
        importedCount, err = importJSON(src, userID)
    } else if len(filename) > 4 && filename[len(filename)-4:] == ".csv" {
        importedCount, err = importCSV(src, userID)
    } else {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file format. Please upload CSV or JSON file"})
        return
    }
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": fmt.Sprintf("Successfully imported %d transactions", importedCount),
        "count":   importedCount,
    })
}

func importCSV(file io.Reader, userID int) (int, error) {
    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return 0, err
    }
    
    if len(records) < 2 {
        return 0, fmt.Errorf("CSV file is empty")
    }
    
    imported := 0
    for i, record := range records {
        if i == 0 {
            continue // Skip header
        }
        
        if len(record) < 5 {
            continue
        }
        
        amount, _ := strconv.ParseFloat(record[0], 64)
        transType := record[1]
        category := record[2]
        description := record[3]
        date, _ := time.Parse("2006-01-02", record[4])
        
        transactionUUID := models.GenerateUUID()
        _, err := database.DB.Exec(`
            INSERT INTO transactions (uuid, user_id, amount, type, category, description, transaction_date)
            VALUES (?, ?, ?, ?, ?, ?, ?)`,
            transactionUUID, userID, amount, transType, category, description, date)
        
        if err == nil {
            imported++
        }
    }
    
    return imported, nil
}

func importJSON(file io.Reader, userID int) (int, error) {
    var transactions []struct {
        Amount      float64 `json:"amount"`
        Type        string  `json:"type"`
        Category    string  `json:"category"`
        Description string  `json:"description"`
        Date        string  `json:"date"`
    }
    
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&transactions); err != nil {
        return 0, err
    }
    
    imported := 0
    for _, t := range transactions {
        date, _ := time.Parse("2006-01-02", t.Date)
        transactionUUID := models.GenerateUUID()
        
        _, err := database.DB.Exec(`
            INSERT INTO transactions (uuid, user_id, amount, type, category, description, transaction_date)
            VALUES (?, ?, ?, ?, ?, ?, ?)`,
            transactionUUID, userID, t.Amount, t.Type, t.Category, t.Description, date)
        
        if err == nil {
            imported++
        }
    }
    
    return imported, nil
}