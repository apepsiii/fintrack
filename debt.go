package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Debt struct {
	ID                  int    `json:"id"`
	UserID              int    `json:"user_id"`
	Name                string `json:"name"`
	Type                string `json:"type"` // owe = saya berhutang, lend = saya piutang
	TotalAmount         int    `json:"total_amount"`
	PaidAmount          int    `json:"paid_amount"`
	Creditor            string `json:"creditor"`
	DueDate             string `json:"due_date"`
	Note                string `json:"note"`
	IsSettled           bool   `json:"is_settled"`
	LinkedTransactionID int    `json:"linked_transaction_id"`

	Remaining        int    `json:"remaining"`
	Percentage       int    `json:"percentage"`
	FormattedTotal   string `json:"formatted_total"`
	FormattedPaid    string `json:"formatted_paid"`
	FormattedRemain  string `json:"formatted_remaining"`
	FormattedDueDate string `json:"formatted_due_date"`
	DaysLeft         int    `json:"days_left"`
	IsOverdue        bool   `json:"is_overdue"`
	StatusColor      string `json:"status_color"`
}

type DebtPayment struct {
	ID              int       `json:"id"`
	DebtID          int       `json:"debt_id"`
	Amount          int       `json:"amount"`
	Note            string    `json:"note"`
	Date            time.Time `json:"date"`
	FormattedAmount string    `json:"formatted_amount"`
	FormattedDate   string    `json:"formatted_date"`
}

func computeDebt(d *Debt) {
	d.Remaining = d.TotalAmount - d.PaidAmount
	if d.Remaining < 0 {
		d.Remaining = 0
	}
	if d.TotalAmount > 0 {
		d.Percentage = d.PaidAmount * 100 / d.TotalAmount
		if d.Percentage > 100 {
			d.Percentage = 100
		}
	}
	d.FormattedTotal = formatRupiah(d.TotalAmount)
	d.FormattedPaid = formatRupiah(d.PaidAmount)
	d.FormattedRemain = formatRupiah(d.Remaining)

	if d.Percentage >= 100 {
		d.IsSettled = true
		d.StatusColor = "bg-brand-lime"
	} else if d.IsOverdue {
		d.StatusColor = "bg-red-500"
	} else if d.Percentage >= 50 {
		d.StatusColor = "bg-yellow-400"
	} else {
		d.StatusColor = "bg-brand-limeDark"
	}

	if d.DueDate != "" {
		due, err := time.Parse("2006-01-02", d.DueDate)
		if err == nil {
			d.FormattedDueDate = due.Format("02 Jan 2006")
			d.DaysLeft = int(time.Until(due).Hours() / 24)
			d.IsOverdue = d.DaysLeft < 0 && !d.IsSettled
			if d.IsOverdue {
				d.StatusColor = "bg-red-500"
			}
		}
	}
}

