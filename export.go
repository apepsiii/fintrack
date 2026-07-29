package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ExportService handles data export functionality
type ExportService struct{}

// NewExportService creates a new export service
func NewExportService() *ExportService {
	return &ExportService{}
}

// ExportTransactionsCSV exports transactions to CSV format
func (e *ExportService) ExportTransactionsCSV(c *gin.Context, userID int, startDate, endDate time.Time) error {
	// Query transactions for the user within date range
	rows, err := db.Query(`
		SELECT t.id, t.type, t.amount, t.date, t.note, c.name as category
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = ? AND t.date BETWEEN ? AND ?
		ORDER BY t.date DESC
	`, userID, startDate, endDate)
	
	if err != nil {
		return err
	}
	defer rows.Close()

	// Set CSV headers
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=fintrack_transactions_%s_to_%s.csv", 
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))

	// Create CSV writer
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write CSV header
	writer.Write([]string{"ID", "Date", "Type", "Category", "Amount", "Note"})

	// Write transactions
	for rows.Next() {
		var id, amount int
		var txType, category, note string
		var date time.Time

		if err := rows.Scan(&id, &txType, &amount, &date, &note, &category); err != nil {
			continue
		}

		// Format type
		typeDisplay := "Pemasukan"
		if txType == "expense" {
			typeDisplay = "Pengeluaran"
		}

		writer.Write([]string{
			strconv.Itoa(id),
			date.Format("02/01/2006 15:04"),
			typeDisplay,
			category,
			fmt.Sprintf("Rp %s", formatRupiah(amount)),
			note,
		})
	}

	return nil
}

// ExportBudgetCSV exports budget report to CSV
func (e *ExportService) ExportBudgetCSV(c *gin.Context, userID int, monthYear string) error {
	rows, err := db.Query(`
		SELECT c.name, b.amount_limit,
			   COALESCE((SELECT SUM(amount) FROM transactions 
			             WHERE category_id = c.id 
			             AND user_id = ? 
			             AND type = 'expense' 
			             AND substr(date, 1, 7) = ?), 0) as spent
		FROM budgets b
		JOIN categories c ON b.category_id = c.id
		WHERE (b.user_id = ? OR b.user_id IS NULL) AND b.month_year = ?
	`, userID, monthYear, userID, monthYear)

	if err != nil {
		return err
	}
	defer rows.Close()

	// Set CSV headers
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=fintrack_budget_%s.csv", monthYear))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Category", "Budget Limit", "Spent", "Remaining", "Percentage"})

	// Write budget data
	for rows.Next() {
		var categoryName string
		var limit, spent int

		if err := rows.Scan(&categoryName, &limit, &spent); err != nil {
			continue
		}

		remaining := limit - spent
		percentage := 0
		if limit > 0 {
			percentage = (spent * 100) / limit
		}

		writer.Write([]string{
			categoryName,
			fmt.Sprintf("Rp %s", formatRupiah(limit)),
			fmt.Sprintf("Rp %s", formatRupiah(spent)),
			fmt.Sprintf("Rp %s", formatRupiah(remaining)),
			fmt.Sprintf("%d%%", percentage),
		})
	}

	return nil
}

// ExportSummaryCSV exports financial summary to CSV
func (e *ExportService) ExportSummaryCSV(c *gin.Context, userID int, startDate, endDate time.Time) error {
	// Get income and expense totals
	var totalIncome, totalExpense int
	
	db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) 
		FROM transactions 
		WHERE user_id = ? AND type = 'income' AND date BETWEEN ? AND ?
	`, userID, startDate, endDate).Scan(&totalIncome)
	
	db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) 
		FROM transactions 
		WHERE user_id = ? AND type = 'expense' AND date BETWEEN ? AND ?
	`, userID, startDate, endDate).Scan(&totalExpense)

	// Get category breakdown
	rows, err := db.Query(`
		SELECT c.name, c.type, COALESCE(SUM(t.amount), 0) as total
		FROM categories c
		LEFT JOIN transactions t ON c.id = t.category_id 
			AND t.user_id = ? 
			AND t.date BETWEEN ? AND ?
		GROUP BY c.id, c.name, c.type
		ORDER BY c.type, total DESC
	`, userID, startDate, endDate)

	if err != nil {
		return err
	}
	defer rows.Close()

	// Set CSV headers
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=fintrack_summary_%s_to_%s.csv",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write summary header
	writer.Write([]string{"FinTrack Financial Summary"})
	writer.Write([]string{"Period", fmt.Sprintf("%s to %s", startDate.Format("02/01/2006"), endDate.Format("02/01/2006"))})
	writer.Write([]string{})

	// Write totals
	writer.Write([]string{"Summary"})
	writer.Write([]string{"Total Income", fmt.Sprintf("Rp %s", formatRupiah(totalIncome))})
	writer.Write([]string{"Total Expense", fmt.Sprintf("Rp %s", formatRupiah(totalExpense))})
	writer.Write([]string{"Balance", fmt.Sprintf("Rp %s", formatRupiah(totalIncome-totalExpense))})
	writer.Write([]string{})

	// Write category breakdown
	writer.Write([]string{"Category Breakdown"})
	writer.Write([]string{"Category", "Type", "Amount", "Percentage"})

	total := totalIncome + totalExpense
	for rows.Next() {
		var categoryName, categoryType string
		var amount int

		if err := rows.Scan(&categoryName, &categoryType, &amount); err != nil {
			continue
		}

		if amount == 0 {
			continue
		}

		typeDisplay := "Pemasukan"
		if categoryType == "expense" {
			typeDisplay = "Pengeluaran"
		}

		percentage := 0
		if total > 0 {
			percentage = (amount * 100) / total
		}

		writer.Write([]string{
			categoryName,
			typeDisplay,
			fmt.Sprintf("Rp %s", formatRupiah(amount)),
			fmt.Sprintf("%d%%", percentage),
		})
	}

	return nil
}

