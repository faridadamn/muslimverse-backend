package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"backend-muslimverse/config"
	"backend-muslimverse/models"
)

func AddFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.FavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert ke Supabase
	url := config.SupabaseURL + "/rest/v1/favorites"

	favorite := map[string]interface{}{
		"user_id":   userID,
		"kota_id":   req.KotaID,
		"kota_nama": req.KotaNama,
	}

	jsonBody, _ := json.Marshal(favorite)

	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", config.SupabaseKey)
	httpReq.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal simpan favorite: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		c.JSON(resp.StatusCode, gin.H{"error": "Gagal simpan favorite"})
		return
	}

	var result []models.Favorite
	json.NewDecoder(resp.Body).Decode(&result)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Berhasil ditambahkan ke favorit",
		"favorite": result[0],
	})
}

func GetFavorites(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Query dari Supabase
	url := config.SupabaseURL + "/rest/v1/favorites?user_id=eq." + userID.(string) + "&order=created_at.desc"

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("apikey", config.SupabaseKey)
	httpReq.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal ambil favorites: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var favorites []models.Favorite
	json.NewDecoder(resp.Body).Decode(&favorites)

	c.JSON(http.StatusOK, gin.H{
		"favorites": favorites,
	})
}

func DeleteFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	favoriteID := c.Param("id")

	// Delete dari Supabase
	url := config.SupabaseURL + "/rest/v1/favorites?id=eq." + favoriteID + "&user_id=eq." + userID.(string)

	httpReq, _ := http.NewRequest("DELETE", url, nil)
	httpReq.Header.Set("apikey", config.SupabaseKey)
	httpReq.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus favorite: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		c.JSON(resp.StatusCode, gin.H{"error": "Gagal hapus favorite"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil dihapus dari favorit",
	})
}
