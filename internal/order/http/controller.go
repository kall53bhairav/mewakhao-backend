package http

import (
	"ecom/internal/order/dto"
	"ecom/internal/order/service"
	"ecom/pkg/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/logger"
	"github.com/quangdangfit/gocommon/validation"
)

type OrderController struct {
	srv       *service.OrderService
	validator validation.Validation
}

func NewOrderController(srv *service.OrderService, validator validation.Validation) *OrderController {
	return &OrderController{srv: srv, validator: validator}
}

func getUserID(ctx *gin.Context) (string, error) {
	userID, exists := ctx.Get("userId")
	if !exists {
		return "", errors.New("user not authenticated")
	}
	id, ok := userID.(string)
	if !ok {
		return "", errors.New("invalid user ID")
	}
	return id, nil
}

func (ctrl *OrderController) GuestCheckout(ctx *gin.Context) {
	var req dto.GuestCheckoutReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if err := ctrl.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Validation failed")
		return
	}

	res, err := ctrl.srv.GuestCheckout(ctx, &req)
	if err != nil {
		logger.Errorf("GuestCheckout failed: %v", err)
		status := http.StatusInternalServerError
		if err.Error() == "an account with this email already exists — please check your password" {
			status = http.StatusUnauthorized
		}
		response.Error(ctx, status, err, err.Error())
		return
	}

	response.JSON(ctx, http.StatusCreated, res)
}

func (ctrl *OrderController) CreateOrder(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err, "Unauthorized")
		return
	}

	var req dto.CreateOrderReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if err := ctrl.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Validation failed")
		return
	}

	res, err := ctrl.srv.CreateOrder(ctx, userID, &req)
	if err != nil {
		logger.Errorf("CreateOrder failed: %v", err)
		response.Error(ctx, http.StatusInternalServerError, err, "Failed to create order")
		return
	}

	response.JSON(ctx, http.StatusCreated, res)
}

func (ctrl *OrderController) VerifyPayment(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err, "Unauthorized")
		return
	}

	orderID := ctx.Param("id")

	var req dto.VerifyPaymentReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if err := ctrl.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Validation failed")
		return
	}

	order, err := ctrl.srv.VerifyPayment(ctx, userID, orderID, &req)
	if err != nil {
		logger.Errorf("VerifyPayment failed: %v", err)
		status := http.StatusInternalServerError
		if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		} else if err.Error() == "order not found" {
			status = http.StatusNotFound
		} else if err.Error() == "invalid payment signature" {
			status = http.StatusBadRequest
		}
		response.Error(ctx, status, err, err.Error())
		return
	}

	response.JSON(ctx, http.StatusOK, order)
}

func (ctrl *OrderController) GetMyOrders(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err, "Unauthorized")
		return
	}

	orders, err := ctrl.srv.GetUserOrders(ctx, userID)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err, "Failed to fetch orders")
		return
	}

	response.JSON(ctx, http.StatusOK, orders)
}

func (ctrl *OrderController) GetOrder(ctx *gin.Context) {
	userID, err := getUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err, "Unauthorized")
		return
	}

	orderID := ctx.Param("id")
	role, _ := ctx.Get("role")
	isAdmin := role == "admin"

	order, err := ctrl.srv.GetOrderByID(ctx, userID, orderID, isAdmin)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "order not found" {
			status = http.StatusNotFound
		} else if err.Error() == "unauthorized" {
			status = http.StatusForbidden
		}
		response.Error(ctx, status, err, err.Error())
		return
	}

	response.JSON(ctx, http.StatusOK, order)
}

// Admin handlers

func (ctrl *OrderController) GetAllOrders(ctx *gin.Context) {
	orders, err := ctrl.srv.GetAllOrders(ctx)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err, "Failed to fetch orders")
		return
	}

	response.JSON(ctx, http.StatusOK, orders)
}

func (ctrl *OrderController) GetDeliveryRequests(ctx *gin.Context) {
	status := ctx.Query("status")
	requests, err := ctrl.srv.GetDeliveryRequests(ctx, status)
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, err, "Failed to fetch delivery requests")
		return
	}

	response.JSON(ctx, http.StatusOK, requests)
}

func (ctrl *OrderController) ApproveDelivery(ctx *gin.Context) {
	adminID, err := getUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err, "Unauthorized")
		return
	}

	requestID := ctx.Param("id")

	var req dto.ApproveDeliveryReq
	_ = ctx.ShouldBindJSON(&req) // optional body

	result, err := ctrl.srv.ApproveDelivery(ctx, requestID, adminID, &req)
	if err != nil {
		logger.Errorf("ApproveDelivery failed: %v", err)
		status := http.StatusInternalServerError
		if err.Error() == "delivery request not found" {
			status = http.StatusNotFound
		}
		response.Error(ctx, status, err, err.Error())
		return
	}

	response.JSON(ctx, http.StatusOK, result)
}

func (ctrl *OrderController) RejectDelivery(ctx *gin.Context) {
	adminID, err := getUserID(ctx)
	if err != nil {
		response.Error(ctx, http.StatusUnauthorized, err, "Unauthorized")
		return
	}

	requestID := ctx.Param("id")

	var req dto.RejectDeliveryReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if err := ctrl.validator.ValidateStruct(req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err, "admin_notes is required for rejection")
		return
	}

	result, err := ctrl.srv.RejectDelivery(ctx, requestID, adminID, &req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "delivery request not found" {
			status = http.StatusNotFound
		}
		response.Error(ctx, status, err, err.Error())
		return
	}

	response.JSON(ctx, http.StatusOK, result)
}
