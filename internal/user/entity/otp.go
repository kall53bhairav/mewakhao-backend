package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OTP struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"index;not null"`
	Code      string    `json:"code" gorm:"not null"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

func (o *OTP) BeforeCreate(_ *gorm.DB) error {
	o.ID = uuid.New().String()
	return nil
}

func (o *OTP) IsValid() bool {
	return !o.Used && time.Now().Before(o.ExpiresAt)
}
