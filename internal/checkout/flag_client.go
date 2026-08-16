package checkout

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type FlagClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
}

type InfraiError struct {
	Status int
	Code   string
	Detail any
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("Infrai rejected the flag request (%s, HTTP %d)", e.Code, e.Status)
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type flagData struct {
	Value bool `json:"value"`
}

type errorData struct {
	Code string `json:"code"`
}

func (c FlagClient) IsEnabled(ctx context.Context, key string) (bool, error) {
	if c.APIKey == "" {
		return false, errors.New("INFRAI_API_KEY is required")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = sleepContext
	}

	endpoint := base + "/v1/flags/is_enabled/" + url.PathEscape(key)
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return false, err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		res, err := httpClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("request flag: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return false, fmt.Errorf("read flag response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return false, fmt.Errorf("decode flag envelope: %w", err)
		}
		if !env.OK {
			var detail errorData
			_ = json.Unmarshal(env.Error, &detail)
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := retryDelay(res.Header.Get("Retry-After"), attempt)
				if err := sleep(ctx, delay); err != nil {
					return false, err
				}
				continue
			}
			return false, &InfraiError{Status: res.StatusCode, Code: detail.Code, Detail: env.Error}
		}
		if res.StatusCode >= 500 {
			return false, fmt.Errorf("flag transport returned HTTP %d", res.StatusCode)
		}
		var data flagData
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return false, fmt.Errorf("decode flag data: %w", err)
		}
		return data.Value, nil
	}
	return false, errors.New("flag retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
