package repository

import (
	"context"
	"time"

	"ecom/internal/user/entity"
	"ecom/pkg/dbs"
)

type UserRepo struct {
	db *dbs.Database
}

func NewUserRepository(db *dbs.Database) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	return r.db.Create(ctx, user)
}

func (r *UserRepo) Update(ctx context.Context, user *entity.User) error {
	return r.db.Update(ctx, user)
}

func (r *UserRepo) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	if err := r.db.FindById(ctx, id, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	query := dbs.NewQuery("email = ?", email)
	if err := r.db.FindOne(ctx, &user, dbs.WithQuery(query)); err != nil {
		return nil, err
	}

	return &user, nil
}

// OTP methods

func (r *UserRepo) CreateOTP(ctx context.Context, otp *entity.OTP) error {
	return r.db.Create(ctx, otp)
}

// DeleteOTPsByEmail removes all existing OTPs for an email before issuing a new one.
func (r *UserRepo) DeleteOTPsByEmail(ctx context.Context, email string) error {
	return r.db.GetDB().WithContext(ctx).
		Where("email = ?", email).
		Delete(&entity.OTP{}).Error
}

func (r *UserRepo) GetActiveOTP(ctx context.Context, email string) (*entity.OTP, error) {
	var otp entity.OTP
	err := r.db.GetDB().WithContext(ctx).
		Where("email = ? AND used = false AND expires_at > ?", email, time.Now()).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *UserRepo) MarkOTPUsed(ctx context.Context, id string) error {
	return r.db.GetDB().WithContext(ctx).
		Model(&entity.OTP{}).
		Where("id = ?", id).
		Update("used", true).Error
}

func (r *UserRepo) GetUserByResetToken(ctx context.Context, token string) (*entity.User, error) {
	var user entity.User
	err := r.db.GetDB().WithContext(ctx).
		Where("password_reset_token = ? AND password_reset_expires_at > ?", token, time.Now()).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) ClearPasswordResetToken(ctx context.Context, userID string) error {
	return r.db.GetDB().WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_reset_token":      "",
			"password_reset_expires_at": nil,
		}).Error
}
