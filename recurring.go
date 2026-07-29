package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RecurringTransaction struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Amount     int    `json:"amount"`
	CategoryID int    `json:"category_id"`
	Category   string `json:"category"`
	Icon       string `json:"icon"`
	Note       string `json:"note"`
	Frequency  string `json:"frequency"`
	NextDate   string `json:"next_date"`
	IsActive   bool   `json:"is_active"`

	FormattedAmount string `json:"formatted_amount"`
	FrequencyLabel  string `json:"frequency_label"`
}

var frequencyLabels = map[string]string{
	"daily":   "Setiap Hari",
	"weekly":  "Setiap Minggu",
	"monthly": "Setiap Bulan",
	"yearly":  "Setiap Tahun",
}

func nextRecurringDate(from time.Time, frequency string) time.Time {
	switch frequency {
	case "daily":
		return from.AddDate(0, 0, 1)
	case "weekly":
		return from.AddDate(0, 0, 7)
	case "monthly":
		return from.AddDate(0, 1, 0)
	case "yearly":
		return from.AddDate(1, 0, 0)
	}
	return from.AddDate(0, 1, 0)
}

func ProcessRecurringTransactions(userID int) int {
	now := nowWIB()
	today := now.Format("2006-01-02")

	rows, err := db.Query(`
		SELECT id, name, type, amount, category_id, COALESCE(note,''), frequency, next_date
		FROM recurring_transactions
		WHERE user_id = ? AND is_active = 1 AND next_date <= ?
	`, userID, today)
	if err != nil {
		return 0
	}
	defer rows.Close()

	count := 0
	var toProcess []RecurringTransaction
	for rows.Next() {
		var r RecurringTransaction
		rows.Scan(&r.ID, &r.Name, &r.Type, &r.Amount, &r.CategoryID, &r.Note, &r.Frequency, &r.NextDate)
		toProcess = append(toProcess, r)
	}

	for _, r := range toProcess {
		db.Exec(`INSERT INTO transactions (user_id, type, amount, category_id, note, date)
			VALUES (?, ?, ?, ?, ?, ?)`,
			userID, r.Type, r.Amount, r.CategoryID, r.Name+": "+r.Note, now.Format("2006-01-02 15:04:05"))

		nextDate := nextRecurringDate(now, r.Frequency)
		db.Exec("UPDATE recurring_transactions SET next_date=? WHERE id=?",
			nextDate.Format("2006-01-02"), r.ID)
		count++
	}
	return count
}

func RegisterRecurringRoutes(protected *gin.RouterGroup) {
	protected.GET("/recurring", func(c *gin.Context) {
		userID := GetCurrentUserID(c)

		ProcessRecurringTransactions(userID)

		rows, err := db.Query(`
			SELECT r.id, r.name, r.type, r.amount, r.category_id, c.name, c.icon,
			       COALESCE(r.note,''), r.frequency, r.next_date, r.is_active
			FROM recurring_transactions r
			JOIN categories c ON r.category_id = c.id
			WHERE r.user_id = ?
			ORDER BY r.is_active DESC, r.next_date ASC
		`, userID)

		var list []RecurringTransaction
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r RecurringTransaction
				var isActive int
				rows.Scan(&r.ID, &r.Name, &r.Type, &r.Amount, &r.CategoryID,
					&r.Category, &r.Icon, &r.Note, &r.Frequency, &r.NextDate, &isActive)
				r.IsActive = isActive == 1
				r.FormattedAmount = formatRupiah(r.Amount)
				r.FrequencyLabel = frequencyLabels[r.Frequency]
				list = append(list, r)
			}
		}

		var categories []Category
		catRows, _ := db.Query("SELECT id, name, type, icon FROM categories ORDER BY type, name")
		if catRows != nil {
			defer catRows.Close()
			for catRows.Next() {
				var cat Category
				catRows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Icon)
				categories = append(categories, cat)
			}
		}

		c.HTML(http.StatusOK, "recurring.html", gin.H{
			"Recurring":  list,
			"Categories": categories,
		})
	})

	protected.POST("/api/recurring", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		name := c.PostForm("name")
		txType := c.PostForm("type")
		amountStr := c.PostForm("amount")
		categoryID := c.PostForm("category_id")
		note := c.PostForm("note")
		frequency := c.DefaultPostForm("frequency", "monthly")
		startDate := c.DefaultPostForm("start_date", nowWIB().Format("2006-01-02"))

		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		_, err = db.Exec(`
			INSERT INTO recurring_transactions (user_id, name, type, amount, category_id, note, frequency, next_date)
			VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?)
		`, userID, name, txType, amount, categoryID, note, frequency, startDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Transaksi berulang berhasil dibuat"})
	})

	protected.PUT("/api/recurring/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		name := c.PostForm("name")
		amountStr := c.PostForm("amount")
		categoryID := c.PostForm("category_id")
		note := c.PostForm("note")
		frequency := c.PostForm("frequency")
		isActiveStr := c.DefaultPostForm("is_active", "1")

		amount, _ := strconv.Atoi(amountStr)
		isActive, _ := strconv.Atoi(isActiveStr)

		db.Exec(`UPDATE recurring_transactions
			SET name=?, amount=?, category_id=?, note=NULLIF(?,?), frequency=?, is_active=?
			WHERE id=? AND user_id=?`,
			name, amount, categoryID, note, "", frequency, isActive, id, userID)
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil diperbarui"})
	})

	protected.DELETE("/api/recurring/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		db.Exec("DELETE FROM recurring_transactions WHERE id=? AND user_id=?", id, userID)
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
	})
}
