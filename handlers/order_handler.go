package handlers

import (
	"encoding/json"
	"io"
	"log"
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
		log.Println("❌ Unauthorized: user_id not found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("📥 Order request: product_id=%s, quantity=%d, address=%s",
		req.ProductID, req.Quantity, req.ShippingAddr)

	// Ambil data produk
	product, err := getProductByID(req.ProductID)
	if err != nil {
		log.Printf("❌ Error getting product %s: %v", req.ProductID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk"})
		return
	}

	if product == nil {
		log.Printf("❌ Product not found: %s", req.ProductID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}

	// Cek stok
	stock, ok := product["stock"].(float64)
	if !ok {
		log.Printf("❌ Invalid stock type: %T", product["stock"])
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data stok tidak valid"})
		return
	}

	if int(stock) < req.Quantity {
		log.Printf("❌ Insufficient stock: requested %d, available %d", req.Quantity, int(stock))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stok tidak cukup"})
		return
	}

	sellerID, ok := product["seller_id"].(string)
	if !ok {
		log.Printf("❌ Invalid seller_id type: %T", product["seller_id"])
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data penjual tidak valid"})
		return
	}

	productName, _ := product["name"].(string)
	price, ok := product["price"].(float64)
	if !ok {
		log.Printf("❌ Invalid price type: %T", product["price"])
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data harga tidak valid"})
		return
	}

	// Hitung total
	totalPrice := int(price) * req.Quantity
	log.Printf("✅ Product valid: %s, price: %f, total: %d", productName, price, totalPrice)

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
	log.Printf("📤 Inserting order: %s", string(orderJSON))

	// Insert order ke Supabase
	client := &http.Client{}
	reqInsert, _ := http.NewRequest("POST", config.SupabaseURL+"/rest/v1/orders", strings.NewReader(string(orderJSON)))
	reqInsert.Header.Set("Content-Type", "application/json")
	reqInsert.Header.Set("apikey", config.SupabaseKey)
	reqInsert.Header.Set("Authorization", "Bearer "+config.SupabaseKey)
	reqInsert.Header.Set("Prefer", "return=representation")

	resp, err := client.Do(reqInsert)
	if err != nil {
		log.Printf("❌ Error inserting order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan order"})
		return
	}
	defer resp.Body.Close()

	// Baca response
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("📥 Supabase insert response code: %d", resp.StatusCode)
	log.Printf("📥 Supabase insert response body: %s", string(respBody))

	if resp.StatusCode >= 400 {
		log.Printf("❌ Supabase error: %s", string(respBody))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Gagal membuat order",
			"details": string(respBody),
		})
		return
	}

	// Parse hasil insert
	var result []map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		log.Printf("❌ Error parsing insert response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal parse response"})
		return
	}

	if len(result) == 0 {
		log.Printf("❌ No data returned from insert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat order"})
		return
	}

	// Kurangi stok produk
	newStock := int(stock) - req.Quantity
	log.Printf("📦 Updating stock for product %s from %d to %d", req.ProductID, int(stock), newStock)
	go updateStock(product, req.ProductID, newStock)

	// Kirim notifikasi ke penjual (dipisah dari response)
	orderID := result[0]["id"].(string)
	go sendNewOrderNotification(sellerID, orderID, productName, req.Quantity, totalPrice)

	log.Printf("✅ Order created successfully: %s", orderID)
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Order berhasil dibuat",
		"data":    result[0],
	})
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

// ==================== HELPER FUNCTIONS ====================

// Fungsi helper untuk notifikasi (DILUAR CreateOrder)
func sendNewOrderNotification(sellerID, orderId, productName string, quantity, totalPrice int) {
	// TODO: Kirim notifikasi via FCM atau OneSignal
	log.Printf("📱 Notifikasi: Pesanan baru untuk penjual %s - %s x%d (Total: %d)",
		sellerID, productName, quantity, totalPrice)
}

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
