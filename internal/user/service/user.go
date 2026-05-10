package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"ecom/internal/user/dto"
	"ecom/internal/user/entity"
	"ecom/internal/user/repository"
	"ecom/pkg/config"
	"ecom/pkg/email"
	"ecom/pkg/jwt"

	"github.com/google/uuid"
	"github.com/quangdangfit/gocommon/logger"
	"github.com/quangdangfit/gocommon/validation"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo   *repository.UserRepo
	mailer *email.Sender
}

func NewUserService(validator validation.Validation, repo *repository.UserRepo) *UserService {
	cfg := config.GetEnv()
	mailer := email.NewSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	return &UserService{repo: repo, mailer: mailer}
}

// Login is for admin accounts only. Customer accounts use OTP.
func (s *UserService) Login(ctx context.Context, req *dto.LoginReq) (*entity.User, string, string, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	if user.Password == "" {
		return nil, "", "", errors.New("this account uses OTP sign-in — please use the customer login page")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	tokenData := map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}
	return user, jwt.GenerateAccessToken(tokenData), jwt.GenerateRefreshToken(tokenData), nil
}

// CheckEmail returns whether the email is already registered.
func (s *UserService) CheckEmail(ctx context.Context, email string) bool {
	user, err := s.repo.GetUserByEmail(ctx, email)
	return err == nil && user != nil
}

// SendOTP generates a 6-digit OTP and emails it to the user.
// If the email is not registered and FirstName is provided, an account is created first.
// Returns (isNewUser, error).
func (s *UserService) SendOTP(ctx context.Context, req *dto.SendOTPReq) (bool, error) {
	isNew := false

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// User not found
		if req.FirstName == "" {
			return false, errors.New("no account found — please provide your name to register")
		}
		newUser := &entity.User{
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Email:     req.Email,
		}
		if createErr := s.repo.Create(ctx, newUser); createErr != nil {
			logger.Errorf("SendOTP: failed to create user %s: %v", req.Email, createErr)
			return false, errors.New("failed to create account")
		}
		user = newUser
		isNew = true
	}
	_ = user

	code, err := generateOTP()
	if err != nil {
		return isNew, fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Invalidate any previous OTPs for this email
	_ = s.repo.DeleteOTPsByEmail(ctx, req.Email)

	otp := &entity.OTP{
		Email:     req.Email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.repo.CreateOTP(ctx, otp); err != nil {
		return isNew, fmt.Errorf("failed to store OTP: %w", err)
	}

	subject := "Your MewaKhao sign-in code"
	body := email.OTPEmailBody(code)
	if err := s.mailer.Send(req.Email, subject, body); err != nil {
		logger.Errorf("SendOTP: email delivery failed for %s: %v", req.Email, err)
		// Don't fail the request — OTP is in DB, dev console shows it
	}

	return isNew, nil
}

// VerifyOTP checks the OTP, marks it used, and returns a JWT pair.
func (s *UserService) VerifyOTP(ctx context.Context, req *dto.VerifyOTPReq) (*entity.User, string, string, error) {
	otp, err := s.repo.GetActiveOTP(ctx, req.Email)
	if err != nil {
		return nil, "", "", errors.New("invalid or expired OTP")
	}

	if otp.Code != req.Code {
		return nil, "", "", errors.New("incorrect OTP")
	}

	if err := s.repo.MarkOTPUsed(ctx, otp.ID); err != nil {
		logger.Errorf("VerifyOTP: could not mark OTP used: %v", err)
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", "", errors.New("account not found")
	}

	tokenData := map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}
	return user, jwt.GenerateAccessToken(tokenData), jwt.GenerateRefreshToken(tokenData), nil
}

// ForgotPassword sends a password reset link to the admin's email.
// Always returns nil to avoid revealing whether an email is registered.
func (s *UserService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordReq) error {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil || user == nil || user.Password == "" {
		return nil
	}

	token := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)
	user.PasswordResetToken = token
	user.PasswordResetExpiresAt = &expiresAt
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	cfg := config.GetEnv()
	resetURL := fmt.Sprintf("%s/admin/reset-password?token=%s", cfg.FrontendURL, token)
	body := email.PasswordResetEmailBody(user.FirstName, resetURL)
	if err := s.mailer.Send(req.Email, "Reset your MewaKhao admin password", body); err != nil {
		logger.Errorf("ForgotPassword: email failed for %s: %v", req.Email, err)
	}

	return nil
}

// ResetPassword validates the reset token and updates the admin's password.
func (s *UserService) ResetPassword(ctx context.Context, req *dto.ResetPasswordReq) error {
	user, err := s.repo.GetUserByResetToken(ctx, req.Token)
	if err != nil || user == nil {
		return errors.New("invalid or expired reset link")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = string(hashed)
	if err := s.repo.ClearPasswordResetToken(ctx, user.ID); err != nil {
		logger.Errorf("ResetPassword: failed to clear token for %s: %v", user.ID, err)
	}
	user.PasswordResetToken = ""
	user.PasswordResetExpiresAt = nil
	return s.repo.Update(ctx, user)
}

// UpdateProfile updates a user's name fields.
func (s *UserService) UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileReq) (*entity.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return user, nil
}

func generateOTP() (string, error) {
	b := make([]byte, 3) // 3 bytes → 6 decimal digits via modulo
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
