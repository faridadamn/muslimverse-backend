package models

import (
	"time"
)

type Order struct {
	ID            string    `json:"id"`
	BuyerID       string    `json:"buyer_id"`
	SellerID      string    `json:"seller_id"`
	ProductID     string    `json:"product_id"`
	ProductName   string    `json:"product_name"`
	Quantity      int       `json:"quantity"`
	TotalPrice    int       `json:"total_price"`
	Status        string    `json:"status"` // pending, paid, processed, shipped, delivered, cancelled
	PaymentMethod string    `json:"payment_method"`
	PaymentProof  string    `json:"payment_proof"`
	ShippingAddr  string    `json:"shipping_addr"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type OrderRequest struct {
	ProductID    string `json:"product_id" binding:"required"`
	Quantity     int    `json:"quantity" binding:"required,min=1"`
	ShippingAddr string `json:"shipping_addr" binding:"required"`
	Notes        string `json:"notes"`
}

type PaymentProofRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required"`
	PaymentProof  string `json:"payment_proof" binding:"required"` // URL bukti transfer
}
