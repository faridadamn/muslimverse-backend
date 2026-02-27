package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"backend-muslimverse/config"
	"backend-muslimverse/models"

	"github.com/gin-gonic/gin"
)

// GetResellerLevel - Ambil level reseller user
func GetResellerLevel(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	url := config.SupabaseURL + "/rest/v1/reseller_levels?user_id=eq." + userID.(string)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var levels []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&levels)

	if len(levels) == 0 {
		// Buat level baru
		newLevel := map[string]interface{}{
			"user_id":         userID.(string),
			"level":           "beginner",
			"total_sales":     0,
			"commission_rate": 5,
		}
		newLevelJSON, _ := json.Marshal(newLevel)

		reqInsert, _ := http.NewRequest("POST", config.SupabaseURL+"/rest/v1/reseller_levels", strings.NewReader(string(newLevelJSON)))
		reqInsert.Header.Set("Content-Type", "application/json")
		reqInsert.Header.Set("apikey", config.SupabaseKey)
		reqInsert.Header.Set("Authorization", "Bearer "+config.SupabaseKey)
		reqInsert.Header.Set("Prefer", "return=representation")

		respInsert, _ := client.Do(reqInsert)
		defer respInsert.Body.Close()

		var newLevels []map[string]interface{}
		json.NewDecoder(respInsert.Body).Decode(&newLevels)

		if len(newLevels) > 0 {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data":   newLevels[0],
			})
			return
		}
	}

	if len(levels) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   levels[0],
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   nil,
		})
	}
}

// GetLevelBenefits - Dapatkan daftar level dan benefitnya
func GetLevelBenefits(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   models.LevelBenefits,
	})
}

// CalculateCommission - Hitung komisi berdasarkan level
func CalculateCommission(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Ambil data level user
	url := config.SupabaseURL + "/rest/v1/reseller_levels?user_id=eq." + userID.(string)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var levels []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&levels)

	if len(levels) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": gin.H{
				"commission_rate": 5,
				"total_sales":     0,
				"level":           "beginner",
			},
		})
		return
	}

	level := levels[0]
	rate, _ := level["commission_rate"].(float64)
	totalSales, _ := level["total_sales"].(float64)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"commission_rate": rate,
			"total_sales":     totalSales,
			"level":           level["level"],
		},
	})
}

// UpdateResellerLevel - Update level berdasarkan total penjualan
func UpdateResellerLevel(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Ambil total penjualan dari orders
	url := config.SupabaseURL + "/rest/v1/orders?seller_id=eq." + userID.(string) + "&status=eq.delivered&select=total_price"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var orders []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orders)

	totalSales := 0
	for _, order := range orders {
		if price, ok := order["total_price"].(float64); ok {
			totalSales += int(price)
		}
	}

	// Tentukan level berdasarkan total penjualan
	var newLevel string
	var commissionRate int

	for _, benefit := range models.LevelBenefits {
		if totalSales >= benefit.MinSales {
			newLevel = benefit.Level
			commissionRate = benefit.CommissionRate
		}
	}

	// Update level
	updateData := map[string]interface{}{
		"level":           newLevel,
		"total_sales":     totalSales,
		"commission_rate": commissionRate,
		"updated_at":      time.Now(),
	}

	updateJSON, _ := json.Marshal(updateData)

	url = config.SupabaseURL + "/rest/v1/reseller_levels?user_id=eq." + userID.(string)
	reqPatch, _ := http.NewRequest("PATCH", url, strings.NewReader(string(updateJSON)))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("apikey", config.SupabaseKey)
	reqPatch.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	respPatch, err := client.Do(reqPatch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer respPatch.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Level berhasil diupdate",
		"data": gin.H{
			"level":           newLevel,
			"total_sales":     totalSales,
			"commission_rate": commissionRate,
		},
	})
}
