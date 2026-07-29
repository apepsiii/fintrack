package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

type Transaction struct {
	ID              int       `json:"id"`
	Type            string    `json:"type"`
	Amount          int       `json:"amount"`
	FormattedAmount string    `json:"formatted_amount"`
	CategoryID      int       `json:"category_id"`
	Date            time.Time `json:"date"`
	Category        string    `json:"category"`
	Icon            string    `json:"icon"`
	FormattedDate   string    `json:"formatted_date"`
	Note            string    `json:"note"`
	ReceiptPath     string    `json:"receipt_path"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Icon string `json:"icon"`
}

type CategoryStat struct {
	CategoryName    string
	Icon            string
	Amount          int
	FormattedAmount string
	Percentage      int
}

type BudgetStat struct {
	CategoryName    string
	Icon            string
	Limit           int
	Spent           int
	FormattedLimit  string
	FormattedSpent  string
	FormattedSisa   string
	Percentage      int
	StatusColor     string
}

type BudgetDisplay struct {
	ID             int
	CategoryID     int
	CategoryName   string
	Icon           string
	Limit          int
	Spent          int
	Remaining      int
	Percentage     int
	StatusColor    string
	StatusText     string
	FormattedLimit string
	FormattedSpent string
	FormattedSisa  string
	MonthYear      string
}

var db *sql.DB

func initDB() {
	var err error
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./finance.db"
	}
	
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal("Gagal membuka database:", err)
	}

	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		icon TEXT NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		type TEXT NOT NULL,
		amount INTEGER NOT NULL,
		category_id INTEGER,
		note TEXT,
		date DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);
	
	CREATE TABLE IF NOT EXISTS budgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		category_id INTEGER NOT NULL,
		amount_limit INTEGER NOT NULL,
		month_year TEXT NOT NULL,
		UNIQUE(user_id, category_id, month_year),
		FOREIGN KEY(user_id) REFERENCES users(id),
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);
	`
	_, err = db.Exec(createTablesSQL)
	if err != nil {
		log.Fatal("Gagal membuat tabel:", err)
	}
	
	// Run migrations
	if err := RunMigrations(db); err != nil {
		log.Printf("Warning: Migration error: %v", err)
	}
	
	// Seed data
	seedCategories()
	
	// Create default admin if no users exist
	if err := CreateDefaultAdmin(); err != nil {
		log.Printf("Warning: Failed to create default admin: %v", err)
	}
}

func seedCategories() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if count > 0 {
		return
	}

	categories := []Category{
		{Name: "Makan", Type: "expense", Icon: "ph-hamburger"},
		{Name: "Transport", Type: "expense", Icon: "ph-car"},
		{Name: "Tagihan", Type: "expense", Icon: "ph-lightning"},
		{Name: "Gaji", Type: "income", Icon: "ph-money"},
	}

	for _, cat := range categories {
		db.Exec("INSERT INTO categories (name, type, icon) VALUES (?, ?, ?)", cat.Name, cat.Type, cat.Icon)
	}

	db.Exec("INSERT OR IGNORE INTO budgets (category_id, amount_limit) VALUES (1, 1500000)")
	db.Exec("INSERT OR IGNORE INTO budgets (category_id, amount_limit) VALUES (2, 500000)")
}

func prevMonthStr(monthYear string) string {
	t, _ := time.Parse("2006-01", monthYear)
	return t.AddDate(0, -1, 0).Format("2006-01")
}

func nextMonthStr(monthYear string) string {
	t, _ := time.Parse("2006-01", monthYear)
	return t.AddDate(0, 1, 0).Format("2006-01")
}