// BackupData represents the full export structure
type BackupData struct {
	Version      int                      `json:"version"`
	ExportedAt   string                   `json:"exported_at"`
	Transactions []map[string]interface{} `json:"transactions"`
	Savings      []map[string]interface{} `json:"savings"`
	SavingsTx    []map[string]interface{} `json:"savings_transactions"`
	Budgets      []map[string]interface{} `json:"budgets"`
}

// ExportBackupJSON exports all user data as JSON
func (e *ExportService) ExportBackupJSON(c *gin.Context, userID int) error {
	backup := BackupData{
		Version:    1,
		ExportedAt: time.Now().Format("2006-01-02T15:04:05"),
	}

	// Transactions
	rows, err := db.Query(`
		SELECT id, type, amount, category_id, COALESCE(note,''), date, COALESCE(receipt_path,'')
		FROM transactions WHERE user_id = ? ORDER BY date ASC
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, amount, categoryID int
			var txType, note, receipt string
			var date time.Time
			rows.Scan(&id, &txType, &amount, &categoryID, &note, &date, &receipt)
			backup.Transactions = append(backup.Transactions, map[string]interface{}{
				"id": id, "type": txType, "amount": amount,
				"category_id": categoryID, "note": note,
				"date": date.Format("2006-01-02T15:04:05"), "receipt_path": receipt,
			})
		}
	}

	// Savings
	savRows, err := db.Query(`
		SELECT id, name, type, icon, color, target_amount, current_amount,
		       COALESCE(deadline,''), COALESCE(description,''), is_completed, created_at
		FROM savings WHERE user_id = ? ORDER BY created_at ASC
	`, userID)
	if err == nil {
		defer savRows.Close()
		for savRows.Next() {
			var id, target, current, completed int
			var name, sType, icon, color, deadline, desc string
			var createdAt time.Time
			savRows.Scan(&id, &name, &sType, &icon, &color, &target, &current, &deadline, &desc, &completed, &createdAt)
			backup.Savings = append(backup.Savings, map[string]interface{}{
				"id": id, "name": name, "type": sType, "icon": icon, "color": color,
				"target_amount": target, "current_amount": current, "deadline": deadline,
				"description": desc, "is_completed": completed,
				"created_at": createdAt.Format("2006-01-02T15:04:05"),
			})
		}
	}

	// Savings transactions
	stRows, err := db.Query(`
		SELECT id, savings_id, type, amount, COALESCE(note,''), date, COALESCE(linked_transaction_id,0)
		FROM savings_transactions WHERE user_id = ? ORDER BY date ASC
	`, userID)
	if err == nil {
		defer stRows.Close()
		for stRows.Next() {
			var id, savingsID, amount, linkedID int
			var txType, note string
			var date time.Time
			stRows.Scan(&id, &savingsID, &txType, &amount, &note, &date, &linkedID)
			backup.SavingsTx = append(backup.SavingsTx, map[string]interface{}{
				"id": id, "savings_id": savingsID, "type": txType, "amount": amount,
				"note": note, "date": date.Format("2006-01-02T15:04:05"),
				"linked_transaction_id": linkedID,
			})
		}
	}

	// Budgets
	bRows, err := db.Query(`
		SELECT id, category_id, amount_limit, month_year FROM budgets WHERE user_id = ?
	`, userID)
	if err == nil {
		defer bRows.Close()
		for bRows.Next() {
			var id, catID, limit int
			var monthYear string
			bRows.Scan(&id, &catID, &limit, &monthYear)
			backup.Budgets = append(backup.Budgets, map[string]interface{}{
				"id": id, "category_id": catID, "amount_limit": limit, "month_year": monthYear,
			})
		}
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=fintrack_backup_%s.json", time.Now().Format("2006-01-02")))
	c.JSON(200, backup)
	return nil
}

// ImportBackupJSON imports data from a JSON backup file
func (e *ExportService) ImportBackupJSON(file io.Reader, userID int) (map[string]interface{}, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file")
	}

	var backup BackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("format file tidak valid")
	}

	if backup.Version != 1 {
		return nil, fmt.Errorf("versi backup tidak didukung")
	}

	txImported, savImported, budgetImported := 0, 0, 0

	// Import transactions
	for _, tx := range backup.Transactions {
		amount := int(toFloat(tx["amount"]))
		catID := int(toFloat(tx["category_id"]))
		note, _ := tx["note"].(string)
		txType, _ := tx["type"].(string)
		dateStr, _ := tx["date"].(string)
		receipt, _ := tx["receipt_path"].(string)
		date, _ := time.Parse("2006-01-02T15:04:05", dateStr)
		if date.IsZero() {
			date = time.Now()
		}
		_, err := db.Exec(`
			INSERT INTO transactions (user_id, type, amount, category_id, note, date, receipt_path)
			VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''))
		`, userID, txType, amount, catID, note, date, receipt)
		if err == nil {
			txImported++
		}
	}

	// Import savings
	oldNewSavingsID := map[int]int64{}
	for _, s := range backup.Savings {
		oldID := int(toFloat(s["id"]))
		name, _ := s["name"].(string)
		sType, _ := s["type"].(string)
		icon, _ := s["icon"].(string)
		color, _ := s["color"].(string)
		target := int(toFloat(s["target_amount"]))
		current := int(toFloat(s["current_amount"]))
		deadline, _ := s["deadline"].(string)
		desc, _ := s["description"].(string)
		completed := int(toFloat(s["is_completed"]))
		res, err := db.Exec(`
			INSERT INTO savings (user_id, name, type, icon, color, target_amount, current_amount,
			deadline, description, is_completed)
			VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?)
		`, userID, name, sType, icon, color, target, current, deadline, desc, completed)
		if err == nil {
			newID, _ := res.LastInsertId()
			oldNewSavingsID[oldID] = newID
			savImported++
		}
	}

	// Import savings transactions
	for _, st := range backup.SavingsTx {
		oldSavID := int(toFloat(st["savings_id"]))
		newSavID, ok := oldNewSavingsID[oldSavID]
		if !ok {
			continue
		}
		txType, _ := st["type"].(string)
		amount := int(toFloat(st["amount"]))
		note, _ := st["note"].(string)
		dateStr, _ := st["date"].(string)
		date, _ := time.Parse("2006-01-02T15:04:05", dateStr)
		if date.IsZero() {
			date = time.Now()
		}
		db.Exec(`
			INSERT INTO savings_transactions (savings_id, user_id, type, amount, note, date)
			VALUES (?, ?, ?, ?, NULLIF(?, ''), ?)
		`, newSavID, userID, txType, amount, note, date)
	}

	// Import budgets
	for _, b := range backup.Budgets {
		catID := int(toFloat(b["category_id"]))
		limit := int(toFloat(b["amount_limit"]))
		monthYear, _ := b["month_year"].(string)
		_, err := db.Exec(`
			INSERT OR IGNORE INTO budgets (user_id, category_id, amount_limit, month_year)
			VALUES (?, ?, ?, ?)
		`, userID, catID, limit, monthYear)
		if err == nil {
			budgetImported++
		}
	}

	return map[string]interface{}{
		"message":         "Import berhasil",
		"tx_imported":     txImported,
		"savings_imported": savImported,
		"budget_imported": budgetImported,
	}, nil
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

// formatRupiah formats integer to rupiah string with thousand separators
func formatRupiah(amount int) string {
	if amount < 0 {
		return fmt.Sprintf("-%s", formatRupiah(-amount))
	}

	str := strconv.Itoa(amount)
	n := len(str)

	if n <= 3 {
		return str
	}

	result := ""
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}

	return result
}

func getAvatarURL(u *User) string {
	if u.Avatar != "" {
		return u.Avatar
	}
	return "https://ui-avatars.com/api/?name=" + u.Name + "&background=c3f545&color=01381b&rounded=true&bold=true"
}
