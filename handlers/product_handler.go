package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"backend-muslimverse/config"

	"github.com/gin-gonic/gin"
)

type ProductRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Price       int      `json:"price" binding:"required,min=1000"`
	Stock       int      `json:"stock" binding:"required,min=0"`
	Category    string   `json:"category" binding:"required"`
	Images      []string `json:"images"`
	IsHalal     bool     `json:"is_halal"`
}

func CreateProduct(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := map[string]interface{}{
		"seller_id":   userID.(string),
		"name":        req.Name,
		"description": req.Description,
		"price":       req.Price,
		"stock":       req.Stock,
		"category":    req.Category,
		"images":      req.Images,
		"is_halal":    req.IsHalal,
		"created_at":  time.Now(),
	}

	productJSON, _ := json.Marshal(product)

	client := &http.Client{}
	reqInsert, _ := http.NewRequest("POST", config.SupabaseURL+"/rest/v1/products", strings.NewReader(string(productJSON)))
	reqInsert.Header.Set("Content-Type", "application/json")
	reqInsert.Header.Set("apikey", config.SupabaseKey)
	reqInsert.Header.Set("Authorization", "Bearer "+config.SupabaseKey)
	reqInsert.Header.Set("Prefer", "return=representation")

	resp, err := client.Do(reqInsert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result) > 0 {
		c.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"message": "Produk berhasil ditambahkan",
			"data":    result[0],
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan produk"})
	}
}

func GetProducts(c *gin.Context) {
	category := c.Query("category")
	sellerID := c.Query("seller_id")
	search := c.Query("search")
	limit := c.DefaultQuery("limit", "20")
	page := c.DefaultQuery("page", "0")

	url := config.SupabaseURL + "/rest/v1/products?select=*&order=created_at.desc&limit=" + limit + "&offset=" + page

	if category != "" {
		url += "&category=eq." + category
	}
	if sellerID != "" {
		url += "&seller_id=eq." + sellerID
	}
	if search != "" {
		url += "&name=ilike.*" + search + "*"
	}

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

	var products []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&products)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   products,
	})
}

func GetProductByID(c *gin.Context) {
	productID := c.Param("id")

	url := config.SupabaseURL + "/rest/v1/products?id=eq." + productID

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

	var products []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&products)

	if len(products) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   products[0],
	})
}

func UpdateProduct(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	productID := c.Param("id")

	// Cek kepemilikan
	if !isProductOwner(userID.(string), productID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak berhak mengubah produk ini"})
		return
	}

	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateData := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"price":       req.Price,
		"stock":       req.Stock,
		"category":    req.Category,
		"images":      req.Images,
		"is_halal":    req.IsHalal,
	}

	updateJSON, _ := json.Marshal(updateData)

	url := config.SupabaseURL + "/rest/v1/products?id=eq." + productID
	client := &http.Client{}
	reqPatch, _ := http.NewRequest("PATCH", url, strings.NewReader(string(updateJSON)))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("apikey", config.SupabaseKey)
	reqPatch.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(reqPatch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Produk berhasil diupdate",
	})
}

func DeleteProduct(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	productID := c.Param("id")

	if !isProductOwner(userID.(string), productID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak berhak menghapus produk ini"})
		return
	}

	url := config.SupabaseURL + "/rest/v1/products?id=eq." + productID
	client := &http.Client{}
	reqDel, _ := http.NewRequest("DELETE", url, nil)
	reqDel.Header.Set("apikey", config.SupabaseKey)
	reqDel.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(reqDel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Produk berhasil dihapus",
	})
}

func isProductOwner(userID, productID string) bool {
	url := config.SupabaseURL + "/rest/v1/products?id=eq." + productID + "&select=seller_id"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var products []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&products)

	if len(products) == 0 {
		return false
	}

	sellerID, ok := products[0]["seller_id"].(string)
	return ok && sellerID == userID
}