func formatMonthLabel(monthYear string) string {
	t, _ := time.Parse("2006-01", monthYear)
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return months[t.Month()] + " " + strconv.Itoa(t.Year())
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	initDB()
	defer db.Close()

	// Initialize OCR service
	ocrService, err := NewOCRService()
	if err != nil {
		log.Printf("OCR service initialization warning: %v", err)
	}
	if ocrService != nil {
		defer ocrService.Close()
	}

	// Set Gin mode from env
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	
	// Serve static files (PWA assets)
	router.Static("/static", "./static")

	// Auth service
	authService := NewAuthService()

	// Public routes (no auth required)
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})

	router.POST("/login", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")

		// Authenticate user
		user, err := AuthenticateUser(email, password)
		if err != nil {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{
				"error": "Email atau password salah",
			})
			return
		}

		// Generate JWT token
		token, err := authService.GenerateToken(user)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "login.html", gin.H{
				"error": "Terjadi kesalahan server",
			})
			return
		}

		// Set token as HTTP-only cookie
		c.SetCookie(
			"auth_token",
			token,
			86400,    // 24 hours
			"/",
			"",
			false,    // Set to true in production with HTTPS
			true,     // HTTP-only
		)

		c.Redirect(http.StatusFound, "/")
	})

	router.POST("/register", func(c *gin.Context) {
		name := c.PostForm("name")
		email := c.PostForm("email")
		password := c.PostForm("password")

		// Validate input
		if name == "" || email == "" || password == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Semua field harus diisi",
			})
			return
		}

		if len(password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Password minimal 6 karakter",
			})
			return
		}

		// Register user
		user, err := RegisterUser(name, email, password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Generate JWT token
		token, err := authService.GenerateToken(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Terjadi kesalahan server",
			})
			return
		}

		// Set token as HTTP-only cookie
		c.SetCookie(
			"auth_token",
			token,
			86400,
			"/",
			"",
			false,
			true,
		)

		c.JSON(http.StatusOK, gin.H{
			"message": "Registrasi berhasil",
			"user":    user,
		})
	})

	router.POST("/logout", func(c *gin.Context) {
		c.SetCookie("auth_token", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/login")
	})

	RegisterForgotPasswordRoutes(router)

	// Protected routes (require authentication)
	protected := router.Group("/")
	protected.Use(AuthMiddleware())

	protected.GET("/", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		user, _ := GetCurrentUser(c)

		var totalIncome int
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'income' AND user_id = ?", userID).Scan(&totalIncome)

		var totalExpense int
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'expense' AND user_id = ?", userID).Scan(&totalExpense)

		balance := totalIncome - totalExpense

		var monthlyExpense int
		db.QueryRow(`
			SELECT COALESCE(SUM(amount), 0) 
			FROM transactions 
			WHERE type = 'expense' AND user_id = ? AND substr(date, 1, 7) = substr(date('now'), 1, 7)
		`, userID).Scan(&monthlyExpense)

		rows, err := db.Query(`
			SELECT t.id, t.type, t.amount, t.date, c.name, c.icon, COALESCE(t.note, '')
			FROM transactions t
			JOIN categories c ON t.category_id = c.id
			WHERE t.user_id = ?
			ORDER BY t.date DESC LIMIT 5
		`, userID)

		var transactions []Transaction
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var trx Transaction
				rows.Scan(&trx.ID, &trx.Type, &trx.Amount, &trx.Date, &trx.Category, &trx.Icon, &trx.Note)
				trx.FormattedDate = trx.Date.Format("02 Jan 2006, 15:04")
				trx.FormattedAmount = formatRupiah(trx.Amount)
				transactions = append(transactions, trx)
			}
		}

		// Top savings goals for dashboard widget
		savRows, err2 := db.Query(`
			SELECT id, name, type, icon, color, target_amount, current_amount,
			       COALESCE(deadline,''), is_completed
			FROM savings
			WHERE user_id = ? AND is_completed = 0
			ORDER BY created_at DESC LIMIT 3
		`, userID)

		var topSavings []Savings
		if err2 == nil {
			defer savRows.Close()
			for savRows.Next() {
				var s Savings
				var isCompleted int
				savRows.Scan(&s.ID, &s.Name, &s.Type, &s.Icon, &s.Color,
					&s.TargetAmount, &s.CurrentAmount, &s.Deadline, &isCompleted)
				s.IsCompleted = isCompleted == 1
				computeSavings(&s)
				topSavings = append(topSavings, s)
			}
		}

		// Top budget categories for dashboard widget
		now := time.Now()
		monthYear := now.Format("2006-01")
		budgetRows, _ := db.Query(`
			SELECT c.name, c.icon,
				   COALESCE(b.amount_limit, 0),
				   COALESCE((SELECT SUM(amount) FROM transactions
				             WHERE category_id = c.id AND user_id = ?
				             AND type = 'expense' AND substr(date,1,7) = ?), 0) as spent
			FROM categories c
			LEFT JOIN budgets b ON b.category_id = c.id AND b.user_id = ? AND b.month_year = ?
			WHERE c.type = 'expense' AND b.amount_limit > 0
			ORDER BY spent DESC LIMIT 3
		`, userID, monthYear, userID, monthYear)

		type BudgetWidget struct {
			Name       string
			Icon       string
			Limit      int
			Spent      int
			Percentage int
			StatusColor string
			FormattedSpent string
			FormattedLimit string
		}
		var topBudgets []BudgetWidget
		if budgetRows != nil {
			defer budgetRows.Close()
			for budgetRows.Next() {
				var b BudgetWidget
				budgetRows.Scan(&b.Name, &b.Icon, &b.Limit, &b.Spent)
				if b.Limit > 0 {
					b.Percentage = b.Spent * 100 / b.Limit
					if b.Percentage > 100 { b.Percentage = 100 }
				}
				if b.Percentage >= 100 { b.StatusColor = "bg-red-500" } else if b.Percentage >= 80 { b.StatusColor = "bg-yellow-400" } else { b.StatusColor = "bg-brand-limeDark" }
				b.FormattedSpent = formatRupiah(b.Spent)
				b.FormattedLimit = formatRupiah(b.Limit)
				topBudgets = append(topBudgets, b)
			}
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Balance":        formatRupiah(balance),
			"MonthlyExpense": formatRupiah(monthlyExpense),
			"Transactions":   transactions,
			"User": gin.H{
				"Name":   user.Name,
				"Avatar": getAvatarURL(user),
			},
			"TopSavings":  topSavings,
			"TopBudgets":  topBudgets,
		})
	})

	protected.POST("/transaction", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		trxType := c.PostForm("type")
		amount := c.PostForm("amount")
		categoryID := c.PostForm("category_id")
		note := c.PostForm("note")
		dateStr := c.PostForm("date")

		var receiptPath string
		file, header, err := c.Request.FormFile("receipt_image")
		if err == nil && header != nil {
			defer file.Close()
			uploadDir := "./static/uploads"
			os.MkdirAll(uploadDir, 0755)
			ext := filepath.Ext(header.Filename)
			if ext == "" {
				ext = ".jpg"
			}
			filename := fmt.Sprintf("receipt_%d_%d%s", userID, nowWIB().UnixNano(), ext)
			destPath := filepath.Join(uploadDir, filename)
			out, err2 := os.Create(destPath)
			if err2 == nil {
				defer out.Close()
				io.Copy(out, file)
				receiptPath = "/static/uploads/" + filename
			}
		}

		if amount != "" && categoryID != "" {
			var parsedDate time.Time
			if dateStr != "" {
				parsedDate, err = time.Parse("2006-01-02T15:04", dateStr)
				if err != nil {
					parsedDate = nowWIB()
				} else {
					parsedDate = parsedDate.In(wib)
				}
			} else {
				parsedDate = nowWIB()
			}

			// Validasi saldo untuk expense
			if trxType == "expense" {
				var totalIncome, totalExpense int
				db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='income'", userID).Scan(&totalIncome)
				db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='expense'", userID).Scan(&totalExpense)
				amountInt, _ := strconv.Atoi(amount)
				saldo := totalIncome - totalExpense
				if saldo < amountInt {
					c.Redirect(http.StatusFound, "/?error=saldo_kurang")
					return
				}
			}

			db.Exec(`INSERT INTO transactions (user_id, type, amount, category_id, note, date, receipt_path)
				VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
				userID, trxType, amount, categoryID, note, parsedDate.Format("2006-01-02 15:04:05"), receiptPath)
		}
		c.Redirect(http.StatusFound, "/?saved=1")
	})

	protected.GET("/stats", func(c *gin.Context) {
		userID := GetCurrentUserID(c)

		filterMonth := c.DefaultQuery("month", time.Now().Format("2006-01"))

		var totalExpense int
		db.QueryRow(`
			SELECT COALESCE(SUM(amount), 0) 
			FROM transactions 
			WHERE type = 'expense' AND user_id = ? AND substr(date, 1, 7) = ?
		`, userID, filterMonth).Scan(&totalExpense)

		var totalIncome int
		db.QueryRow(`
			SELECT COALESCE(SUM(amount), 0) 
			FROM transactions 
			WHERE type = 'income' AND user_id = ? AND substr(date, 1, 7) = ?
		`, userID, filterMonth).Scan(&totalIncome)

		rows, err := db.Query(`
			SELECT c.name, c.icon, COALESCE(SUM(t.amount), 0) as total
			FROM categories c
			LEFT JOIN transactions t ON c.id = t.category_id 
				AND t.type = 'expense'
				AND t.user_id = ?
				AND substr(t.date, 1, 7) = ?
			WHERE c.type = 'expense'
			GROUP BY c.id
			ORDER BY total DESC
		`, userID, filterMonth)

		var stats []CategoryStat
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var stat CategoryStat
				rows.Scan(&stat.CategoryName, &stat.Icon, &stat.Amount)
				if totalExpense > 0 {
					stat.Percentage = (stat.Amount * 100) / totalExpense
				}
				stat.FormattedAmount = formatRupiah(stat.Amount)
				if stat.Amount > 0 {
					stats = append(stats, stat)
				}
			}
		}

		incomeRows, err2 := db.Query(`
			SELECT c.name, c.icon, COALESCE(SUM(t.amount), 0) as total
			FROM categories c
			LEFT JOIN transactions t ON c.id = t.category_id 
				AND t.type = 'income'
				AND t.user_id = ?
				AND substr(t.date, 1, 7) = ?
			WHERE c.type = 'income'
			GROUP BY c.id
			ORDER BY total DESC
		`, userID, filterMonth)

		var incomeStats []CategoryStat
		if err2 == nil {
			defer incomeRows.Close()
			for incomeRows.Next() {
				var stat CategoryStat
				incomeRows.Scan(&stat.CategoryName, &stat.Icon, &stat.Amount)
				if totalIncome > 0 {
					stat.Percentage = (stat.Amount * 100) / totalIncome
				}
				stat.FormattedAmount = formatRupiah(stat.Amount)
				if stat.Amount > 0 {
					incomeStats = append(incomeStats, stat)
				}
			}
		}

		netBalance := totalIncome - totalExpense
		netSign := "+"
		if netBalance < 0 {
			netSign = "-"
			netBalance = -netBalance
		}

		// Tren 6 bulan terakhir
		type MonthTrend struct {
			Month          string
			Label          string
			Income         int
			Expense        int
			FormattedIncome  string
			FormattedExpense string
		}
		var trends []MonthTrend
		now := time.Now().In(wib)
		monthNames := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agt", "Sep", "Okt", "Nov", "Des"}
		for i := 5; i >= 0; i-- {
			t := now.AddDate(0, -i, 0)
			m := t.Format("2006-01")
			var inc, exp int
			db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='income' AND substr(date,1,7)=?", userID, m).Scan(&inc)
			db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions WHERE user_id=? AND type='expense' AND substr(date,1,7)=?", userID, m).Scan(&exp)
			label := monthNames[t.Month()] + " " + strconv.Itoa(t.Year())[2:]
			trends = append(trends, MonthTrend{
				Month: m, Label: label,
				Income: inc, Expense: exp,
				FormattedIncome: formatRupiah(inc), FormattedExpense: formatRupiah(exp),
			})
		}

		// Max value for chart scaling
		maxVal := 1
		for _, tr := range trends {
			if tr.Income > maxVal { maxVal = tr.Income }
			if tr.Expense > maxVal { maxVal = tr.Expense }
		}

		type MonthTrendDisplay struct {
			MonthTrend
			IncomeH  int
			ExpenseH int
		}
		var trendsDisplay []MonthTrendDisplay
		for _, tr := range trends {
			ih := tr.Income * 80 / maxVal
			eh := tr.Expense * 80 / maxVal
			if ih < 2 { ih = 2 }
			if eh < 2 { eh = 2 }
			trendsDisplay = append(trendsDisplay, MonthTrendDisplay{tr, ih, eh})
		}

		c.HTML(http.StatusOK, "stats.html", gin.H{
			"TotalExpense":    formatRupiah(totalExpense),
			"TotalExpenseRaw": totalExpense,
			"TotalIncome":     formatRupiah(totalIncome),
			"TotalIncomeRaw":  totalIncome,
			"NetBalance":      formatRupiah(netBalance),
			"NetSign":         netSign,
			"Stats":           stats,
			"IncomeStats":     incomeStats,
			"FilterMonth":     filterMonth,
			"Trends":          trendsDisplay,
		})
	})

	protected.GET("/targets", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		monthYear := c.DefaultQuery("month", time.Now().Format("2006-01"))

		prevMonth := prevMonthStr(monthYear)
		nextMonth := nextMonthStr(monthYear)
		monthLabel := formatMonthLabel(monthYear)

		rows, err := db.Query(`
			SELECT c.id, c.name, c.icon,
				   COALESCE(b.amount_limit, 0) as budget_limit,
				   COALESCE((SELECT SUM(amount) FROM transactions
					         WHERE category_id = c.id AND user_id = ?
					         AND type = 'expense'
					         AND substr(date, 1, 7) = ?), 0) as spent
			FROM categories c
			LEFT JOIN budgets b ON b.category_id = c.id AND b.user_id = ? AND b.month_year = ?
			WHERE c.type = 'expense'
			  AND c.name NOT IN ('Transfer ke Tabungan', 'Tarik dari Tabungan')
			  AND (c.user_id IS NULL OR c.user_id = ?)
			ORDER BY c.name
		`, userID, monthYear, userID, monthYear, userID)

		var budgets []BudgetDisplay
		var totalLimit, totalSpent, totalRemaining int
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var b BudgetDisplay
				rows.Scan(&b.CategoryID, &b.CategoryName, &b.Icon, &b.Limit, &b.Spent)

				if b.Limit > 0 {
					b.Percentage = (b.Spent * 100) / b.Limit
				}
				if b.Percentage >= 100 {
					b.Percentage = 100
					b.StatusColor = "bg-red-500"
					b.StatusText = "Over budget!"
				} else if b.Percentage >= 80 {
					b.StatusColor = "bg-yellow-400"
					b.StatusText = "Hampir habis"
				} else if b.Percentage >= 50 {
					b.StatusColor = "bg-brand-lime"
					b.StatusText = "On track"
				} else {
					b.StatusColor = "bg-brand-limeDark"
					b.StatusText = "Aman"
				}

				b.Remaining = b.Limit - b.Spent
				if b.Remaining < 0 {
					b.Remaining = 0
				}
				b.FormattedLimit = formatRupiah(b.Limit)
				b.FormattedSpent = formatRupiah(b.Spent)
				b.FormattedSisa = formatRupiah(b.Remaining)

				totalLimit += b.Limit
				totalSpent += b.Spent
				totalRemaining += b.Remaining
				budgets = append(budgets, b)
			}
		}

		totalPct := 0
		if totalLimit > 0 {
			totalPct = (totalSpent * 100) / totalLimit
			if totalPct > 100 {
				totalPct = 100
			}
		}

		c.HTML(http.StatusOK, "targets.html", gin.H{
			"Budgets":        budgets,
			"TotalLimit":     formatRupiah(totalLimit),
			"TotalSpent":     formatRupiah(totalSpent),
			"TotalRemaining": formatRupiah(totalRemaining),
			"TotalPct":       totalPct,
			"TotalLimitRaw":  totalLimit,
			"MonthYear":      monthYear,
			"MonthLabel":     monthLabel,
			"PrevMonth":      prevMonth,
			"NextMonth":      nextMonth,
		})
	})

	protected.POST("/api/budget", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		categoryID, _ := strconv.Atoi(c.PostForm("category_id"))
		limit, _ := strconv.Atoi(c.PostForm("amount_limit"))
		monthYear := c.DefaultPostForm("month_year", time.Now().Format("2006-01"))

		if categoryID == 0 || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kategori dan nominal wajib diisi"})
			return
		}

		db.Exec(`INSERT INTO budgets (user_id, category_id, amount_limit, month_year)
			     VALUES (?, ?, ?, ?)
			     ON CONFLICT(user_id, category_id, month_year) DO UPDATE SET amount_limit=?`,
			userID, categoryID, limit, monthYear, limit)

		c.JSON(http.StatusOK, gin.H{"message": "Anggaran berhasil disimpan"})
	})

	protected.DELETE("/api/budget", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		categoryID := c.Query("category_id")
		monthYear := c.DefaultQuery("month_year", time.Now().Format("2006-01"))
		db.Exec("DELETE FROM budgets WHERE user_id=? AND category_id=? AND month_year=?", userID, categoryID, monthYear)
		c.JSON(http.StatusOK, gin.H{"message": "Anggaran dihapus"})
	})

	protected.GET("/history", func(c *gin.Context) {
		userID := GetCurrentUserID(c)

		// Pagination
		page := 1
		if p := c.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		perPage := 20
		offset := (page - 1) * perPage

		// Filters
		filterType := c.DefaultQuery("type", "all")
		filterMonth := c.DefaultQuery("month", "")

		// Build query
		whereClause := "WHERE t.user_id = ?"
		args := []interface{}{userID}

		if filterType == "income" || filterType == "expense" {
			whereClause += " AND t.type = ?"
			args = append(args, filterType)
		}
		if filterMonth != "" {
			whereClause += " AND substr(t.date, 1, 7) = ?"
			args = append(args, filterMonth)
		}

		// Count total
		var totalCount int
		countArgs := make([]interface{}, len(args))
		copy(countArgs, args)
		db.QueryRow("SELECT COUNT(*) FROM transactions t "+whereClause, countArgs...).Scan(&totalCount)

		totalPages := (totalCount + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}

		// Get totals
		var totalIncome, totalExpense int
		db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions t "+whereClause+" AND t.type='income'",
			countArgs...).Scan(&totalIncome)
		db.QueryRow("SELECT COALESCE(SUM(amount),0) FROM transactions t "+whereClause+" AND t.type='expense'",
			countArgs...).Scan(&totalExpense)

		// Get transactions
		queryArgs := make([]interface{}, len(args))
		copy(queryArgs, args)
		queryArgs = append(queryArgs, perPage, offset)

		rows, err := db.Query(`
			SELECT t.id, t.type, t.amount, t.date, c.name, c.icon, COALESCE(t.note,''), COALESCE(t.receipt_path,'')
			FROM transactions t
			JOIN categories c ON t.category_id = c.id
			`+whereClause+`
			ORDER BY t.date DESC
			LIMIT ? OFFSET ?
		`, queryArgs...)

		type HistoryTransaction struct {
			Transaction
			FormattedDay  string
			FormattedTime string
		}

		var transactions []HistoryTransaction
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var trx HistoryTransaction
				rows.Scan(&trx.ID, &trx.Type, &trx.Amount, &trx.Date, &trx.Category, &trx.Icon, &trx.Note, &trx.ReceiptPath)
				trx.FormattedDate = trx.Date.Format("02 Jan 2006, 15:04")
				trx.FormattedDay = trx.Date.Format("02 January 2006")
				trx.FormattedTime = trx.Date.Format("15:04")
				trx.FormattedAmount = formatRupiah(trx.Amount)
				transactions = append(transactions, trx)
			}
		}

		c.HTML(http.StatusOK, "history.html", gin.H{
			"Transactions":  transactions,
			"TotalIncome":   formatRupiah(totalIncome),
			"TotalExpense":  formatRupiah(totalExpense),
			"TotalCount":    totalCount,
			"CurrentPage":   page,
			"TotalPages":    totalPages,
			"HasPrev":       page > 1,
			"HasNext":       page < totalPages,
			"PrevPage":      page - 1,
			"NextPage":      page + 1,
			"FilterType":    filterType,
			"FilterMonth":   filterMonth,
		})
	})

	protected.GET("/api/transaction/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")

		var trx Transaction
		err := db.QueryRow(`
			SELECT t.id, t.type, t.amount, t.date, c.name, c.icon, COALESCE(t.note,''), COALESCE(t.receipt_path,'')
			FROM transactions t
			JOIN categories c ON t.category_id = c.id
			WHERE t.id = ? AND t.user_id = ?
		`, id, userID).Scan(&trx.ID, &trx.Type, &trx.Amount, &trx.Date, &trx.Category, &trx.Icon, &trx.Note, &trx.ReceiptPath)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		trx.FormattedDate = trx.Date.Format("02 Jan 2006, 15:04")
		trx.FormattedAmount = formatRupiah(trx.Amount)

		c.JSON(http.StatusOK, gin.H{
			"id":               trx.ID,
			"type":             trx.Type,
			"amount":           trx.Amount,
			"formatted_amount": trx.FormattedAmount,
			"formatted_date":   trx.FormattedDate,
			"category":         trx.Category,
			"icon":             trx.Icon,
			"note":             trx.Note,
			"receipt_path":     trx.ReceiptPath,
		})
	})

	protected.GET("/profile", func(c *gin.Context) {
		user, _ := GetCurrentUser(c)

		avatar := user.Avatar
		if avatar == "" {
			avatar = "https://ui-avatars.com/api/?name=" + user.Name + "&background=c3f545&color=01381b&rounded=true&bold=true"
		}

		c.HTML(http.StatusOK, "profile.html", gin.H{
			"User": gin.H{
				"Name":     user.Name,
				"Email":    user.Email,
				"Avatar":   avatar,
				"JoinDate": "Bergabung sejak " + user.CreatedAt.Format("Jan 2006"),
			},
		})
	})

	protected.PUT("/api/profile", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		name := c.PostForm("name")
		email := c.PostForm("email")
		newPassword := c.PostForm("new_password")
		currentPassword := c.PostForm("current_password")

		if name == "" || email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan email wajib diisi"})
			return
		}

		var passwordHash string
		err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User tidak ditemukan"})
			return
		}

		authSvc := NewAuthService()
		if !authSvc.CheckPassword(currentPassword, passwordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Password saat ini salah"})
			return
		}

		var emailExists int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? AND id != ?", email, userID).Scan(&emailExists)
		if emailExists > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email sudah digunakan akun lain"})
			return
		}

		if newPassword != "" {
			if len(newPassword) < 6 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Password baru minimal 6 karakter"})
				return
			}
			newHash, err := authSvc.HashPassword(newPassword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
				return
			}
			db.Exec("UPDATE users SET name=?, email=?, password_hash=? WHERE id=?", name, email, newHash, userID)
		} else {
			db.Exec("UPDATE users SET name=?, email=? WHERE id=?", name, email, userID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
	})

	protected.POST("/api/avatar/preset", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		var req struct {
			Index int `json:"index"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		colors := []string{"c3f545", "01381b", "3b82f6", "8b5cf6", "ec4899", "f59e0b", "10b981", "6366f1"}
		if req.Index < 0 || req.Index >= len(colors) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid avatar index"})
			return
		}
		user, _ := GetCurrentUser(c)
		url := fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=%s&color=ffffff&rounded=true&bold=true&size=200", user.Name, colors[req.Index])
		db.Exec("UPDATE users SET avatar=? WHERE id=?", url, userID)
		c.JSON(http.StatusOK, gin.H{"avatar": url, "message": "Avatar berhasil diperbarui"})
	})

	protected.POST("/api/avatar/upload", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		file, err := c.FormFile("avatar")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
			return
		}
		if file.Size > 2*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File maksimal 2MB"})
			return
		}
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("avatar_%d_%d%s", userID, time.Now().Unix(), ext)
		savePath := filepath.Join("static", "avatars", filename)
		os.MkdirAll(filepath.Join("static", "avatars"), 0755)
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
			return
		}
		url := "/static/avatars/" + filename
		db.Exec("UPDATE users SET avatar=? WHERE id=?", url, userID)
		c.JSON(http.StatusOK, gin.H{"avatar": url, "message": "Avatar berhasil diperbarui"})
	})

	protected.POST("/api/ocr", func(c *gin.Context) {
		file, _, err := c.Request.FormFile("receipt")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
			return
		}
		defer file.Close()

		// Use OCR service to extract amount
		amount, err := ocrService.ScanReceipt(file)
		if err != nil {
			log.Printf("OCR error: %v", err)
			// Fallback to mock if OCR fails
			amount = 85000
		}

		// Return HTML response for HTMX
		htmlResponse := `
		<div class="relative flex gap-2 items-center" id="amountInputContainer">
			<div class="relative flex-1">
				<span class="absolute left-5 top-1/2 -translate-y-1/2 text-brand-dark/50 font-bold text-xl">Rp</span>
				<input type="number" name="amount" id="amountInput" value="` + strconv.Itoa(amount) + `" required
					class="w-full bg-brand-lime/20 border border-brand-lime text-brand-dark text-3xl font-bold rounded-2xl pl-14 pr-4 py-5 focus:outline-none focus:ring-4 focus:ring-brand-lime/40 transition">
				<span class="absolute right-4 top-1/2 -translate-y-1/2 text-brand-dark text-[10px] font-bold bg-brand-lime px-2 py-1 rounded-lg shadow-sm animate-pulse">
					<i class="ph ph-check-circle"></i> Hasil Scan
				</span>
			</div>
			<label for="ocrInput" class="w-16 h-[72px] bg-brand-dark text-white rounded-2xl flex flex-col items-center justify-center cursor-pointer hover:bg-[#024a24] transition active:scale-95 shadow-sm">
				<i class="ph ph-camera-rotate text-2xl mb-1 text-brand-lime"></i>
				<span class="text-[9px] font-semibold text-brand-lime">Ulangi</span>
			</label>
			<input type="file" id="ocrInput" name="receipt" accept="image/*" capture="environment" class="hidden"
				hx-post="/api/ocr" hx-encoding="multipart/form-data" hx-target="#amountInputContainer" hx-swap="outerHTML" hx-indicator="#ocrLoading">
		</div>
		`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlResponse))
	})

	// Export endpoints
	exportService := NewExportService()

	protected.GET("/export/transactions/csv", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		startDate := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
		endDate := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		if err := exportService.ExportTransactionsCSV(c, userID, start, end); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export CSV"})
		}
	})

	protected.GET("/export/budget/csv", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		monthYear := c.DefaultQuery("month", time.Now().Format("2006-01"))
		if err := exportService.ExportBudgetCSV(c, userID, monthYear); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export budget CSV"})
		}
	})

	protected.GET("/export/summary/csv", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		startDate := c.DefaultQuery("start", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
		endDate := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		if err := exportService.ExportSummaryCSV(c, userID, start, end); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export summary CSV"})
		}
	})

	// PDF report — render HTML printable page
	protected.GET("/export/pdf", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		startDate := c.DefaultQuery("start", time.Now().Format("2006-01-01"))
		endDate := c.DefaultQuery("end", time.Now().Format("2006-01-02"))
		// default: current month
		now := time.Now()
		startDate = c.DefaultQuery("start", fmt.Sprintf("%d-%02d-01", now.Year(), now.Month()))
		endDate = c.DefaultQuery("end", now.Format("2006-01-02"))

		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		endEOD := end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		var totalIncome, totalExpense, incomeCount, expenseCount int
		db.QueryRow(`SELECT COALESCE(SUM(amount),0), COUNT(*) FROM transactions WHERE user_id=? AND type='income' AND date BETWEEN ? AND ?`, userID, start, endEOD).Scan(&totalIncome, &incomeCount)
		db.QueryRow(`SELECT COALESCE(SUM(amount),0), COUNT(*) FROM transactions WHERE user_id=? AND type='expense' AND date BETWEEN ? AND ?`, userID, start, endEOD).Scan(&totalExpense, &expenseCount)

		// Category breakdown
		catRows, _ := db.Query(`
			SELECT c.name, COALESCE(SUM(t.amount),0) as total
			FROM categories c
			LEFT JOIN transactions t ON c.id=t.category_id AND t.user_id=? AND t.type='expense' AND t.date BETWEEN ? AND ?
			WHERE c.type='expense'
			GROUP BY c.id ORDER BY total DESC
		`, userID, start, endEOD)
		var expenseStats []CategoryStat
		if catRows != nil {
			defer catRows.Close()
			for catRows.Next() {
				var s CategoryStat
				catRows.Scan(&s.CategoryName, &s.Amount)
				if s.Amount == 0 { continue }
				if totalExpense > 0 { s.Percentage = s.Amount * 100 / totalExpense }
				s.FormattedAmount = formatRupiah(s.Amount)
				expenseStats = append(expenseStats, s)
			}
		}

		// Transactions
		txRows, _ := db.Query(`
			SELECT t.id, t.type, t.amount, t.date, c.name, c.icon, COALESCE(t.note,''), COALESCE(t.receipt_path,'')
			FROM transactions t JOIN categories c ON t.category_id=c.id
			WHERE t.user_id=? AND t.date BETWEEN ? AND ?
			ORDER BY t.date DESC LIMIT 100
		`, userID, start, endEOD)
		var transactions []Transaction
		if txRows != nil {
			defer txRows.Close()
			for txRows.Next() {
				var trx Transaction
				txRows.Scan(&trx.ID, &trx.Type, &trx.Amount, &trx.Date, &trx.Category, &trx.Icon, &trx.Note, &trx.ReceiptPath)
				trx.FormattedDate = trx.Date.Format("02 Jan 2006, 15:04")
				trx.FormattedAmount = formatRupiah(trx.Amount)
				transactions = append(transactions, trx)
			}
		}

		// Savings
		savRows, _ := db.Query(`
			SELECT id, name, type, icon, color, target_amount, current_amount, COALESCE(deadline,''), is_completed
			FROM savings WHERE user_id=? ORDER BY is_completed ASC, created_at DESC
		`, userID)
		var savingsList []Savings
		if savRows != nil {
			defer savRows.Close()
			for savRows.Next() {
				var s Savings
				var isCompleted int
				savRows.Scan(&s.ID, &s.Name, &s.Type, &s.Icon, &s.Color, &s.TargetAmount, &s.CurrentAmount, &s.Deadline, &isCompleted)
				s.IsCompleted = isCompleted == 1
				computeSavings(&s)
				savingsList = append(savingsList, s)
			}
		}

		net := totalIncome - totalExpense
		isDeficit := net < 0
		if isDeficit { net = -net }

		c.HTML(http.StatusOK, "export_pdf.html", gin.H{
			"Period":       start.Format("02 Jan 2006") + " — " + end.Format("02 Jan 2006"),
			"GeneratedAt":  now.Format("02 Jan 2006, 15:04"),
			"TotalIncome":  formatRupiah(totalIncome),
			"TotalExpense": formatRupiah(totalExpense),
			"NetBalance":   formatRupiah(net),
			"IsDeficit":    isDeficit,
			"IncomeCount":  incomeCount,
			"ExpenseCount": expenseCount,
			"ExpenseStats": expenseStats,
			"Transactions": transactions,
			"Savings":      savingsList,
		})
	})

	// Backup export — full JSON
	protected.GET("/export/backup", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		if err := exportService.ExportBackupJSON(c, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal export backup"})
		}
	})

	// Backup import — upload JSON
	protected.POST("/import/backup", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		file, _, err := c.Request.FormFile("backup_file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
			return
		}
		defer file.Close()
		result, err := exportService.ImportBackupJSON(file, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	// Transaction management endpoints
	protected.PUT("/api/transaction/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		transactionID := c.Param("id")

		// Verify ownership
		var ownerID int
		err := db.QueryRow("SELECT user_id FROM transactions WHERE id = ?", transactionID).Scan(&ownerID)
		if err != nil || ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}

		trxType := c.PostForm("type")
		amount := c.PostForm("amount")
		categoryID := c.PostForm("category_id")
		note := c.PostForm("note")

		_, err = db.Exec(`
			UPDATE transactions 
			SET type = ?, amount = ?, category_id = ?, note = ?
			WHERE id = ? AND user_id = ?
		`, trxType, amount, categoryID, note, transactionID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Transaction updated successfully"})
	})

	protected.DELETE("/api/transaction/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		transactionID := c.Param("id")

		// Verify ownership
		var ownerID int
		err := db.QueryRow("SELECT user_id FROM transactions WHERE id = ?", transactionID).Scan(&ownerID)
		if err != nil || ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}

		_, err = db.Exec("DELETE FROM transactions WHERE id = ? AND user_id = ?", transactionID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
	})

	// Budget management endpoints
	protected.GET("/api/categories", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, type, icon FROM categories ORDER BY type, name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load categories"})
			return
		}
		defer rows.Close()

		var categories []Category
		for rows.Next() {
			var cat Category
			rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Icon)
			categories = append(categories, cat)
		}

		c.JSON(http.StatusOK, categories)
	})

	protected.GET("/api/budgets", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		
		rows, err := db.Query(`
			SELECT b.id, b.category_id, b.amount_limit, b.month_year, c.name, c.icon
			FROM budgets b
			JOIN categories c ON b.category_id = c.id
			WHERE b.user_id = ? OR b.user_id IS NULL
			ORDER BY b.month_year DESC, c.name
		`, userID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load budgets"})
			return
		}
		defer rows.Close()

		var budgets []map[string]interface{}
		for rows.Next() {
			var id, categoryID, amountLimit int
			var monthYear, categoryName, icon string
			
			rows.Scan(&id, &categoryID, &amountLimit, &monthYear, &categoryName, &icon)
			
			budgets = append(budgets, map[string]interface{}{
				"id":            id,
				"category_id":   categoryID,
				"category_name": categoryName,
				"icon":          icon,
				"amount_limit":  amountLimit,
				"month_year":    monthYear,
			})
		}

		c.JSON(http.StatusOK, budgets)
	})

	protected.GET("/api/budget/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		budgetID := c.Param("id")

		var budget struct {
			ID          int
			CategoryID  int
			AmountLimit int
			MonthYear   string
		}

		err := db.QueryRow(`
			SELECT id, category_id, amount_limit, month_year
			FROM budgets
			WHERE id = ? AND (user_id = ? OR user_id IS NULL)
		`, budgetID, userID).Scan(&budget.ID, &budget.CategoryID, &budget.AmountLimit, &budget.MonthYear)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":           budget.ID,
			"category_id":  budget.CategoryID,
			"amount_limit": budget.AmountLimit,
			"month_year":   budget.MonthYear,
		})
	})

	// Register savings routes
	RegisterSavingsRoutes(protected)
	RegisterRecurringRoutes(protected)
	RegisterCategoryRoutes(protected)
	RegisterAIRoutes(protected)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server berjalan di http://localhost:%s", port)
	router.Run(":" + port)
}
