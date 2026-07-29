package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func RegisterForgotPasswordRoutes(router *gin.Engine) {
	router.GET("/forgot-password", func(c *gin.Context) {
		c.HTML(http.StatusOK, "forgot_password.html", gin.H{
			"message": c.Query("msg"),
			"error":   c.Query("err"),
		})
	})

	router.POST("/forgot-password", func(c *gin.Context) {
		email := c.PostForm("email")
		if email == "" {
			c.Redirect(http.StatusFound, "/forgot-password?err=Email+wajib+diisi")
			return
		}

		var userID int
		var userName string
		err := db.QueryRow("SELECT id, name FROM users WHERE email=?", email).Scan(&userID, &userName)
		if err != nil {
			c.Redirect(http.StatusFound, "/forgot-password?msg=Jika+email+terdaftar,+kode+reset+sudah+dikirim")
			return
		}

		token := generateToken()
		expires := nowWIB().Add(1 * time.Hour)
		db.Exec("DELETE FROM password_resets WHERE user_id=?", userID)
		db.Exec(`INSERT INTO password_resets (user_id, token, expires_at) VALUES (?, ?, ?)`,
			userID, token, expires.Format("2006-01-02 15:04:05"))

		c.Redirect(http.StatusFound, "/reset-password?token="+token+"&msg=Masukkan+token+untuk+reset+password")
	})

	router.GET("/reset-password", func(c *gin.Context) {
		c.HTML(http.StatusOK, "reset_password.html", gin.H{
			"token":   c.Query("token"),
			"message": c.Query("msg"),
			"error":   c.Query("err"),
		})
	})

	router.POST("/reset-password", func(c *gin.Context) {
		token := c.PostForm("token")
		newPassword := c.PostForm("new_password")
		confirmPassword := c.PostForm("confirm_password")

		if token == "" || newPassword == "" {
			c.Redirect(http.StatusFound, "/reset-password?token="+token+"&err=Semua+field+wajib+diisi")
			return
		}
		if newPassword != confirmPassword {
			c.Redirect(http.StatusFound, "/reset-password?token="+token+"&err=Password+tidak+cocok")
			return
		}
		if len(newPassword) < 6 {
			c.Redirect(http.StatusFound, "/reset-password?token="+token+"&err=Password+minimal+6+karakter")
			return
		}

		var userID int
		var expiresAt string
		var used int
		err := db.QueryRow(`SELECT user_id, expires_at, used FROM password_resets WHERE token=?`, token).
			Scan(&userID, &expiresAt, &used)
		if err != nil || used == 1 {
			c.Redirect(http.StatusFound, "/forgot-password?err=Token+tidak+valid+atau+sudah+digunakan")
			return
		}

		expTime, _ := time.ParseInLocation("2006-01-02 15:04:05", expiresAt, wib)
		if nowWIB().After(expTime) {
			c.Redirect(http.StatusFound, "/forgot-password?err=Token+sudah+kadaluarsa")
			return
		}

		authSvc := NewAuthService()
		hash, err := authSvc.HashPassword(newPassword)
		if err != nil {
			c.Redirect(http.StatusFound, "/reset-password?token="+token+"&err=Terjadi+kesalahan")
			return
		}

		db.Exec("UPDATE users SET password_hash=? WHERE id=?", hash, userID)
		db.Exec("UPDATE password_resets SET used=1 WHERE token=?", token)

		c.Redirect(http.StatusFound, "/login?msg=Password+berhasil+direset,+silakan+login")
	})
}
