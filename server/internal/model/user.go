package model

import (
	"llmux/internal/constant"
	"time"
)

// User represents a user in the system
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"`           // Never expose password in JSON
	Role      string    `gorm:"size:20;default:user" json:"role"`     // admin, user
	Status    string    `gorm:"size:20;default:active" json:"status"` // active, disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == constant.RoleAdmin
}

// IsActive checks if the user is active
func (u *User) IsActive() bool {
	return u.Status == constant.StatusActive
}
