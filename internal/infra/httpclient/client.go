package httpclient

import (
	"context"
	"io"
	"net/http"
	"time"
)

var Client = &http.Client{Timeout: 30 * time.Second}

func Post(ctx context.Context, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return Client.Do(req)
}

func PostWithAuth(ctx context.Context, url, contentType, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	return Client.Do(req)
}
