package model

import (
	"time"
)

// Provider represents an LLM provider configuration
type Provider struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`                   // Creator
	ProviderID   string    `gorm:"uniqueIndex;size:50;not null" json:"provider_id"` // e.g., openai, anthropic
	ProviderName string    `gorm:"size:100;not null" json:"provider_name"`
	BaseURL      string    `gorm:"size:500;not null" json:"base_url"`
	APIKey       string    `gorm:"size:500;not null" json:"-"`   // Never expose API key in JSON
	Type         string    `gorm:"size:50;not null" json:"type"` // openai, anthropic
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
