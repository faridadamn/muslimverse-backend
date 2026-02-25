package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors" // <--- TAMBAH INI
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

	// TAMBAHKAN CORS MIDDLEWARE DI SINI!
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
	}

	// Protected routes (perlu JWT)
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/favorites", handlers.AddFavorite)
		protected.GET("/favorites", handlers.GetFavorites)
		protected.DELETE("/favorites/:id", handlers.DeleteFavorite)
		protected.GET("/shalat/tracker/today", handlers.GetTodayTracker)
		protected.POST("/shalat/tracker/update", handlers.UpdateShalat)
		protected.GET("/shalat/history", handlers.GetShalatHistory)
		protected.GET("/shalat/stats", handlers.GetShalatStats)

	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port:", port)
	r.Run("0.0.0.0:" + port) // Biar bisa diakses dari mana aja
}
