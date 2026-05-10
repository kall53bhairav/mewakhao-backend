package razorpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const baseURL = "https://api.razorpay.com/v1"

type Client struct {
	keyID     string
	keySecret string
}

func NewClient(keyID, keySecret string) *Client {
	return &Client{keyID: keyID, keySecret: keySecret}
}

type CreateOrderRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
}

type CreateOrderResponse struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
	Status   string `json:"status"`
}

func (c *Client) CreateOrder(amount float64, currency, receipt string) (*CreateOrderResponse, error) {
	payload := CreateOrderRequest{
		Amount:   int64(amount * 100), // paise
		Currency: currency,
		Receipt:  receipt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/orders", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.keyID, c.keySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("razorpay error %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateOrderResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// VerifyPaymentSignature validates the Razorpay HMAC-SHA256 signature.
// Signature = HMAC-SHA256(razorpay_order_id + "|" + razorpay_payment_id, key_secret)
func (c *Client) VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(c.keySecret))
	h.Write([]byte(data))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
