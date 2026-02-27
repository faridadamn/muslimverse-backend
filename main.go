package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"backend-muslimverse/config"
	"backend-muslimverse/handlers"
	"backend-muslimverse/middleware"
)

func main() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Init config
	config.InitDB()

	// Init router
	r := gin.Default()

	// CORS MIDDLEWARE
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://localhost:3000", "http://localhost:5000", "http://127.0.0.1:8080", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public routes
	api := r.Group("/api")
	{
		// Auth
		api.POST("/register", handlers.Register)
		api.POST("/login", handlers.Login)

		// Jadwal (public)
		api.GET("/jadwal/:kota", handlers.GetJadwalShalat)
		api.GET("/daftar-kota", handlers.GetDaftarKota)

		// Produk - beberapa endpoint public (lihat produk, detail produk)
		api.GET("/products", handlers.GetProducts)
		api.GET("/products/:id", handlers.GetProductByID)
		api.GET("/reseller/benefits", handlers.GetLevelBenefits) // Benefit level public
	}

	// Protected routes (perlu JWT)
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Favorites
		protected.POST("/favorites", handlers.AddFavorite)
		protected.GET("/favorites", handlers.GetFavorites)
		protected.DELETE("/favorites/:id", handlers.DeleteFavorite)

		// Shalat Tracker
		protected.GET("/shalat/tracker/today", handlers.GetTodayTracker)
		protected.POST("/shalat/tracker/update", handlers.UpdateShalat)
		protected.GET("/shalat/history", handlers.GetShalatHistory)
		protected.GET("/shalat/stats", handlers.GetShalatStats)

		// ========== MARKETPLACE ROUTES ==========

		// PRODUCT ROUTES
		protected.POST("/products", handlers.CreateProduct)       // Tambah produk
		protected.PUT("/products/:id", handlers.UpdateProduct)    // Update produk
		protected.DELETE("/products/:id", handlers.DeleteProduct) // Hapus produk

		// ORDER ROUTES
		protected.POST("/orders", handlers.CreateOrder)                    // Buat order
		protected.GET("/orders/my", handlers.GetMyOrders)                  // Order sebagai pembeli
		protected.GET("/orders/seller", handlers.GetSellerOrders)          // Order sebagai penjual
		protected.POST("/orders/:id/payment", handlers.UploadPaymentProof) // Upload bukti bayar
		protected.PUT("/orders/:id/status", handlers.UpdateOrderStatus)    // Update status order

		// RESELLER ROUTES
		protected.GET("/reseller/level", handlers.GetResellerLevel)            // Level reseller user
		protected.GET("/reseller/commission", handlers.CalculateCommission)    // Hitung komisi
		protected.POST("/reseller/update-level", handlers.UpdateResellerLevel) // Update level manual
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port:", port)
	r.Run(":" + port)
}
