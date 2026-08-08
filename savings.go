package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var wib = time.FixedZone("WIB", 7*60*60)

func nowWIB() time.Time {
	return time.Now().In(wib)
}

// Savings represents a savings/investment goal
type Savings struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Icon            string    `json:"icon"`
	Color           string    `json:"color"`
	TargetAmount    int       `json:"target_amount"`
	CurrentAmount   int       `json:"current_amount"`
	Deadline        string    `json:"deadline"`
	Description     string    `json:"description"`
	IsCompleted     bool      `json:"is_completed"`
	CreatedAt       time.Time `json:"created_at"`

		// Computed fields
	FormattedTarget       string `json:"formatted_target"`
	FormattedCurrent      string `json:"formatted_current"`
	FormattedSisa         string `json:"formatted_sisa"`
	Percentage            int    `json:"percentage"`
	StatusColor           string `json:"status_color"`
	DaysLeft              int    `json:"days_left"`
	FormattedDeadline     string `json:"formatted_deadline"`
	MonthlyNeeded         int    `json:"monthly_needed"`
	FormattedMonthly      string `json:"formatted_monthly"`
	MonthsLeft            int    `json:"months_left"`
}

// SavingsTransaction represents a deposit/withdrawal
type SavingsTransaction struct {
	ID        int       `json:"id"`
	SavingsID int       `json:"savings_id"`
	UserID    int       `json:"user_id"`
	Type      string    `json:"type"` // deposit / withdraw
	Amount    int       `json:"amount"`
	Note      string    `json:"note"`
	Date      time.Time `json:"date"`

	FormattedAmount string `json:"formatted_amount"`
	FormattedDate   string `json:"formatted_date"`
}

// SavingsTypeOption represents a type choice for UI
type SavingsTypeOption struct {
	Value string
	Label string
	Icon  string
	Color string
}

var savingsTypes = []SavingsTypeOption{
	{Value: "savings", Label: "Tabungan Umum", Icon: "ph-piggy-bank", Color: "#c3f545"},
	{Value: "emergency", Label: "Dana Darurat", Icon: "ph-first-aid-kit", Color: "#ff9800"},
	{Value: "hajj", Label: "Tabungan Haji/Umrah", Icon: "ph-mosque", Color: "#4caf50"},
	{Value: "wedding", Label: "Tabungan Nikah", Icon: "ph-heart", Color: "#e91e63"},
	{Value: "education", Label: "Tabungan Pendidikan", Icon: "ph-graduation-cap", Color: "#2196f3"},
	{Value: "house", Label: "Beli Rumah/KPR", Icon: "ph-house", Color: "#795548"},
	{Value: "vehicle", Label: "Beli Kendaraan", Icon: "ph-car", Color: "#607d8b"},
	{Value: "vacation", Label: "Liburan", Icon: "ph-airplane", Color: "#00bcd4"},
	{Value: "investment", Label: "Investasi", Icon: "ph-trend-up", Color: "#9c27b0"},
	{Value: "stocks", Label: "Saham/Reksa Dana", Icon: "ph-chart-line-up", Color: "#3f51b5"},
	{Value: "gold", Label: "Emas/Logam Mulia", Icon: "ph-coins", Color: "#ffc107"},
	{Value: "crypto", Label: "Kripto", Icon: "ph-currency-circle-dollar", Color: "#ff5722"},
	{Value: "business", Label: "Modal Usaha", Icon: "ph-briefcase", Color: "#009688"},
	{Value: "gadget", Label: "Gadget/Elektronik", Icon: "ph-device-mobile", Color: "#8bc34a"},
	{Value: "other", Label: "Tujuan Lainnya", Icon: "ph-star", Color: "#9e9e9e"},
}

func getSavingsTypeOption(typeVal string) SavingsTypeOption {
	for _, t := range savingsTypes {
		if t.Value == typeVal {
			return t
		}
	}
	return savingsTypes[0]
}

