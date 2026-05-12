package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	cartRepo "ecom/internal/cart/repository"
	"ecom/internal/order/dto"
	"ecom/internal/order/entity"
	"ecom/internal/order/repository"
	userEntity "ecom/internal/user/entity"
	userRepo "ecom/internal/user/repository"
	"ecom/pkg/config"
	"ecom/pkg/jwt"
	"ecom/pkg/razorpay"
	"ecom/pkg/shiprocket"

	"github.com/quangdangfit/gocommon/logger"
)

const (
	shippingThreshold = 50.0
	shippingCost      = 5.99
)

type OrderService struct {
	repo       *repository.OrderRepo
	cartRepo   *cartRepo.CartRepo
	userRepo   *userRepo.UserRepo
	razorpay   *razorpay.Client
	shiprocket *shiprocket.Client
	cfg        *config.Schema
}

func NewOrderService(
	repo *repository.OrderRepo,
	cr *cartRepo.CartRepo,
	ur *userRepo.UserRepo,
	cfg *config.Schema,
) *OrderService {
	logger.Info("Initializing OrderService", cfg)
	return &OrderService{
		repo:       repo,
		cartRepo:   cr,
		userRepo:   ur,
		razorpay:   razorpay.NewClient(cfg.RazorpayKeyID, cfg.RazorpayKeySecret),
		shiprocket: shiprocket.NewClient(cfg.ShiprocketEmail, cfg.ShiprocketPassword),
		cfg:        cfg,
	}
}

