package model

import "time"

type Alias struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Model     string    `gorm:"size:100;not null" json:"-"`
	Target    string    `gorm:"size:100;not null" json:"target"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
