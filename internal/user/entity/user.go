package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ecom/pkg/utils"
)

type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleCustomer UserRole = "customer"
)

type User struct {
	ID                    string     `json:"id" gorm:"unique;not null;index;primary_key"`
	FirstName             string     `json:"first_name"`
	LastName              string     `json:"last_name"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	DeletedAt             *time.Time `json:"deleted_at" gorm:"index"`
	Email                 string     `json:"email" gorm:"unique;not null;index:idx_user_email"`
	Password              string     `json:"password"`
	Role                  UserRole   `json:"role"`
	PasswordResetToken    string     `json:"-" gorm:"index"`
	PasswordResetExpiresAt *time.Time `json:"-"`
}

func (user *User) BeforeCreate(_ *gorm.DB) error {
	user.ID = uuid.New().String()
	// Only hash non-empty passwords. Customer accounts use OTP and have no password.
	if user.Password != "" {
		user.Password = utils.HashAndSalt([]byte(user.Password))
	}
	if user.Role == "" {
		user.Role = UserRoleCustomer
	}
	return nil
}
