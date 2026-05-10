package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusProcessing     OrderStatus = "processing"
	OrderStatusShipped        OrderStatus = "shipped"
	OrderStatusDelivered      OrderStatus = "delivered"
	OrderStatusCancelled      OrderStatus = "cancelled"
)

type DeliveryStatus string

const (
	DeliveryStatusPending  DeliveryStatus = "pending"
	DeliveryStatusApproved DeliveryStatus = "approved"
	DeliveryStatusRejected DeliveryStatus = "rejected"
	DeliveryStatusShipped  DeliveryStatus = "shipped"
)

type Order struct {
	ID                string          `json:"id" gorm:"primaryKey"`
	UserID            string          `json:"user_id" gorm:"index;not null"`
	Items             []OrderItem     `json:"items" gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	ShippingAddress   ShippingAddress `json:"shipping_address" gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	Subtotal          float64         `json:"subtotal"`
	ShippingCost      float64         `json:"shipping_cost"`
	Total             float64         `json:"total"`
	Status            OrderStatus     `json:"status" gorm:"default:'pending_payment'"`
	RazorpayOrderID   string          `json:"razorpay_order_id"`
	RazorpayPaymentID string          `json:"razorpay_payment_id"`
	DeliveryRequest   *DeliveryRequest `json:"delivery_request,omitempty" gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (o *Order) BeforeCreate(_ *gorm.DB) error {
	o.ID = uuid.New().String()
	return nil
}

type OrderItem struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	OrderID       string    `json:"order_id" gorm:"index"`
	ProductID     string    `json:"product_id"`
	ProductName   string    `json:"product_name"`
	VariantID     string    `json:"variant_id"`
	VariantWeight string    `json:"variant_weight"`
	Price         float64   `json:"price"`
	Quantity      int       `json:"quantity"`
	ImageURL      string    `json:"image_url"`
	CreatedAt     time.Time `json:"created_at"`
}

func (oi *OrderItem) BeforeCreate(_ *gorm.DB) error {
	oi.ID = uuid.New().String()
	return nil
}

type ShippingAddress struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	OrderID    string    `json:"order_id" gorm:"uniqueIndex"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Address    string    `json:"address"`
	Apartment  string    `json:"apartment"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	PostalCode string    `json:"postal_code"`
	Country    string    `json:"country"`
	CreatedAt  time.Time `json:"created_at"`
}

func (sa *ShippingAddress) BeforeCreate(_ *gorm.DB) error {
	sa.ID = uuid.New().String()
	return nil
}

type DeliveryRequest struct {
	ID                string         `json:"id" gorm:"primaryKey"`
	OrderID           string         `json:"order_id" gorm:"uniqueIndex"`
	Status            DeliveryStatus `json:"status" gorm:"default:'pending'"`
	ShiprocketOrderID string         `json:"shiprocket_order_id"`
	AWBCode           string         `json:"awb_code"`
	CourierName       string         `json:"courier_name"`
	TrackingURL       string         `json:"tracking_url"`
	AdminNotes        string         `json:"admin_notes"`
	ApprovedAt        *time.Time     `json:"approved_at"`
	ApprovedByID      string         `json:"approved_by_id"`
	Order             *Order         `json:"order,omitempty" gorm:"foreignKey:OrderID"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (dr *DeliveryRequest) BeforeCreate(_ *gorm.DB) error {
	dr.ID = uuid.New().String()
	return nil
}
