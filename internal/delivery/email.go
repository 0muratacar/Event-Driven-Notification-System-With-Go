package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/insiderone/notifier/internal/domain"
)

type EmailProvider struct {
	baseURL string
	path    string
	client  *http.Client
}

func NewEmailProvider(baseURL, path string, timeout time.Duration) *EmailProvider {
	return &EmailProvider{
		baseURL: baseURL,
		path:    path,
		client:  &http.Client{Timeout: timeout},
	}
}

func (p *EmailProvider) Channel() domain.Channel {
	return domain.ChannelEmail
}

func (p *EmailProvider) Send(ctx context.Context, n *domain.Notification) Result {
	start := time.Now()

	payload := map[string]string{
		"to":      n.Recipient,
		"subject": n.Subject,
		"body":    n.Body,
		"channel": string(n.Channel),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.path, bytes.NewReader(body))
	if err != nil {
		return Result{Error: fmt.Errorf("creating request: %w", err), DurationMs: time.Since(start).Milliseconds()}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return Result{Error: fmt.Errorf("sending email: %w", err), DurationMs: time.Since(start).Milliseconds()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	result := Result{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(respBody),
		DurationMs:   time.Since(start).Milliseconds(),
	}

	if resp.StatusCode >= 400 {
		result.Error = fmt.Errorf("%w: status %d", domain.ErrProviderFailure, resp.StatusCode)
	}
	return result
}
