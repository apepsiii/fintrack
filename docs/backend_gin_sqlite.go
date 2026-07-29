package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Transaction struct {
	ID         int       `json:"id"`
	Type       string    `json:"type"`
	Amount     int       `json:"amount"`
	CategoryID int       `json:"category_id"`
	Date       time.Time `json:"date"`
	Category   string    `json:"category"`
	Icon       string    `json:"icon"`
	FormattedDate string `json:"formatted_date"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Icon string `json:"icon"`
}

type CategoryStat struct {
	CategoryName string
	Icon         string
	Amount       int
	Percentage   int
}

type BudgetStat struct {
	CategoryName string
	Icon         string
	Limit        int
	Spent        int
	Percentage   int
	StatusColor  string 
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./finance.db")
	if err != nil {
		log.Fatal("Gagal membuka database:", err)
	}

	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		icon TEXT NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		amount INTEGER NOT NULL,
		category_id INTEGER,
		date DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);
	
	CREATE TABLE IF NOT EXISTS budgets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL UNIQUE,
		amount_limit INTEGER NOT NULL,
		FOREIGN KEY(category_id) REFERENCES categories(id)
	);
	`
	_, err = db.Exec(createTablesSQL)
	if err != nil {
		log.Fatal("Gagal membuat tabel:", err)
	}
	seedCategories()
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

func main() {
	initDB()
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})

	router.POST("/login", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/")
	})

	router.GET("/", func(c *gin.Context) {
		var totalIncome int
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'income'").Scan(&totalIncome)

		var totalExpense int
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'expense'").Scan(&totalExpense)

		balance := totalIncome - totalExpense

		var monthlyExpense int
		db.QueryRow(`
			SELECT COALESCE(SUM(amount), 0) 
			FROM transactions 
			WHERE type = 'expense' AND strftime('%Y-%m', date) = strftime('%Y-%m', 'now')
		`).Scan(&monthlyExpense)

		rows, err := db.Query(`
			SELECT t.id, t.type, t.amount, t.date, c.name, c.icon 
			FROM transactions t
			JOIN categories c ON t.category_id = c.id
			ORDER BY t.date DESC LIMIT 5
		`)
		
		var transactions []Transaction
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var trx Transaction
				rows.Scan(&trx.ID, &trx.Type, &trx.Amount, &trx.Date, &trx.Category, &trx.Icon)
				trx.FormattedDate = trx.Date.Format("02 Jan 2006, 15:04")
				transactions = append(transactions, trx)
			}
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"Balance":        balance,
			"MonthlyExpense": monthlyExpense,
			"Transactions":   transactions,
		})
	})

	router.POST("/transaction", func(c *gin.Context) {
		trxType := c.PostForm("type")
		amount := c.PostForm("amount")
		categoryID := c.PostForm("category_id")

		if amount != "" && categoryID != "" {
			db.Exec("INSERT INTO transactions (type, amount, category_id) VALUES (?, ?, ?)", trxType, amount, categoryID)
		}
		c.Redirect(http.StatusFound, "/")
	})

	router.GET("/stats", func(c *gin.Context) {
		var totalExpense int
		db.QueryRow(`
			SELECT COALESCE(SUM(amount), 0) 
			FROM transactions 
			WHERE type = 'expense' AND strftime('%Y-%m', date) = strftime('%Y-%m', 'now')
		`).Scan(&totalExpense)

		rows, err := db.Query(`
			SELECT c.name, c.icon, COALESCE(SUM(t.amount), 0) as total
			FROM categories c
			LEFT JOIN transactions t ON c.id = t.category_id 
				AND t.type = 'expense' 
				AND strftime('%Y-%m', t.date) = strftime('%Y-%m', 'now')
			WHERE c.type = 'expense'
			GROUP BY c.id
			ORDER BY total DESC
		`)
		
		var stats []CategoryStat
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var stat CategoryStat
				rows.Scan(&stat.CategoryName, &stat.Icon, &stat.Amount)
				
				if totalExpense > 0 {
					stat.Percentage = (stat.Amount * 100) / totalExpense
				} else {
					stat.Percentage = 0
				}
				stats = append(stats, stat)
			}
		}

		c.HTML(http.StatusOK, "stats.html", gin.H{
			"TotalExpense": totalExpense,
			"Stats":        stats,
		})
	})

	router.GET("/targets", func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT c.name, c.icon, b.amount_limit,
				   COALESCE((SELECT SUM(amount) FROM transactions WHERE category_id = c.id AND type = 'expense' AND strftime('%Y-%m', date) = strftime('%Y-%m', 'now')), 0) as spent
			FROM budgets b
			JOIN categories c ON b.category_id = c.id
		`)
		
		var budgets []BudgetStat
		var totalLimit, totalSpent int

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var b BudgetStat
				rows.Scan(&b.CategoryName, &b.Icon, &b.Limit, &b.Spent)
				
				if b.Limit > 0 {
					b.Percentage = (b.Spent * 100) / b.Limit
				} else {
					b.Percentage = 0
				}
				
				if b.Percentage >= 100 {
					b.Percentage = 100
					b.StatusColor = "bg-[#ff4d4f]"
				} else if b.Percentage >= 80 {
					b.StatusColor = "bg-yellow-400"
				} else {
					b.StatusColor = "bg-brand-limeDark"
				}

				totalLimit += b.Limit
				totalSpent += b.Spent
				budgets = append(budgets, b)
			}
		}

		c.HTML(http.StatusOK, "targets.html", gin.H{
			"Budgets":    budgets,
			"TotalLimit": totalLimit,
			"TotalSpent": totalSpent,
		})
	})

	router.GET("/profile", func(c *gin.Context) {
		user := gin.H{
			"Name":     "ST Racson",
			"Email":    "racson@fintrack.id",
			"JoinDate": "Bergabung sejak Jan 2026",
		}

		c.HTML(http.StatusOK, "profile.html", gin.H{
			"User": user,
		})
	})

	router.POST("/api/ocr", func(c *gin.Context) {
		time.Sleep(1500 * time.Millisecond)
		detectedAmount := "85000"

		htmlResponse := `
		<div class="relative flex gap-2 items-center" id="amountInputContainer">
			<div class="relative flex-1">
				<span class="absolute left-5 top-1/2 -translate-y-1/2 text-brand-dark/50 font-bold text-xl">Rp</span>
				<input type="number" name="amount" id="amountInput" value="` + detectedAmount + `" required
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

	log.Println("Server berjalan di http://localhost:8080")
	router.Run(":8080")
}