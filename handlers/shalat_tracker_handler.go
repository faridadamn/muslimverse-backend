package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"backend-muslimverse/config"

	"github.com/gin-gonic/gin"
)

type UpdateShalatRequest struct {
	Shalat string `json:"shalat" binding:"required"`
	Status bool   `json:"status" binding:"required"`
}

func GetTodayTracker(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	today := time.Now().Format("2006-01-02")

	// Cek apakah sudah ada data hari ini
	url := config.SupabaseURL + "/rest/v1/shalat_tracker?user_id=eq." + userID.(string) + "&tanggal=eq." + today

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var trackers []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&trackers)

	// Jika belum ada, buat baru
	if len(trackers) == 0 {
		newTracker := map[string]interface{}{
			"user_id": userID.(string),
			"tanggal": today,
			"subuh":   false,
			"dzuhur":  false,
			"ashar":   false,
			"maghrib": false,
			"isya":    false,
		}

		insertJSON, _ := json.Marshal(newTracker)
		insertReq, _ := http.NewRequest("POST", config.SupabaseURL+"/rest/v1/shalat_tracker", strings.NewReader(string(insertJSON)))
		insertReq.Header.Set("Content-Type", "application/json")
		insertReq.Header.Set("apikey", config.SupabaseKey)
		insertReq.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

		insertResp, _ := client.Do(insertReq)
		defer insertResp.Body.Close()

		var newTrackers []map[string]interface{}
		json.NewDecoder(insertResp.Body).Decode(&newTrackers)

		if len(newTrackers) > 0 {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data":   newTrackers[0],
			})
			return
		}
	}

	if len(trackers) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   trackers[0],
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   nil,
		})
	}
}

func UpdateShalat(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateShalatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	today := time.Now().Format("2006-01-02")

	// Validasi nama shalat
	validShalat := map[string]bool{
		"subuh": true, "dzuhur": true, "ashar": true, "maghrib": true, "isya": true,
	}
	if !validShalat[req.Shalat] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama shalat tidak valid"})
		return
	}

	// Update data
	url := config.SupabaseURL + "/rest/v1/shalat_tracker?user_id=eq." + userID.(string) + "&tanggal=eq." + today

	updateData := map[string]interface{}{
		req.Shalat:   req.Status,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	updateJSON, _ := json.Marshal(updateData)

	updateReq, _ := http.NewRequest("PATCH", url, strings.NewReader(string(updateJSON)))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("apikey", config.SupabaseKey)
	updateReq.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Catat ke history
	go recordHistory(userID.(string), req.Shalat, req.Status)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Shalat updated",
	})
}

func GetShalatHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	days := c.DefaultQuery("days", "30")

	url := config.SupabaseURL + "/rest/v1/shalat_history?user_id=eq." + userID.(string) + "&order=tanggal.desc&limit=" + days

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var history []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&history)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   history,
	})
}

func GetShalatStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Hitung streak dan statistik lainnya
	// Ini implementasi sederhana, bisa dikembangin

	url := config.SupabaseURL + "/rest/v1/shalat_history?user_id=eq." + userID.(string) + "&order=tanggal.desc"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var history []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&history)

	// Hitung total completed
	totalCompleted := 0
	for _, item := range history {
		if item["status"] == true {
			totalCompleted++
		}
	}

	// Streak sederhana (bisa dikembangin)
	currentStreak := 0
	completionRate := 0.0
	if len(history) > 0 {
		completionRate = float64(totalCompleted) / float64(len(history)) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"current_streak":  currentStreak,
			"total_completed": totalCompleted,
			"completion_rate": int(completionRate),
			"total_days":      len(history),
		},
	})
}

// Fungsi helper untuk mencatat history
func recordHistory(userID string, shalat string, status bool) {
	history := map[string]interface{}{
		"user_id":      userID,
		"tanggal":      time.Now().Format("2006-01-02"),
		"shalat":       shalat,
		"status":       status,
		"completed_at": time.Now().Format(time.RFC3339),
	}

	historyJSON, _ := json.Marshal(history)

	req, _ := http.NewRequest("POST", config.SupabaseURL+"/rest/v1/shalat_history", strings.NewReader(string(historyJSON)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client := &http.Client{}
	client.Do(req)
}
