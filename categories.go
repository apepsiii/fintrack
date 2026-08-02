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

var defaultCategories = []Category{
	{Name: "Makanan & Minuman", Type: "expense", Icon: "ph-hamburger"},
	{Name: "Transport", Type: "expense", Icon: "ph-car"},
	{Name: "Belanja", Type: "expense", Icon: "ph-shopping-cart"},
	{Name: "Tagihan & Utilitas", Type: "expense", Icon: "ph-lightning"},
	{Name: "Kesehatan", Type: "expense", Icon: "ph-heart"},
	{Name: "Pendidikan", Type: "expense", Icon: "ph-graduation-cap"},
	{Name: "Hiburan", Type: "expense", Icon: "ph-game-controller"},
	{Name: "Pakaian", Type: "expense", Icon: "ph-shirt"},
	{Name: "Perawatan Diri", Type: "expense", Icon: "ph-sparkle"},
	{Name: "Rumah & Perabot", Type: "expense", Icon: "ph-house"},
	{Name: "Komunikasi", Type: "expense", Icon: "ph-device-mobile"},
	{Name: "Olahraga", Type: "expense", Icon: "ph-barbell"},
	{Name: "Sosial & Hadiah", Type: "expense", Icon: "ph-gift"},
	{Name: "Investasi", Type: "expense", Icon: "ph-trend-up"},
	{Name: "Lainnya", Type: "expense", Icon: "ph-dots-three-outline"},
	{Name: "Gaji", Type: "income", Icon: "ph-money"},
	{Name: "Freelance", Type: "income", Icon: "ph-laptop"},
	{Name: "Bisnis", Type: "income", Icon: "ph-briefcase"},
	{Name: "Investasi", Type: "income", Icon: "ph-chart-line-up"},
	{Name: "Hadiah", Type: "income", Icon: "ph-gift"},
	{Name: "Bonus", Type: "income", Icon: "ph-star"},
	{Name: "Lainnya", Type: "income", Icon: "ph-dots-three-outline"},
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
		if !CheckCategoryLimit(c) {
			return
		}
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

	protected.POST("/api/categories/generate-default", func(c *gin.Context) {
		userID := GetCurrentUserID(c)
		added := 0

		for _, cat := range defaultCategories {
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM categories WHERE name=? AND type=? AND (user_id IS NULL OR user_id=?)`, cat.Name, cat.Type, userID).Scan(&count)
			if count == 0 {
				db.Exec(`INSERT INTO categories (name, type, icon, user_id, is_default) VALUES (?, ?, ?, ?, 0)`, cat.Name, cat.Type, cat.Icon, userID)
				added++
			}
		}

		c.JSON(http.StatusOK, gin.H{"added": added, "message": "Kategori default berhasil digenerate"})
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
