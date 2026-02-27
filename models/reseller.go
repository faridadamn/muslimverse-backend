package models

import (
	"time"
)

type ResellerLevel struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Level          string    `json:"level"` // beginner, salesperson, captain, merchant, sultan
	TotalSales     int       `json:"total_sales"`
	CommissionRate int       `json:"commission_rate"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type LevelBenefit struct {
	Level          string `json:"level"`
	MinSales       int    `json:"min_sales"`
	CommissionRate int    `json:"commission_rate"`
	Benefits       string `json:"benefits"`
}

var LevelBenefits = []LevelBenefit{
	{Level: "beginner", MinSales: 0, CommissionRate: 5, Benefits: "Dapat link afiliasi, komisi 5%"},
	{Level: "salesperson", MinSales: 1000000, CommissionRate: 7, Benefits: "Komisi 7%, akses kelas bisnis"},
	{Level: "captain", MinSales: 5000000, CommissionRate: 10, Benefits: "Komisi 10%, mentoring grup"},
	{Level: "merchant", MinSales: 20000000, CommissionRate: 12, Benefits: "Komisi 12%, prioritas support"},
	{Level: "sultan", MinSales: 100000000, CommissionRate: 15, Benefits: "Komisi 15%, undian umroh"},
}
