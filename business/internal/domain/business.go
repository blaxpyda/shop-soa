package domain

import "time"

type Business struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	OwnerID     string    `gorm:"not null;index" json:"owner_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	Country     string    `json:"country"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type CreateBusinessInput struct {
	OwnerID     string
	Name        string
	Description string
	Email       string
	Phone       string
	Address     string
	City        string
	Country     string
}

type UpdateBusinessInput struct {
	Name        string
	Description string
	Email       string
	Phone       string
	Address     string
	City        string
	Country     string
	IsActive    *bool
}
