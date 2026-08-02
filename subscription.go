package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Plan limits
const (
	FreePlanTxPerMonth   = 50
	FreePlanCategories   = 3
	FreePlanSavings      = 3
	FreePlanHistoryDays  = 30
	PremiumPriceMonthly  = 29000 // Rp 29.000/bulan
)

type Subscription struct {
	ID                    int
	UserID                int
	Plan                  string
	Status                string
	StartedAt             time.Time
	ExpiresAt             *time.Time
	MidtransOrderID       string
	MidtransTransactionID string
}

type PlanInfo struct {
	IsPremium       bool
	Plan            string
	ExpiresAt       *time.Time
	TxUsedThisMonth int
	TxLimit         int
	CatUsed         int
	CatLimit        int
	SavingsUsed     int
	SavingsLimit    int
}

func GetSubscription(userID int) (*Subscription, error) {
	var sub Subscription
	var expiresAt sql.NullTime
	err := db.QueryRow(`
		SELECT id, user_id, plan, status, started_at, expires_at,
		       COALESCE(midtrans_order_id,''), COALESCE(midtrans_transaction_id,'')
		FROM subscriptions WHERE user_id = ?
	`, userID).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status,
		&sub.StartedAt, &expiresAt, &sub.MidtransOrderID, &sub.MidtransTransactionID)

	if err == sql.ErrNoRows {
		// Auto-create free subscription
		db.Exec(`INSERT OR IGNORE INTO subscriptions (user_id, plan, status) VALUES (?, 'free', 'active')`, userID)
		return &Subscription{UserID: userID, Plan: "free", Status: "active"}, nil
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		sub.ExpiresAt = &expiresAt.Time
	}
	return &sub, nil
}

func IsPremium(userID int) bool {
	sub, err := GetSubscription(userID)
	if err != nil {
		return false
	}
	if sub.Plan != "premium" || sub.Status != "active" {
		return false
	}
	if sub.ExpiresAt != nil && sub.ExpiresAt.Before(time.Now()) {
		// Expired — downgrade ke free
		db.Exec(`UPDATE subscriptions SET plan='free', status='active' WHERE user_id=?`, userID)
		return false
	}
	return true
}

func GetPlanInfo(userID int) *PlanInfo {
	isPremium := IsPremium(userID)
	sub, _ := GetSubscription(userID)

	info := &PlanInfo{
		IsPremium: isPremium,
		Plan:      "free",
	}
	if sub != nil {
		info.Plan = sub.Plan
		info.ExpiresAt = sub.ExpiresAt
	}

	if isPremium {
		info.TxLimit = -1
		info.CatLimit = -1
		info.SavingsLimit = -1
		return info
	}

	// Hitung usage free plan
	currentMonth := nowWIB().Format("2006-01")
	db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE user_id=? AND substr(date,1,7)=?`,
		userID, currentMonth).Scan(&info.TxUsedThisMonth)
	db.QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id=?`, userID).Scan(&info.CatUsed)
	db.QueryRow(`SELECT COUNT(*) FROM savings WHERE user_id=? AND is_completed=0`, userID).Scan(&info.SavingsUsed)

	info.TxLimit = FreePlanTxPerMonth
	info.CatLimit = FreePlanCategories
	info.SavingsLimit = FreePlanSavings

	return info
}

func RequirePremium(c *gin.Context) bool {
	userID := GetCurrentUserID(c)
	if !IsPremium(userID) {
		if c.GetHeader("Accept") == "application/json" || c.Request.Header.Get("Content-Type") == "application/json" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":       "Fitur ini hanya tersedia untuk pengguna Premium",
				"upgrade_url": "/upgrade",
			})
		} else {
			c.Redirect(http.StatusFound, "/upgrade")
		}
		c.Abort()
		return false
	}
	return true
}

func CheckTxLimit(c *gin.Context) bool {
	userID := GetCurrentUserID(c)
	if IsPremium(userID) {
		return true
	}
	currentMonth := nowWIB().Format("2006-01")
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE user_id=? AND substr(date,1,7)=?`,
		userID, currentMonth).Scan(&count)
	if count >= FreePlanTxPerMonth {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       fmt.Sprintf("Batas transaksi free plan (%d/bulan) tercapai. Upgrade ke Premium untuk unlimited.", FreePlanTxPerMonth),
			"upgrade_url": "/upgrade",
			"limit_type":  "transaction",
		})
		return false
	}
	return true
}

func CheckCategoryLimit(c *gin.Context) bool {
	userID := GetCurrentUserID(c)
	if IsPremium(userID) {
		return true
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id=?`, userID).Scan(&count)
	if count >= FreePlanCategories {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       fmt.Sprintf("Batas kategori custom free plan (%d kategori) tercapai. Upgrade ke Premium.", FreePlanCategories),
			"upgrade_url": "/upgrade",
			"limit_type":  "category",
		})
		return false
	}
	return true
}