func RegisterDebtRoutes(protected *gin.RouterGroup) {

	protected.GET("/debt", func(c *gin.Context) {
		userID := GetCurrentUserID(c)

		rows, err := db.Query(`
			SELECT id, name, type, total_amount, paid_amount, creditor,
			       COALESCE(due_date,''), COALESCE(note,''), is_settled,
			       COALESCE(linked_transaction_id,0)
			FROM debts
			WHERE user_id = ?
			ORDER BY is_settled ASC, created_at DESC
		`, userID)

		var debts []Debt
		var totalOwe, totalLend, totalOwePaid, totalLendPaid int

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var d Debt
				var isSettled int
				rows.Scan(&d.ID, &d.Name, &d.Type, &d.TotalAmount, &d.PaidAmount,
					&d.Creditor, &d.DueDate, &d.Note, &isSettled, &d.LinkedTransactionID)
				d.IsSettled = isSettled == 1
				computeDebt(&d)
				if d.Type == "owe" {
					totalOwe += d.TotalAmount
					totalOwePaid += d.PaidAmount
				} else {
					totalLend += d.TotalAmount
					totalLendPaid += d.PaidAmount
				}
				debts = append(debts, d)
			}
		}

		c.HTML(http.StatusOK, "debt.html", gin.H{
			"Debts":        debts,
			"TotalOwe":     formatRupiah(totalOwe - totalOwePaid),
			"TotalLend":    formatRupiah(totalLend - totalLendPaid),
			"TotalOweRaw":  totalOwe - totalOwePaid,
			"TotalLendRaw": totalLend - totalLendPaid,
		})
	})

	protected.GET("/debt/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")

		var d Debt
		var isSettled int
		err := db.QueryRow(`
			SELECT id, name, type, total_amount, paid_amount, creditor,
			       COALESCE(due_date,''), COALESCE(note,''), is_settled,
			       COALESCE(linked_transaction_id,0)
			FROM debts WHERE id = ? AND user_id = ?
		`, id, userID).Scan(&d.ID, &d.Name, &d.Type, &d.TotalAmount, &d.PaidAmount,
			&d.Creditor, &d.DueDate, &d.Note, &isSettled, &d.LinkedTransactionID)

		if err != nil {
			c.Redirect(http.StatusFound, "/debt")
			return
		}
		d.IsSettled = isSettled == 1
		computeDebt(&d)

		rows, _ := db.Query(`
			SELECT id, amount, COALESCE(note,''), date
			FROM debt_payments WHERE debt_id = ? AND user_id = ?
			ORDER BY date DESC LIMIT 20
		`, d.ID, userID)

		var payments []DebtPayment
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var p DebtPayment
				rows.Scan(&p.ID, &p.Amount, &p.Note, &p.Date)
				p.FormattedAmount = formatRupiah(p.Amount)
				p.FormattedDate = p.Date.Format("02 Jan 2006, 15:04")
				payments = append(payments, p)
			}
		}

		c.HTML(http.StatusOK, "debt_detail.html", gin.H{
			"Debt":     d,
			"Payments": payments,
		})
	})

	protected.POST("/api/debt", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		name := c.PostForm("name")
		debtType := c.DefaultPostForm("type", "owe")
		totalStr := c.PostForm("total_amount")
		creditor := c.PostForm("creditor")
		dueDate := c.PostForm("due_date")
		note := c.PostForm("note")
		linkSaldo := c.PostForm("link_saldo") // "yes" = buat transaksi di saldo utama

		total, err := strconv.Atoi(totalStr)
		if err != nil || total <= 0 || name == "" || creditor == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama, nominal, dan pihak terkait wajib diisi"})
			return
		}

		result, err := db.Exec(`
			INSERT INTO debts (user_id, name, type, total_amount, creditor, due_date, note)
			VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))
		`, userID, name, debtType, total, creditor, dueDate, note)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan"})
			return
		}
		debtID, _ := result.LastInsertId()

		// Integrasi saldo utama
		if linkSaldo == "yes" {
			var catName, txType string
			if debtType == "owe" {
				// Saya berhutang = dapat uang = income
				catName = "Hutang Diterima"
				txType = "income"
			} else {
				// Saya piutang = beri uang = expense
				catName = "Piutang Diberikan"
				txType = "expense"
			}

			var catID int
			db.QueryRow("SELECT id FROM categories WHERE name = ? LIMIT 1", catName).Scan(&catID)
			if catID == 0 {
				res, _ := db.Exec("INSERT INTO categories (name, type, icon) VALUES (?, ?, 'ph-hand-coins')", catName, txType)
				if id, err := res.LastInsertId(); err == nil {
					catID = int(id)
				}
			}

			txNote := name + " - " + creditor
			txResult, txErr := db.Exec(`
				INSERT INTO transactions (user_id, type, amount, category_id, note, date)
				VALUES (?, ?, ?, ?, ?, ?)
			`, userID, txType, total, catID, txNote, nowWIB().Format("2006-01-02 15:04:05"))
			if txErr == nil {
				txID, _ := txResult.LastInsertId()
				db.Exec("UPDATE debts SET linked_transaction_id=? WHERE id=?", txID, debtID)
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Berhasil disimpan", "id": debtID})
	})

	protected.PUT("/api/debt/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		name := c.PostForm("name")
		totalStr := c.PostForm("total_amount")
		creditor := c.PostForm("creditor")
		dueDate := c.PostForm("due_date")
		note := c.PostForm("note")

		total, _ := strconv.Atoi(totalStr)
		db.Exec(`UPDATE debts SET name=?, total_amount=?, creditor=?,
			due_date=NULLIF(?,?), note=NULLIF(?,?)
			WHERE id=? AND user_id=?`,
			name, total, creditor, dueDate, "", note, "", id, userID)
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil diperbarui"})
	})

	protected.DELETE("/api/debt/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		db.Exec("DELETE FROM debt_payments WHERE debt_id=? AND user_id=?", id, userID)
		db.Exec("DELETE FROM debts WHERE id=? AND user_id=?", id, userID)
		c.JSON(http.StatusOK, gin.H{"message": "Berhasil dihapus"})
	})

	protected.POST("/api/debt/:id/pay", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		amountStr := c.PostForm("amount")
		note := c.PostForm("note")

		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nominal tidak valid"})
			return
		}

		var currentPaid, totalAmount int
		var debtType, debtName, creditor string
		var linkedTxID int
		err = db.QueryRow(`SELECT paid_amount, total_amount, type, name, creditor, COALESCE(linked_transaction_id,0)
			FROM debts WHERE id=? AND user_id=?`, id, userID).
			Scan(&currentPaid, &totalAmount, &debtType, &debtName, &creditor, &linkedTxID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data tidak ditemukan"})
			return
		}

		newPaid := currentPaid + amount
		if newPaid > totalAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pembayaran melebihi total hutang"})
			return
		}

		isSettled := 0
		if newPaid >= totalAmount {
			isSettled = 1
		}

		db.Exec(`INSERT INTO debt_payments (debt_id, user_id, amount, note, date)
			VALUES (?, ?, ?, NULLIF(?,?), ?)`,
			id, userID, amount, note, "", nowWIB().Format("2006-01-02 15:04:05"))
		db.Exec("UPDATE debts SET paid_amount=?, is_settled=? WHERE id=? AND user_id=?",
			newPaid, isSettled, id, userID)

		// Integrasi saldo: jika debt punya linked_transaction_id, berarti saldo sudah diintegrasikan
		if linkedTxID > 0 {
			var catName, txType string
			if debtType == "owe" {
				// Saya bayar hutang = pengeluaran
				catName = "Bayar Hutang"
				txType = "expense"
			} else {
				// Piutang diterima = pemasukan
				catName = "Terima Piutang"
				txType = "income"
			}

			var catID int
			db.QueryRow("SELECT id FROM categories WHERE name = ? LIMIT 1", catName).Scan(&catID)
			if catID == 0 {
				res, _ := db.Exec("INSERT INTO categories (name, type, icon) VALUES (?, ?, 'ph-hand-coins')", catName, txType)
				if cid, cerr := res.LastInsertId(); cerr == nil {
					catID = int(cid)
				}
			}

			txNote := debtName + " - " + creditor
			if note != "" {
				txNote = txNote + " - " + note
			}
			db.Exec(`INSERT INTO transactions (user_id, type, amount, category_id, note, date)
				VALUES (?, ?, ?, ?, ?, ?)`,
				userID, txType, amount, catID, txNote, nowWIB().Format("2006-01-02 15:04:05"))
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Pembayaran berhasil dicatat",
			"new_paid":   newPaid,
			"is_settled": isSettled == 1,
			"formatted":  formatRupiah(newPaid),
			"debt_name":  debtName,
			"creditor":   creditor,
			"debt_type":  debtType,
			"amount":     amount,
		})
	})
}
