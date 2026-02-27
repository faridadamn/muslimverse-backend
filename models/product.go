package models

import (
	"time"
)

type Product struct {
	ID          string    `json:"id"`
	SellerID    string    `json:"seller_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"`
	Stock       int       `json:"stock"`
	Category    string    `json:"category"`
	Images      []string  `json:"images"`
	IsHalal     bool      `json:"is_halal"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProductRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Price       int      `json:"price" binding:"required,min=1000"`
	Stock       int      `json:"stock" binding:"required,min=0"`
	Category    string   `json:"category" binding:"required"`
	Images      []string `json:"images"`
	IsHalal     bool     `json:"is_halal"`
}
