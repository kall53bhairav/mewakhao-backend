package shiprocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const baseURL = "https://apiv2.shiprocket.in/v1/external"

type Client struct {
	email    string
	password string
	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func NewClient(email, password string) *Client {
	return &Client{email: email, password: password}
}

func (c *Client) authenticate() error {
	payload := map[string]string{
		"email":    c.email,
		"password": c.password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if result.Token == "" {
		return fmt.Errorf("shiprocket authentication failed: %s", string(respBody))
	}

	c.token = result.Token
	c.tokenExp = time.Now().Add(9 * 24 * time.Hour) // tokens last 10 days
	return nil
}

func (c *Client) getToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token == "" || time.Now().After(c.tokenExp) {
		if err := c.authenticate(); err != nil {
			return "", err
		}
	}
	return c.token, nil
}

type OrderItem struct {
	Name         string  `json:"name"`
	SKU          string  `json:"sku"`
	Units        int     `json:"units"`
	SellingPrice float64 `json:"selling_price"`
}

type CreateOrderRequest struct {
	OrderID           string      `json:"order_id"`
	OrderDate         string      `json:"order_date"`
	PickupLocation    string      `json:"pickup_location"`
	BillingName       string      `json:"billing_customer_name"`
	BillingLastName   string      `json:"billing_last_name"`
	BillingAddress    string      `json:"billing_address"`
	BillingCity       string      `json:"billing_city"`
	BillingPincode    string      `json:"billing_pincode"`
	BillingState      string      `json:"billing_state"`
	BillingCountry    string      `json:"billing_country"`
	BillingEmail      string      `json:"billing_email"`
	BillingPhone      string      `json:"billing_phone"`
	ShippingName      string      `json:"shipping_customer_name"`
	ShippingLastName  string      `json:"shipping_last_name"`
	ShippingAddress   string      `json:"shipping_address"`
	ShippingCity      string      `json:"shipping_city"`
	ShippingPincode   string      `json:"shipping_pincode"`
	ShippingState     string      `json:"shipping_state"`
	ShippingCountry   string      `json:"shipping_country"`
	ShippingEmail     string      `json:"shipping_email"`
	ShippingPhone     string      `json:"shipping_phone"`
	ShippingIsBilling bool        `json:"shipping_is_billing"`
	OrderItems        []OrderItem `json:"order_items"`
	PaymentMethod     string      `json:"payment_method"`
	SubTotal          float64     `json:"sub_total"`
	Length            float64     `json:"length"`
	Breadth           float64     `json:"breadth"`
	Height            float64     `json:"height"`
	Weight            float64     `json:"weight"`
}

type CreateOrderResponse struct {
	OrderID    int    `json:"order_id"`
	ShipmentID int    `json:"shipment_id"`
	Status     string `json:"status"`
	AWBCode    string `json:"awb_code"`
}

func (c *Client) CreateOrder(req *CreateOrderRequest) (*CreateOrderResponse, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", baseURL+"/orders/create/adhoc", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("shiprocket error %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