func computeSavings(s *Savings) {
	opt := getSavingsTypeOption(s.Type)
	if s.Icon == "" || s.Icon == "ph-piggy-bank" {
		s.Icon = opt.Icon
	}
	if s.Color == "" || s.Color == "#c3f545" {
		s.Color = opt.Color
	}

	s.FormattedTarget = formatRupiah(s.TargetAmount)
	s.FormattedCurrent = formatRupiah(s.CurrentAmount)

	sisa := s.TargetAmount - s.CurrentAmount
	if sisa < 0 {
		sisa = 0
	}
	s.FormattedSisa = formatRupiah(sisa)

	if s.TargetAmount > 0 {
		s.Percentage = (s.CurrentAmount * 100) / s.TargetAmount
		if s.Percentage > 100 {
			s.Percentage = 100
		}
	}

	if s.Percentage >= 100 {
		s.StatusColor = "bg-brand-lime"
		s.IsCompleted = true
	} else if s.Percentage >= 75 {
		s.StatusColor = "bg-blue-400"
	} else if s.Percentage >= 50 {
		s.StatusColor = "bg-yellow-400"
	} else {
		s.StatusColor = "bg-brand-limeDark"
	}

	if s.Deadline != "" {
		deadline, err := time.Parse("2006-01-02", s.Deadline)
		if err == nil {
			s.DaysLeft = int(time.Until(deadline).Hours() / 24)
			s.FormattedDeadline = deadline.Format("02 Jan 2006")

			sisa := s.TargetAmount - s.CurrentAmount
			if sisa < 0 {
				sisa = 0
			}
			now := time.Now()
			months := (deadline.Year()-now.Year())*12 + int(deadline.Month()-now.Month())
			if deadline.Day() > now.Day() {
				months++
			}
			if months < 1 {
				months = 1
			}
			s.MonthsLeft = months
			s.MonthlyNeeded = sisa / months
			s.FormattedMonthly = formatRupiah(s.MonthlyNeeded)
		}
	}
}

