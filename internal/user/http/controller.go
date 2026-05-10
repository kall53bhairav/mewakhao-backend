package http

import (
	"ecom/internal/user/dto"
	"ecom/internal/user/service"
	"ecom/pkg/response"
	"ecom/pkg/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/logger"
	"github.com/quangdangfit/gocommon/validation"
)

type UserController struct {
	srv       *service.UserService
	validator validation.Validation
}

func NewUserController(srv *service.UserService, validator validation.Validation) *UserController {
	return &UserController{srv: srv, validator: validator}
}

// Login is for admin accounts only (password-based).
func (c *UserController) Login(ctx *gin.Context) {
	var req dto.LoginReq
	if err := ctx.ShouldBindJSON(&req); ctx.Request.Body == nil || err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	if err := c.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}

	user, accessToken, refreshToken, err := c.srv.Login(ctx, &req)
	if err != nil {
		logger.Error("Login failed", err)
		response.Error(ctx, http.StatusUnauthorized, err, err.Error())
		return
	}

	var res dto.LoginRes
	utils.Copy(&res.User, &user)
	res.AccessToken = accessToken
	res.RefreshToken = refreshToken
	response.JSON(ctx, http.StatusOK, res)
}

// CheckEmail returns whether an email is already registered.
func (c *UserController) CheckEmail(ctx *gin.Context) {
	var req dto.CheckEmailReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	response.JSON(ctx, http.StatusOK, dto.CheckEmailRes{
		Exists: c.srv.CheckEmail(ctx, req.Email),
	})
}

// SendOTP generates an OTP and emails it. Creates the account if name is supplied and email is new.
func (c *UserController) SendOTP(ctx *gin.Context) {
	var req dto.SendOTPReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	if err := c.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}

	_, err := c.srv.SendOTP(ctx, &req)
	if err != nil {
		logger.Errorf("SendOTP: %v", err)
		status := http.StatusInternalServerError
		if err.Error() == "no account found — please provide your name to register" {
			status = http.StatusNotFound
		}
		response.Error(ctx, status, err, err.Error())
		return
	}

	response.JSON(ctx, http.StatusOK, gin.H{"message": "OTP sent to your email"})
}

// VerifyOTP checks the OTP and returns a JWT pair.
func (c *UserController) VerifyOTP(ctx *gin.Context) {
	var req dto.VerifyOTPReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	if err := c.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}

	user, accessToken, refreshToken, err := c.srv.VerifyOTP(ctx, &req)
	if err != nil {
		logger.Errorf("VerifyOTP: %v", err)
		response.Error(ctx, http.StatusUnauthorized, err, err.Error())
		return
	}

	var res dto.VerifyOTPRes
	utils.Copy(&res.User, &user)
	res.AccessToken = accessToken
	res.RefreshToken = refreshToken
	response.JSON(ctx, http.StatusOK, res)
}

func (c *UserController) GetMe(ctx *gin.Context) {
	user, ok := ctx.Get("user")
	if !ok {
		response.Error(ctx, http.StatusBadRequest, errors.New("invalid user"), "Invalid user")
		return
	}
	response.JSON(ctx, http.StatusOK, user)
}

func (c *UserController) UpdateProfile(ctx *gin.Context) {
	userID := ctx.GetString("userId")
	if userID == "" {
		response.Error(ctx, http.StatusUnauthorized, errors.New("unauthorized"), "Unauthorized")
		return
	}

	var req dto.UpdateProfileReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	if err := c.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}

	user, err := c.srv.UpdateProfile(ctx, userID, &req)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err, err.Error())
		return
	}

	var res dto.User
	utils.Copy(&res, &user)
	response.JSON(ctx, http.StatusOK, res)
}

func (c *UserController) ForgotPassword(ctx *gin.Context) {
	var req dto.ForgotPasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	_ = c.srv.ForgotPassword(ctx, &req)
	response.JSON(ctx, http.StatusOK, gin.H{"message": "If that email is registered, a reset link has been sent"})
}

func (c *UserController) ResetPassword(ctx *gin.Context) {
	var req dto.ResetPasswordReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	if err := c.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid parameters")
		return
	}
	if err := c.srv.ResetPassword(ctx, &req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, err.Error())
		return
	}
	response.JSON(ctx, http.StatusOK, gin.H{"message": "Password reset successfully"})
}
