package model

import "time"

type APIKey struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index;not null" json:"user_id"`
	APIKey    string     `gorm:"size:500;not null" json:"-"`
	ExpiredAt *time.Time `json:"expired_at"`
	Enabled   bool       `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