// RegisterSavingsRoutes registers all savings-related routes
func RegisterSavingsRoutes(protected *gin.RouterGroup) {

	// List page
	protected.GET("/savings", func(c *gin.Context) {
		userID := GetCurrentUserID(c)

		rows, err := db.Query(`
			SELECT id, user_id, name, type, icon, color, target_amount, current_amount,
			       COALESCE(deadline,''), COALESCE(description,''), is_completed, created_at
			FROM savings
			WHERE user_id = ?
			ORDER BY is_completed ASC, created_at DESC
		`, userID)

		var savingsList []Savings
		var totalTarget, totalSaved int

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var s Savings
				var isCompleted int
				rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Type, &s.Icon, &s.Color,
					&s.TargetAmount, &s.CurrentAmount, &s.Deadline, &s.Description,
					&isCompleted, &s.CreatedAt)
				s.IsCompleted = isCompleted == 1
				computeSavings(&s)
				totalTarget += s.TargetAmount
				totalSaved += s.CurrentAmount
				savingsList = append(savingsList, s)
			}
		}

		c.HTML(http.StatusOK, "savings.html", gin.H{
			"Savings":      savingsList,
			"TotalTarget":  formatRupiah(totalTarget),
			"TotalSaved":   formatRupiah(totalSaved),
			"TotalCount":   len(savingsList),
			"SavingsTypes": savingsTypes,
		})
	})

	// Detail page
	protected.GET("/savings/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")

		var s Savings
		var isCompleted int
		err := db.QueryRow(`
			SELECT id, user_id, name, type, icon, color, target_amount, current_amount,
			       COALESCE(deadline,''), COALESCE(description,''), is_completed, created_at
			FROM savings WHERE id = ? AND user_id = ?
		`, id, userID).Scan(&s.ID, &s.UserID, &s.Name, &s.Type, &s.Icon, &s.Color,
			&s.TargetAmount, &s.CurrentAmount, &s.Deadline, &s.Description,
			&isCompleted, &s.CreatedAt)

		if err != nil {
			c.Redirect(http.StatusFound, "/savings")
			return
		}
		s.IsCompleted = isCompleted == 1
		computeSavings(&s)

		// Get transactions
		rows, err := db.Query(`
			SELECT id, type, amount, COALESCE(note,''), date
			FROM savings_transactions
			WHERE savings_id = ? AND user_id = ?
			ORDER BY date DESC LIMIT 20
		`, s.ID, userID)

		var transactions []SavingsTransaction
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var t SavingsTransaction
				rows.Scan(&t.ID, &t.Type, &t.Amount, &t.Note, &t.Date)
				t.FormattedAmount = formatRupiah(t.Amount)
				t.FormattedDate = t.Date.Format("02 Jan 2006, 15:04")
				transactions = append(transactions, t)
			}
		}

		// Hitung saldo utama user
		var totalIncome, totalExpense int
		db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='income'", userID).Scan(&totalIncome)
		db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='expense'", userID).Scan(&totalExpense)
		balanceRaw := totalIncome - totalExpense

		c.HTML(http.StatusOK, "savings_detail.html", gin.H{
			"Savings":      s,
			"Transactions": transactions,
			"BalanceRaw":   balanceRaw,
		})
	})

	// Create savings
	protected.POST("/api/savings", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		if !CheckSavingsLimit(c) {
			return
		}
		name := c.PostForm("name")
		savType := c.PostForm("type")
		targetStr := c.PostForm("target_amount")
		deadline := c.PostForm("deadline")
		description := c.PostForm("description")

		if name == "" || targetStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan target wajib diisi"})
			return
		}

		target, err := strconv.Atoi(targetStr)
		if err != nil || target <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target tidak valid"})
			return
		}

		opt := getSavingsTypeOption(savType)

		result, err := db.Exec(`
			INSERT INTO savings (user_id, name, type, icon, color, target_amount, deadline, description)
			VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))
		`, userID, name, savType, opt.Icon, opt.Color, target, deadline, description)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat tabungan"})
			return
		}

		newID, _ := result.LastInsertId()
		c.JSON(http.StatusOK, gin.H{"message": "Tabungan berhasil dibuat", "id": newID})
	})

	// Update savings
	protected.PUT("/api/savings/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		name := c.PostForm("name")
		savType := c.PostForm("type")
		targetStr := c.PostForm("target_amount")
		deadline := c.PostForm("deadline")
		description := c.PostForm("description")

		target, _ := strconv.Atoi(targetStr)
		opt := getSavingsTypeOption(savType)

		_, err := db.Exec(`
			UPDATE savings SET name=?, type=?, icon=?, color=?, target_amount=?,
			deadline=NULLIF(?, ''), description=NULLIF(?, '')
			WHERE id=? AND user_id=?
		`, name, savType, opt.Icon, opt.Color, target, deadline, description, id, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil diperbarui"})
	})

	// Delete savings
	protected.DELETE("/api/savings/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")

		db.Exec("DELETE FROM savings_transactions WHERE savings_id = ? AND user_id = ?", id, userID)
		_, err := db.Exec("DELETE FROM savings WHERE id = ? AND user_id = ?", id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
	})

	// Deposit / Withdraw
	protected.POST("/api/savings/:id/transaction", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		txType := c.PostForm("type") // deposit or withdraw
		amountStr := c.PostForm("amount")
		note := c.PostForm("note")

		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Jumlah tidak valid"})
			return
		}

		// Verify ownership and get savings info
		var currentAmount, targetAmount int
		var savingsName string
		err = db.QueryRow("SELECT current_amount, target_amount, name FROM savings WHERE id = ? AND user_id = ?", id, userID).
			Scan(&currentAmount, &targetAmount, &savingsName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tabungan tidak ditemukan"})
			return
		}

		// Calculate new amount
		newAmount := currentAmount
		if txType == "deposit" {
			// Validasi saldo utama mencukupi
			var totalIncome, totalExpense int
			db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='income'", userID).Scan(&totalIncome)
			db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='expense'", userID).Scan(&totalExpense)
			saldoUtama := totalIncome - totalExpense
			if saldoUtama < amount {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Saldo utama tidak mencukupi (Rp " + formatRupiah(saldoUtama) + " tersedia)"})
				return
			}
			newAmount += amount
		} else if txType == "withdraw" {
			newAmount -= amount
			if newAmount < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Saldo tabungan tidak mencukupi"})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe transaksi tidak valid"})
			return
		}

		// Find linked category IDs — auto-create jika belum ada
		var transferOutCatID, transferInCatID int
		db.QueryRow("SELECT id FROM categories WHERE name = 'Transfer ke Tabungan' LIMIT 1").Scan(&transferOutCatID)
		if transferOutCatID == 0 {
			res, _ := db.Exec("INSERT INTO categories (name, type, icon) VALUES ('Transfer ke Tabungan', 'expense', 'ph-piggy-bank')")
			id64, _ := res.LastInsertId()
			transferOutCatID = int(id64)
		}
		db.QueryRow("SELECT id FROM categories WHERE name = 'Tarik dari Tabungan' LIMIT 1").Scan(&transferInCatID)
		if transferInCatID == 0 {
			res, _ := db.Exec("INSERT INTO categories (name, type, icon) VALUES ('Tarik dari Tabungan', 'income', 'ph-piggy-bank')")
			id64, _ := res.LastInsertId()
			transferInCatID = int(id64)
		}

		// Auto-create transaction in main account
		txNote := savingsName
		if note != "" {
			txNote = savingsName + " - " + note
		}

		var linkedTxID int64
		if txType == "deposit" && transferOutCatID > 0 {
			res, err := db.Exec(`
				INSERT INTO transactions (user_id, type, amount, category_id, note, date)
				VALUES (?, 'expense', ?, ?, ?, ?)
			`, userID, amount, transferOutCatID, txNote, nowWIB().Format("2006-01-02 15:04:05"))
			if err == nil {
				linkedTxID, _ = res.LastInsertId()
			}
		} else if txType == "withdraw" && transferInCatID > 0 {
			res, err := db.Exec(`
				INSERT INTO transactions (user_id, type, amount, category_id, note, date)
				VALUES (?, 'income', ?, ?, ?, ?)
			`, userID, amount, transferInCatID, txNote, nowWIB().Format("2006-01-02 15:04:05"))
			if err == nil {
				linkedTxID, _ = res.LastInsertId()
			}
		}

		// Record savings transaction
		_, err = db.Exec(`
			INSERT INTO savings_transactions (savings_id, user_id, type, amount, note, date, linked_transaction_id)
			VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, NULLIF(?, 0))
		`, id, userID, txType, amount, note, nowWIB().Format("2006-01-02 15:04:05"), linkedTxID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		// Update current amount & completion status
		isCompleted := 0
		if newAmount >= targetAmount {
			isCompleted = 1
		}

		_, err = db.Exec("UPDATE savings SET current_amount = ?, is_completed = ? WHERE id = ? AND user_id = ?",
			newAmount, isCompleted, id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui saldo"})
			return
		}

		msg := "Setoran berhasil, saldo utama berkurang"
		if txType == "withdraw" {
			msg = "Penarikan berhasil, saldo utama bertambah"
		}
		c.JSON(http.StatusOK, gin.H{
			"message":      msg,
			"new_amount":   newAmount,
			"formatted":    formatRupiah(newAmount),
			"is_completed": isCompleted == 1,
		})
	})

	// Get savings types for dropdown
	protected.GET("/api/savings/types", func(c *gin.Context) {
		c.JSON(http.StatusOK, savingsTypes)
	})
}
