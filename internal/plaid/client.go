package plaid

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "https://sandbox.plaid.com"

type Client struct {
	ClientID string
	Secret   string
	HTTP     *http.Client
}

type Transaction struct {
	TransactionID string   `json:"transaction_id"`
	AccountID     string   `json:"account_id"`
	Amount        float64  `json:"amount"`
	Date          string   `json:"date"`
	Name          string   `json:"name"`
	MerchantName  string   `json:"merchant_name"`
	Category      []string `json:"category"`
	Pending       bool     `json:"pending"`
}

func NewClient() (*Client, error) {
	clientID := os.Getenv("PLAID_CLIENT_ID")
	secret := os.Getenv("PLAID_SECRET")

	if clientID == "" || secret == "" {
		return nil, errors.New("il manque PLAID_CLIENT_ID ou PLAID_SECRET dans l'environnement")
	}

	return &Client{
		ClientID: clientID,
		Secret:   secret,
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) postJSON(endpoint string, payload any, result any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// On lit toute la réponse en octets pour pouvoir soit lire l'erreur Plaid, soit décoder le résultat
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var plaidErr struct {
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		}
		_ = json.Unmarshal(bodyBytes, &plaidErr)

		if plaidErr.ErrorCode != "" {
			return fmt.Errorf("erreur Plaid (HTTP %d - %s) : %s", resp.StatusCode, plaidErr.ErrorCode, plaidErr.ErrorMessage)
		}
		return fmt.Errorf("erreur Plaid (code HTTP %d)", resp.StatusCode)
	}

	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) CreatePublicToken(institutionID string, products []string) (string, error) {
	body := map[string]any{
		"client_id":        c.ClientID,
		"secret":          c.Secret,
		"institution_id":   institutionID,
		"initial_products": products,
	}

	var resp struct {
		PublicToken string `json:"public_token"`
	}

	if err := c.postJSON("/sandbox/public_token/create", body, &resp); err != nil {
		return "", err
	}
	return resp.PublicToken, nil
}

func (c *Client) ExchangePublicToken(publicToken string) (string, error) {
	body := map[string]any{
		"client_id":    c.ClientID,
		"secret":      c.Secret,
		"public_token": publicToken,
	}

	var resp struct {
		AccessToken string `json:"access_token"`
	}

	if err := c.postJSON("/item/public_token/exchange", body, &resp); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

func (c *Client) GetTransactions(accessToken, startDate, endDate string) ([]Transaction, error) {
	body := map[string]any{
		"client_id":    c.ClientID,
		"secret":      c.Secret,
		"access_token": accessToken,
		"start_date":   startDate,
		"end_date":     endDate,
	}

	var resp struct {
		Transactions []Transaction `json:"transactions"`
	}

	if err := c.postJSON("/transactions/get", body, &resp); err != nil {
		return nil, err
	}
	return resp.Transactions, nil
}