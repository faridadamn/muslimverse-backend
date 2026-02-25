package config

import (
	"log"
	"os"
)

var (
	SupabaseURL string
	SupabaseKey string
)

func InitDB() {
	SupabaseURL = os.Getenv("SUPABASE_URL")
	SupabaseKey = os.Getenv("SUPABASE_ANON_KEY")

	if SupabaseURL == "" || SupabaseKey == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_ANON_KEY must be set")
	}

	log.Println("✅ Supabase config loaded")
}
