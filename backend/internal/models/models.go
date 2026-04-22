package models

import (
	"time"
)

type Lead struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	TelegramChatID  int64          `gorm:"uniqueIndex;not null" json:"telegram_chat_id"`
	Name            string         `json:"name"`
	Phone           string         `json:"phone"`
	City            string         `json:"city"`
	Budget          float64        `json:"budget"`
	PropertyType    string         `json:"property_type"`
	Timeline        string         `json:"timeline"`
	Score           int            `gorm:"default:0" json:"score"`
	Tag             string         `gorm:"default:'Curious'" json:"tag"`
	Status          string         `gorm:"default:'new'" json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type Property struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	City        string    `gorm:"not null" json:"city"`
	Price       float64   `gorm:"not null" json:"price"`
	Type        string    `gorm:"not null" json:"type"`
	Bedrooms    int       `json:"bedrooms"`
	Bathrooms   int       `json:"bathrooms"`
	AreaSqm     float64   `json:"area_sqm"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	LeadID    uint      `gorm:"not null;index" json:"lead_id"`
	Role      string    `gorm:"not null" json:"role"`
	Content   string    `gorm:"not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Stats struct {
	TotalLeads      int64   `json:"total_leads"`
	ConversionRate  float64 `json:"conversion_rate"`
	ActiveChatsToday int64  `json:"active_chats_today"`
	RecentActivity  []Lead  `json:"recent_activity"`
}
