package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var phosphorIcons = []string{
	"ph-hamburger", "ph-coffee", "ph-shopping-cart", "ph-car", "ph-bus",
	"ph-train", "ph-airplane", "ph-house", "ph-buildings", "ph-hospital",
	"ph-pill", "ph-graduation-cap", "ph-book", "ph-game-controller",
	"ph-music-note", "ph-film-strip", "ph-dumbbell", "ph-soccer-ball",
	"ph-shirt", "ph-bag", "ph-scissors", "ph-wrench", "ph-lightning",
	"ph-drop", "ph-flame", "ph-wifi", "ph-phone", "ph-device-mobile",
	"ph-monitor", "ph-printer", "ph-credit-card", "ph-bank", "ph-money",
	"ph-piggy-bank", "ph-chart-line-up", "ph-briefcase", "ph-handshake",
	"ph-gift", "ph-heart", "ph-star", "ph-shield", "ph-tree",
	"ph-dog", "ph-cat", "ph-baby", "ph-person", "ph-users",
	"ph-first-aid-kit", "ph-bicycle", "ph-motorcycle",
}

func RegisterCategoryRoutes(protected *gin.RouterGroup) {

	protected.GET("/categories", func(c *gin.Context) {
		userID := GetCurrentUserID(c)

		rows, err := db.Query(`
			SELECT id, name, type, icon, COALESCE(user_id, 0), COALESCE(is_default, 1)
			FROM categories
			WHERE user_id IS NULL OR user_id = ?
			ORDER BY is_default DESC, type, name
		`, userID)

		type CategoryRow struct {
			Category
			IsDefault bool
			IsCustom  bool
		}

		var cats []CategoryRow
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cat CategoryRow
				var ownerID, isDefault int
				rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Icon, &ownerID, &isDefault)
				cat.IsDefault = isDefault == 1
				cat.IsCustom = ownerID != 0
				cats = append(cats, cat)
			}
		}

		c.HTML(http.StatusOK, "categories.html", gin.H{
			"Categories": cats,
			"Icons":      phosphorIcons,
		})
	})

	protected.POST("/api/categories", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		name := c.PostForm("name")
		catType := c.PostForm("type")
		icon := c.DefaultPostForm("icon", "ph-tag")

		if name == "" || (catType != "income" && catType != "expense") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan tipe wajib diisi"})
			return
		}

		result, err := db.Exec(`
			INSERT INTO categories (name, type, icon, user_id, is_default)
			VALUES (?, ?, ?, ?, 0)
		`, name, catType, icon, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan kategori"})
			return
		}
		id, _ := result.LastInsertId()
		c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil dibuat", "id": id})
	})

	protected.PUT("/api/categories/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")
		name := c.PostForm("name")
		icon := c.PostForm("icon")

		var ownerID int
		err := db.QueryRow("SELECT COALESCE(user_id,0) FROM categories WHERE id=?", id).Scan(&ownerID)
		if err != nil || (ownerID != 0 && ownerID != userID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak bisa mengedit kategori ini"})
			return
		}

		db.Exec("UPDATE categories SET name=?, icon=? WHERE id=?", name, icon, id)
		c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil diperbarui"})
	})

	protected.DELETE("/api/categories/:id", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		id := c.Param("id")

		var ownerID int
		err := db.QueryRow("SELECT COALESCE(user_id,0) FROM categories WHERE id=?", id).Scan(&ownerID)
		if err != nil || ownerID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya kategori custom yang bisa dihapus"})
			return
		}
		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bukan kategori milikmu"})
			return
		}

		db.Exec("DELETE FROM categories WHERE id=? AND user_id=?", id, userID)
		c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil dihapus"})
	})
}
