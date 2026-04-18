package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Discord struct {
	webhookURL string
	client     *http.Client
}

func NewDiscord(webhookURL string) *Discord {
	return &Discord{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

func (d *Discord) Name() string { return "discord" }

func (d *Discord) Send(ctx context.Context, message string) error {
	payload, err := json.Marshal(map[string]string{"content": message})
	if err != nil {
		return fmt.Errorf("discord marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord send: status %d", resp.StatusCode)
	}
	return nil
}
