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

// CreateOrder - Buat order baru
func CreateOrder(c *gin.Context) {
	buyerID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ambil data produk
	product, err := getProductByID(req.ProductID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	// Cek stok
	stock, ok := product["stock"].(float64)
	if !ok || int(stock) < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok tidak cukup"})
		return
	}

	sellerID, _ := product["seller_id"].(string)
	productName, _ := product["name"].(string)
	price, _ := product["price"].(float64)

	// Hitung total
	totalPrice := int(price) * req.Quantity

	// Buat order
	order := map[string]interface{}{
		"buyer_id":      buyerID.(string),
		"seller_id":     sellerID,
		"product_id":    req.ProductID,
		"product_name":  productName,
		"quantity":      req.Quantity,
		"total_price":   totalPrice,
		"status":        "pending",
		"shipping_addr": req.ShippingAddr,
		"notes":         req.Notes,
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	}

	orderJSON, _ := json.Marshal(order)

	// Insert order
	client := &http.Client{}
	reqInsert, _ := http.NewRequest("POST", config.SupabaseURL+"/rest/v1/orders", strings.NewReader(string(orderJSON)))
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

	// Kurangi stok produk
	updateStock(product, req.ProductID, int(stock)-req.Quantity)

	if len(result) > 0 {
		c.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"message": "Order berhasil dibuat",
			"data":    result[0],
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat order"})
	}
}

// GetMyOrders - Ambil order user (sebagai pembeli)
func GetMyOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	url := config.SupabaseURL + "/rest/v1/orders?buyer_id=eq." + userID.(string) + "&order=created_at.desc"

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

	var orders []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orders)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   orders,
	})
}

// GetSellerOrders - Ambil order yang diterima penjual
func GetSellerOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	url := config.SupabaseURL + "/rest/v1/orders?seller_id=eq." + userID.(string) + "&order=created_at.desc"

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

	var orders []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orders)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   orders,
	})
}

// UploadPaymentProof - Upload bukti pembayaran
func UploadPaymentProof(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID := c.Param("id")
	var req models.PaymentProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek kepemilikan order
	if !isOrderBuyer(userID.(string), orderID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak berhak mengupdate order ini"})
		return
	}

	updateData := map[string]interface{}{
		"payment_method": req.PaymentMethod,
		"payment_proof":  req.PaymentProof,
		"status":         "paid",
		"updated_at":     time.Now(),
	}

	updateJSON, _ := json.Marshal(updateData)

	url := config.SupabaseURL + "/rest/v1/orders?id=eq." + orderID
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
		"message": "Bukti pembayaran berhasil diupload",
	})
}

// UpdateOrderStatus - Update status order (untuk penjual)
func UpdateOrderStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi status
	validStatus := map[string]bool{
		"processed": true, "shipped": true, "delivered": true, "cancelled": true,
	}
	if !validStatus[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status tidak valid"})
		return
	}

	// Cek kepemilikan order (sebagai penjual)
	if !isOrderSeller(userID.(string), orderID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak berhak mengupdate order ini"})
		return
	}

	updateData := map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now(),
	}

	updateJSON, _ := json.Marshal(updateData)

	url := config.SupabaseURL + "/rest/v1/orders?id=eq." + orderID
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
		"message": "Status order berhasil diupdate",
	})
}

// Helper functions
func getProductByID(productID string) (map[string]interface{}, error) {
	url := config.SupabaseURL + "/rest/v1/products?id=eq." + productID
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&products)

	if len(products) == 0 {
		return nil, nil
	}
	return products[0], nil
}

func updateStock(product map[string]interface{}, productID string, newStock int) {
	updateData := map[string]interface{}{
		"stock": newStock,
	}
	updateJSON, _ := json.Marshal(updateData)

	client := &http.Client{}
	url := config.SupabaseURL + "/rest/v1/products?id=eq." + productID
	reqPatch, _ := http.NewRequest("PATCH", url, strings.NewReader(string(updateJSON)))
	reqPatch.Header.Set("Content-Type", "application/json")
	reqPatch.Header.Set("apikey", config.SupabaseKey)
	reqPatch.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	client.Do(reqPatch)
}

func isOrderBuyer(userID, orderID string) bool {
	url := config.SupabaseURL + "/rest/v1/orders?id=eq." + orderID + "&select=buyer_id"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var orders []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orders)

	if len(orders) == 0 {
		return false
	}

	buyerID, ok := orders[0]["buyer_id"].(string)
	return ok && buyerID == userID
}

func isOrderSeller(userID, orderID string) bool {
	url := config.SupabaseURL + "/rest/v1/orders?id=eq." + orderID + "&select=seller_id"
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var orders []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orders)

	if len(orders) == 0 {
		return false
	}

	sellerID, ok := orders[0]["seller_id"].(string)
	return ok && sellerID == userID
}
