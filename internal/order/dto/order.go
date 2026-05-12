package dto

type OrderItemReq struct {
	ProductID     string  `json:"product_id" validate:"required"`
	VariantID     string  `json:"variant_id" validate:"required"`
	ProductName   string  `json:"product_name"`
	VariantWeight string  `json:"variant_weight"`
	Price         float64 `json:"price" validate:"required,gt=0"`
	Quantity      int     `json:"quantity" validate:"required,min=1"`
	ImageURL      string  `json:"image_url"`
}

type ShippingAddressReq struct {
	FirstName  string `json:"first_name" validate:"required"`
	LastName   string `json:"last_name" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	PhoneCode  string `json:"phone_code" validate:"required"`
	Phone      string `json:"phone" validate:"required"`
	Address    string `json:"address" validate:"required"`
	Apartment  string `json:"apartment"`
	City       string `json:"city" validate:"required"`
	State      string `json:"state" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Country    string `json:"country" validate:"required"`
}

type CreateOrderReq struct {
	Items           []OrderItemReq     `json:"items" validate:"required,min=1"`
	ShippingAddress ShippingAddressReq `json:"shipping_address" validate:"required"`
}

type CreateOrderRes struct {
	OrderID         string  `json:"order_id"`
	RazorpayOrderID string  `json:"razorpay_order_id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	RazorpayKeyID   string  `json:"razorpay_key_id"`
}

type VerifyPaymentReq struct {
	RazorpayOrderID   string `json:"razorpay_order_id" validate:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
}

// GuestCheckoutReq is used when the customer is not logged in.
// An account is found-or-created by email (no password needed — customers use OTP).
type GuestCheckoutReq struct {
	FirstName       string             `json:"first_name" validate:"required"`
	LastName        string             `json:"last_name"`
	Email           string             `json:"email" validate:"required,email"`
	Items           []OrderItemReq     `json:"items" validate:"required,min=1"`
	ShippingAddress ShippingAddressReq `json:"shipping_address" validate:"required"`
}

type GuestCheckoutRes struct {
	AccessToken     string  `json:"access_token"`
	RefreshToken    string  `json:"refresh_token"`
	OrderID         string  `json:"order_id"`
	RazorpayOrderID string  `json:"razorpay_order_id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	RazorpayKeyID   string  `json:"razorpay_key_id"`
}

type ApproveDeliveryReq struct {
	AdminNotes string `json:"admin_notes"`
}

type RejectDeliveryReq struct {
	AdminNotes string `json:"admin_notes" validate:"required"`
}