func CheckSavingsLimit(c *gin.Context) bool {
	userID := GetCurrentUserID(c)
	if IsPremium(userID) {
		return true
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM savings WHERE user_id=? AND is_completed=0`, userID).Scan(&count)
	if count >= FreePlanSavings {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       fmt.Sprintf("Batas tabungan free plan (%d goals) tercapai. Upgrade ke Premium.", FreePlanSavings),
			"upgrade_url": "/upgrade",
			"limit_type":  "savings",
		})
		return false
	}
	return true
}

func RegisterSubscriptionRoutes(protected *gin.RouterGroup) {
	midtransServerKey := os.Getenv("MIDTRANS_SERVER_KEY")
	midtransClientKey := os.Getenv("MIDTRANS_CLIENT_KEY")
	midtransEnv := os.Getenv("MIDTRANS_ENV")
	if midtransEnv == "" {
		midtransEnv = "sandbox"
	}

	protected.GET("/upgrade", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		user, _ := GetCurrentUser(c)
		planInfo := GetPlanInfo(userID)
		c.HTML(http.StatusOK, "upgrade.html", gin.H{
			"User":            user,
			"PlanInfo":        planInfo,
			"PriceMonthly":    PremiumPriceMonthly,
			"MidtransClientKey": midtransClientKey,
		})
	})

	protected.POST("/api/subscription/create", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		user, _ := GetCurrentUser(c)

		if IsPremium(userID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kamu sudah Premium"})
			return
		}

		if midtransServerKey == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway belum dikonfigurasi"})
			return
		}

		orderID := fmt.Sprintf("FT-PREMIUM-%d-%d", userID, time.Now().Unix())

		// Simpan payment record
		db.Exec(`INSERT INTO subscription_payments (user_id, order_id, amount, plan, period_months, status)
			VALUES (?, ?, ?, 'premium', 1, 'pending')`, userID, orderID, PremiumPriceMonthly)

		// Request Snap token ke Midtrans
		snapToken, redirectURL, err := createMidtransTransaction(
			midtransServerKey, midtransEnv, orderID,
			PremiumPriceMonthly, user.Name, user.Email,
		)
		if err != nil {
			logError("Midtrans error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"snap_token":   snapToken,
			"redirect_url": redirectURL,
			"order_id":     orderID,
		})
	})

	protected.POST("/api/subscription/verify", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		var req struct {
			OrderID       string `json:"order_id"`
			TransactionID string `json:"transaction_id"`
			Status        string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Verify dengan Midtrans server-side
		status, err := verifyMidtransPayment(midtransServerKey, midtransEnv, req.OrderID)
		if err != nil {
			logError("Midtrans verify error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal verifikasi pembayaran"})
			return
		}

		if status == "settlement" || status == "capture" {
			expiresAt := time.Now().AddDate(0, 1, 0)
			db.Exec(`INSERT INTO subscriptions (user_id, plan, status, expires_at, midtrans_order_id, midtrans_transaction_id)
				VALUES (?, 'premium', 'active', ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					plan='premium', status='active', expires_at=excluded.expires_at,
					midtrans_order_id=excluded.midtrans_order_id,
					midtrans_transaction_id=excluded.midtrans_transaction_id`,
				userID, expiresAt.Format("2006-01-02 15:04:05"), req.OrderID, req.TransactionID)

			db.Exec(`UPDATE subscription_payments SET status='paid', paid_at=?, midtrans_transaction_id=?
				WHERE order_id=?`, time.Now().Format("2006-01-02 15:04:05"), req.TransactionID, req.OrderID)

			logInfo("User %d upgraded to Premium until %s", userID, expiresAt.Format("2006-01-02"))
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Selamat! Kamu sekarang Premium 🎉"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pembayaran belum selesai", "status": status})
		}
	})

	// Midtrans webhook
	protected.POST("/api/subscription/webhook", func(c *gin.Context) {
		var notif map[string]interface{}
		if err := c.ShouldBindJSON(&notif); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid"})
			return
		}

		orderID, _ := notif["order_id"].(string)
		txStatus, _ := notif["transaction_status"].(string)
		txID, _ := notif["transaction_id"].(string)

		if orderID == "" {
			c.JSON(http.StatusOK, gin.H{"status": "ignored"})
			return
		}

		var userID int
		db.QueryRow(`SELECT user_id FROM subscription_payments WHERE order_id=?`, orderID).Scan(&userID)
		if userID == 0 {
			c.JSON(http.StatusOK, gin.H{"status": "not found"})
			return
		}

		if txStatus == "settlement" || txStatus == "capture" {
			expiresAt := time.Now().AddDate(0, 1, 0)
			db.Exec(`INSERT INTO subscriptions (user_id, plan, status, expires_at, midtrans_order_id, midtrans_transaction_id)
				VALUES (?, 'premium', 'active', ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					plan='premium', status='active', expires_at=excluded.expires_at,
					midtrans_order_id=excluded.midtrans_order_id,
					midtrans_transaction_id=excluded.midtrans_transaction_id`,
				userID, expiresAt.Format("2006-01-02 15:04:05"), orderID, txID)
			db.Exec(`UPDATE subscription_payments SET status='paid', paid_at=?, midtrans_transaction_id=? WHERE order_id=?`,
				time.Now().Format("2006-01-02 15:04:05"), txID, orderID)
			logInfo("Webhook: user %d upgraded to Premium via %s", userID, orderID)
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
