package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminUserRow struct {
	ID            int
	Name          string
	Email         string
	Plan          string
	CreatedAt     time.Time
	ExpiresAt     string
	TotalTx       int
	IsAdmin       bool
	FormattedDate string
}

type AdminStats struct {
	TotalUsers   int
	PremiumUsers int
	MonthlyTx    int
	MonthlyRev   int
	FormattedRev string
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		if userID == 0 {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		var isAdmin int
		err := db.QueryRow("SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?", userID).Scan(&isAdmin)
		if err != nil || isAdmin != 1 {
			c.HTML(http.StatusForbidden, "404.html", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func getAdminStats() AdminStats {
	var stats AdminStats
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	db.QueryRow(`
		SELECT COUNT(*) FROM subscriptions
		WHERE plan='premium' AND status='active'
		AND (expires_at IS NULL OR expires_at > datetime('now'))
	`).Scan(&stats.PremiumUsers)
	currentMonth := nowWIB().Format("2006-01")
	db.QueryRow("SELECT COUNT(*) FROM transactions WHERE substr(date,1,7)=?", currentMonth).Scan(&stats.MonthlyTx)
	db.QueryRow(`
		SELECT COALESCE(SUM(amount),0) FROM subscription_payments
		WHERE status='paid' AND substr(paid_at,1,7)=?
	`, currentMonth).Scan(&stats.MonthlyRev)
	stats.FormattedRev = formatRupiah(stats.MonthlyRev)
	return stats
}

func getAdminUsers() []AdminUserRow {
	rows, err := db.Query(`
		SELECT u.id, u.name, u.email, u.created_at, COALESCE(u.is_admin,0),
		       COALESCE(s.plan,'free'),
		       COALESCE(s.expires_at,''),
		       (SELECT COUNT(*) FROM transactions t WHERE t.user_id=u.id) as total_tx
		FROM users u
		LEFT JOIN subscriptions s ON s.user_id=u.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var users []AdminUserRow
	for rows.Next() {
		var u AdminUserRow
		var isAdmin int
		rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt, &isAdmin, &u.Plan, &u.ExpiresAt, &u.TotalTx)
		u.IsAdmin = isAdmin == 1
		u.FormattedDate = u.CreatedAt.Format("02 Jan 2006")
		users = append(users, u)
	}
	return users
}

func RegisterAdminRoutes(router *gin.Engine, protected *gin.RouterGroup) {
	admin := protected.Group("/backoffice")
	admin.Use(AdminMiddleware())

	admin.GET("/", func(c *gin.Context) {
		user, _ := GetCurrentUser(c)
		stats := getAdminStats()
		users := getAdminUsers()
		c.HTML(http.StatusOK, "admin.html", gin.H{
			"User":  user,
			"Stats": stats,
			"Users": users,
		})
	})

	admin.GET("/api/stats", func(c *gin.Context) {
		stats := getAdminStats()
		c.JSON(http.StatusOK, stats)
	})

	admin.GET("/api/users", func(c *gin.Context) {
		users := getAdminUsers()
		c.JSON(http.StatusOK, users)
	})

	admin.POST("/api/users/:id/toggle-premium", func(c *gin.Context) {
		actorID := GetCurrentUserID(c)
		targetID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
			return
		}

		var plan string
		var expiresAt *time.Time
		row := db.QueryRow(`SELECT COALESCE(plan,'free'), expires_at FROM subscriptions WHERE user_id=?`, targetID)
		var rawExpires *time.Time
		row.Scan(&plan, &rawExpires)
		expiresAt = rawExpires

		isPremiumNow := plan == "premium" && (expiresAt == nil || expiresAt.After(time.Now()))

		if isPremiumNow {
			db.Exec(`UPDATE subscriptions SET plan='free', status='active', expires_at=NULL WHERE user_id=?`, targetID)
			logInfo("Admin %d downgraded user %d to free", actorID, targetID)
			c.JSON(http.StatusOK, gin.H{"message": "User di-downgrade ke Free", "plan": "free"})
		} else {
			newExpiry := time.Now().AddDate(1, 0, 0)
			db.Exec(`INSERT INTO subscriptions (user_id, plan, status, expires_at)
				VALUES (?, 'premium', 'active', ?)
				ON CONFLICT(user_id) DO UPDATE SET plan='premium', status='active', expires_at=excluded.expires_at`,
				targetID, newExpiry.Format("2006-01-02 15:04:05"))
			logInfo("Admin %d manually granted Premium to user %d until %s", actorID, targetID, newExpiry.Format("2006-01-02"))
			c.JSON(http.StatusOK, gin.H{"message": "User di-upgrade ke Premium (1 tahun)", "plan": "premium"})
		}
	})

	admin.POST("/api/users/:id/toggle-admin", func(c *gin.Context) {
		actorID := GetCurrentUserID(c)
		targetID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
			return
		}
		if targetID == actorID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak bisa mengubah status admin diri sendiri"})
			return
		}

		var isAdmin int
		db.QueryRow("SELECT COALESCE(is_admin,0) FROM users WHERE id=?", targetID).Scan(&isAdmin)

		newVal := 1
		if isAdmin == 1 {
			newVal = 0
		}
		db.Exec("UPDATE users SET is_admin=? WHERE id=?", newVal, targetID)
		logInfo("Admin %d set user %d is_admin=%d", actorID, targetID, newVal)

		action := "dipromosikan jadi admin"
		if newVal == 0 {
			action = "dicopot dari admin"
		}
		c.JSON(http.StatusOK, gin.H{"message": "User " + action, "is_admin": newVal == 1})
	})
}
