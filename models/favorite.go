package models

import (
	"time"
)

type Favorite struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	KotaID    string    `json:"kota_id"`
	KotaNama  string    `json:"kota_nama"`
	CreatedAt time.Time `json:"created_at"`
}

type FavoriteRequest struct {
	KotaID   string `json:"kota_id" binding:"required"`
	KotaNama string `json:"kota_nama" binding:"required"`
}
