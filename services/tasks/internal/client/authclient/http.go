package authclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type verifyResponse struct {
	Valid   bool   `json:"valid"`
	Subject string `json:"subject"`
	Error   string `json:"error"`
}

// Verify Р С—РЎР‚Р С•Р Р†Р ВµРЎР‚РЎРЏР ВµРЎвЂљ Р В°РЎС“РЎвЂљР ВµР Р…РЎвЂљР С‘РЎвЂћР С‘Р С”Р В°РЎвЂ Р С‘РЎР‹ РЎвЂЎР ВµРЎР‚Р ВµР В· Auth service.
// Р СџР С•Р Т‘Р Т‘Р ВµРЎР‚Р В¶Р С‘Р Р†Р В°Р ВµРЎвЂљ Р Т‘Р Р†Р В° РЎР‚Р ВµР В¶Р С‘Р СР В°: Р В»Р С‘Р В±Р С• Bearer-РЎвЂљР С•Р С”Р ВµР Р… (Р С”Р В°Р С” Р Р† Р СџР вЂ” 5), Р В»Р С‘Р В±Р С• session cookie (Р СџР вЂ” 6).
// Р вЂўРЎРѓР В»Р С‘ Р С—Р ВµРЎР‚Р ВµР Т‘Р В°Р Р… token РІР‚вЂќ РЎв‚¬Р В»РЎвЂР С Authorization. Р вЂўРЎРѓР В»Р С‘ sessionCookie РІР‚вЂќ Р С—РЎР‚Р С•Р В±РЎР‚Р В°РЎРѓРЎвЂ№Р Р†Р В°Р ВµР С cookie.
func (c *Client) Verify(ctx context.Context, token, sessionCookie, requestID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/auth/verify", nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionCookie})
	}
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth returned %d", resp.StatusCode)
	}

	var body verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return body.Subject, nil
}

// ErrUnauthorized РІР‚вЂќ РЎвЂљР С•Р С”Р ВµР Р… Р Р…Р ВµР Р†Р В°Р В»Р С‘Р Т‘Р ВµР Р….
var ErrUnauthorized = fmt.Errorf("unauthorized")
