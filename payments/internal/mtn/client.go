package mtn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	baseURL         string
	subscriptionKey string
	apiUser         string
	apiKey          string
	environment     string
	payeeID         string
	http            *http.Client
}

func NewClient(baseURL, subscriptionKey, apiUser, apiKey, environment, payeeID string) *Client {
	return &Client{
		baseURL:         baseURL,
		subscriptionKey: subscriptionKey,
		apiUser:         apiUser,
		apiKey:          apiKey,
		environment:     environment,
		payeeID:         payeeID,
		http:            &http.Client{Timeout: 30 * time.Second},
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/collection/token/", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.apiUser, c.apiKey)
	req.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mtn token: status %d: %s", resp.StatusCode, body)
	}

	var t tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", fmt.Errorf("mtn token decode: %w", err)
	}
	return t.AccessToken, nil
}

type Party struct {
	PartyIDType string `json:"partyIdType"`
	PartyID     string `json:"partyId"`
}

type CreateInvoiceRequest struct {
	ExternalID       string `json:"externalId"`
	Amount           string `json:"amount"`
	Currency         string `json:"currency"`
	ValidityDuration string `json:"validityDuration"`
	IntendedPayer    Party  `json:"intendedPayer"`
	Payee            Party  `json:"payee"`
	Description      string `json:"description,omitempty"`
}

// CreateInvoice sends a create-invoice request to MTN and returns the reference ID.
func (c *Client) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (referenceID string, err error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("mtn auth: %w", err)
	}

	referenceID = uuid.NewString()

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/collection/v2_0/invoice", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Reference-Id", referenceID)
	httpReq.Header.Set("X-Target-Environment", c.environment)
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", c.subscriptionKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("mtn invoice: %w", err)
	}
	defer resp.Body.Close()

	// MTN returns 202 Accepted on success
	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mtn invoice: status %d: %s", resp.StatusCode, respBody)
	}

	return referenceID, nil
}