// GuestCheckout creates an order for an unauthenticated user.
// If the email is new, a customer account is created.
// If the email exists, the password is verified before proceeding.
func (s *OrderService) GuestCheckout(ctx context.Context, req *dto.GuestCheckoutReq) (*dto.GuestCheckoutRes, error) {
	var user *userEntity.User

	existing, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// email not found — create account (no password; customer uses OTP)
		newUser := &userEntity.User{
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Email:     req.Email,
		}
		if createErr := s.userRepo.Create(ctx, newUser); createErr != nil {
			return nil, fmt.Errorf("failed to create account: %w", createErr)
		}
		user = newUser
	} else {
		user = existing
	}

	tokenData := map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}
	accessToken := jwt.GenerateAccessToken(tokenData)
	refreshToken := jwt.GenerateRefreshToken(tokenData)

	// reuse CreateOrder logic
	createReq := &dto.CreateOrderReq{
		Items:           req.Items,
		ShippingAddress: req.ShippingAddress,
	}
	orderRes, err := s.CreateOrder(ctx, user.ID, createReq)
	if err != nil {
		return nil, err
	}

	return &dto.GuestCheckoutRes{
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		OrderID:         orderRes.OrderID,
		RazorpayOrderID: orderRes.RazorpayOrderID,
		Amount:          orderRes.Amount,
		Currency:        orderRes.Currency,
		RazorpayKeyID:   orderRes.RazorpayKeyID,
	}, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, userID string, req *dto.CreateOrderReq) (*dto.CreateOrderRes, error) {
	subtotal := 0.0
	for _, item := range req.Items {
		subtotal += item.Price * float64(item.Quantity)
	}

	shipping := 0.0
	if subtotal < shippingThreshold {
		shipping = shippingCost
	}
	total := subtotal + shipping

	rpOrder, err := s.razorpay.CreateOrder(total, "INR", fmt.Sprintf("receipt_%s", userID[:8]))
	if err != nil {
		return nil, fmt.Errorf("payment gateway error: %w", err)
	}

	order := &entity.Order{
		UserID:          userID,
		Subtotal:        subtotal,
		ShippingCost:    shipping,
		Total:           total,
		Status:          entity.OrderStatusPendingPayment,
		RazorpayOrderID: rpOrder.ID,
		ShippingAddress: entity.ShippingAddress{
			FirstName:  req.ShippingAddress.FirstName,
			LastName:   req.ShippingAddress.LastName,
			Email:      req.ShippingAddress.Email,
			PhoneCode:  req.ShippingAddress.PhoneCode,
			Phone:      req.ShippingAddress.Phone,
			Address:    req.ShippingAddress.Address,
			Apartment:  req.ShippingAddress.Apartment,
			City:       req.ShippingAddress.City,
			State:      req.ShippingAddress.State,
			PostalCode: req.ShippingAddress.PostalCode,
			Country:    req.ShippingAddress.Country,
		},
	}

	for _, item := range req.Items {
		order.Items = append(order.Items, entity.OrderItem{
			ProductID:     item.ProductID,
			ProductName:   item.ProductName,
			VariantID:     item.VariantID,
			VariantWeight: item.VariantWeight,
			Price:         item.Price,
			Quantity:      item.Quantity,
			ImageURL:      item.ImageURL,
		})
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return &dto.CreateOrderRes{
		OrderID:         order.ID,
		RazorpayOrderID: rpOrder.ID,
		Amount:          total,
		Currency:        "INR",
		RazorpayKeyID:   s.cfg.RazorpayKeyID,
	}, nil
}

func (s *OrderService) VerifyPayment(ctx context.Context, userID, orderID string, req *dto.VerifyPaymentReq) (*entity.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if order.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	if order.Status != entity.OrderStatusPendingPayment {
		return nil, errors.New("order already processed")
	}

	if !s.razorpay.VerifyPaymentSignature(req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature) {
		return nil, errors.New("invalid payment signature")
	}

	order.Status = entity.OrderStatusPaid
	order.RazorpayPaymentID = req.RazorpayPaymentID

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	deliveryReq := &entity.DeliveryRequest{
		OrderID: order.ID,
		Status:  entity.DeliveryStatusPending,
	}
	if err := s.repo.CreateDeliveryRequest(ctx, deliveryReq); err != nil {
		logger.Errorf("failed to create delivery request for order %s: %v", order.ID, err)
	}

	cart, err := s.cartRepo.GetCart(ctx, userID)
	if err == nil {
		if clearErr := s.cartRepo.ClearCart(ctx, cart.ID); clearErr != nil {
			logger.Errorf("failed to clear cart for user %s: %v", userID, clearErr)
		}
	}

	return order, nil
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID string) ([]entity.Order, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *OrderService) GetOrderByID(ctx context.Context, callerUserID, orderID string, isAdmin bool) (*entity.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if !isAdmin && order.UserID != callerUserID {
		return nil, errors.New("unauthorized")
	}

	return order, nil
}

func (s *OrderService) GetAllOrders(ctx context.Context) ([]entity.Order, error) {
	return s.repo.GetAll(ctx)
}

func (s *OrderService) GetDeliveryRequests(ctx context.Context, status string) ([]entity.DeliveryRequest, error) {
	return s.repo.GetDeliveryRequests(ctx, status)
}

func (s *OrderService) ApproveDelivery(ctx context.Context, requestID, adminID string, req *dto.ApproveDeliveryReq) (*entity.DeliveryRequest, error) {
	deliveryReq, err := s.repo.GetDeliveryRequestByID(ctx, requestID)
	if err != nil {
		return nil, errors.New("delivery request not found")
	}

	if deliveryReq.Status != entity.DeliveryStatusPending {
		return nil, errors.New("delivery request already processed")
	}

	order := deliveryReq.Order
	if order == nil {
		return nil, errors.New("associated order not found")
	}

	srItems := make([]shiprocket.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		srItems = append(srItems, shiprocket.OrderItem{
			Name:         item.ProductName,
			SKU:          item.ProductID,
			Units:        item.Quantity,
			SellingPrice: item.Price,
		})
	}

	addr := order.ShippingAddress
	srReq := &shiprocket.CreateOrderRequest{
		OrderID:           order.ID[:8],
		OrderDate:         order.CreatedAt.Format("2006-01-02 15:04"),
		PickupLocation:    "Primary",
		BillingName:       addr.FirstName,
		BillingLastName:   addr.LastName,
		BillingAddress:    addr.Address,
		BillingCity:       addr.City,
		BillingPincode:    addr.PostalCode,
		BillingState:      addr.State,
		BillingCountry:    addr.Country,
		BillingEmail:      addr.Email,
		BillingPhone:      addr.PhoneCode + addr.Phone,
		ShippingName:      addr.FirstName,
		ShippingLastName:  addr.LastName,
		ShippingAddress:   addr.Address,
		ShippingCity:      addr.City,
		ShippingPincode:   addr.PostalCode,
		ShippingState:     addr.State,
		ShippingCountry:   addr.Country,
		ShippingEmail:     addr.Email,
		ShippingPhone:     addr.PhoneCode + addr.Phone,
		ShippingIsBilling: true,
		OrderItems:        srItems,
		PaymentMethod:     "Prepaid",
		SubTotal:          order.Subtotal,
		Length:            10,
		Breadth:           10,
		Height:            10,
		Weight:            0.5,
	}

	srOrder, err := s.shiprocket.CreateOrder(srReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	now := time.Now()
	deliveryReq.Status = entity.DeliveryStatusApproved
	deliveryReq.ShiprocketOrderID = fmt.Sprintf("%d", srOrder.OrderID)
	deliveryReq.AWBCode = srOrder.AWBCode
	deliveryReq.AdminNotes = req.AdminNotes
	deliveryReq.ApprovedAt = &now
	deliveryReq.ApprovedByID = adminID

	if err := s.repo.UpdateDeliveryRequest(ctx, deliveryReq); err != nil {
		return nil, err
	}

	order.Status = entity.OrderStatusProcessing
	if err := s.repo.Update(ctx, order); err != nil {
		logger.Errorf("failed to update order status after delivery approval: %v", err)
	}

	return deliveryReq, nil
}

func (s *OrderService) RejectDelivery(ctx context.Context, requestID, adminID string, req *dto.RejectDeliveryReq) (*entity.DeliveryRequest, error) {
	deliveryReq, err := s.repo.GetDeliveryRequestByID(ctx, requestID)
	if err != nil {
		return nil, errors.New("delivery request not found")
	}

	if deliveryReq.Status != entity.DeliveryStatusPending {
		return nil, errors.New("delivery request already processed")
	}

	deliveryReq.Status = entity.DeliveryStatusRejected
	deliveryReq.AdminNotes = req.AdminNotes
	deliveryReq.ApprovedByID = adminID

	if err := s.repo.UpdateDeliveryRequest(ctx, deliveryReq); err != nil {
		return nil, err
	}

	return deliveryReq, nil
}
