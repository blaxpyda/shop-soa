package domain

import "time"

const (
	RoleSuperAdmin = "super-admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

type User struct {
	ID               string    `gorm:"primaryKey;type:uuid" json:"id"`
	Email            string    `gorm:"uniqueIndex:idx_users_email,where:email <> ''" json:"email"`
	Phone            string    `gorm:"uniqueIndex:idx_users_phone,where:phone <> ''" json:"phone"`
	Password         string    `gorm:"not null" json:"password"`
	FirstName        string    `gorm:"not null" json:"first_name"`
	LastName         string    `gorm:"not null" json:"last_name"`
	Role             string    `gorm:"default:'user'" json:"role"`
	BusinessID       string    `json:"business_id"`
	Address          string    `json:"address"`
	City             string    `json:"city"`
	Country          string    `json:"country"`
	IsVerified       bool      `gorm:"default:false" json:"is_verified"`
	VerificationCode string    `json:"verification_code"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type CreateUserInput struct {
	Email      string
	Phone      string
	Password   string
	FirstName  string
	LastName   string
	Role       string
	BusinessID string
}

type UpdateUserInput struct {
	FirstName  string
	LastName   string
	Email      string
	Phone      string
	Address    string
	City       string
	Country    string
	Role       string
	BusinessID string
}

type ListUsersFilter struct {
	Page       int
	PageSize   int
	Role       string
	BusinessID string
}
