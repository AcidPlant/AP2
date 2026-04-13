package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPaymentClient(baseURL string, httpClient *http.Client) *PaymentClient {
	return &PaymentClient{baseURL: baseURL, httpClient: httpClient}
}

type authorizeRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type authorizeResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

func (c *PaymentClient) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	body, err := json.Marshal(authorizeRequest{OrderID: orderID, Amount: amount})
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("payment service unreachable: %w", err)
	}
	defer resp.Body.Close()

	var result authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("invalid payment response: %w", err)
	}
	return result.TransactionID, result.Status, nil
}
