package domain

import "time"

// Notification is the in-app inbox record persisted to the DB.
type Notification struct {
	ID             string    `gorm:"primaryKey"`
	RecipientID    string    `gorm:"not null;index"`
	Title          string    `gorm:"not null"`
	Body           string    `gorm:"not null"`
	Category       string    `gorm:"not null"`
	Read           bool      `gorm:"default:false"`
	Data           string    `gorm:"type:text"` // JSON-encoded map[string]string
	IdempotencyKey string    `gorm:"uniqueIndex;default:null"`
	CreatedAt      time.Time `gorm:"autoCreateTime;index"`
}

// NotificationPreference stores per-user, per-category channel opt-ins.
type NotificationPreference struct {
	ID        string    `gorm:"primaryKey"`
	UserID    string    `gorm:"not null;index"`
	Category  string    `gorm:"not null"` // matches Category enum string
	Channels  string    `gorm:"type:text"` // JSON-encoded []string of Channel enum strings
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// SendInput is passed from the handler to the service.
type SendInput struct {
	RecipientUserID     string
	RecipientBusinessID string
	// Override contact details (optional — service falls back to stored values)
	Email     string
	Phone     string
	PushToken string

	TemplateID     string
	Variables      map[string]string
	Category       string   // Category enum string
	Channels       []string // Channel enum strings; empty = template defaults
	IdempotencyKey string
}

// ChannelResult is the per-channel outcome of a Send call.
type ChannelResult struct {
	Channel string
	Status  string // DeliveryStatus enum string
	Detail  string
}
