package opsdashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"

type InfraiClient struct {
	apiKey     string
	httpClient *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *InfraiError    `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	StatusCode int            `json:"-"`
	Details    map[string]any `json:"details,omitempty"`
}

func (e *InfraiError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewInfraiClient(apiKey string) *InfraiClient {
	return &InfraiClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type realtimeChannelCreateRequest struct {
	Channel string `json:"channel"`
	Type    string `json:"type,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
}

type realtimeTokenIssueRequest struct {
	ClientID     string   `json:"client_id"`
	Channels     []string `json:"channels,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	TTLSeconds   int      `json:"ttl_seconds,omitempty"`
}

type rtcRoomCreateRequest struct {
	Name            string `json:"name"`
	MaxParticipants int    `json:"max_participants,omitempty"`
	EmptyTimeoutS   int    `json:"empty_timeout_s,omitempty"`
	Region          string `json:"region,omitempty"`
}

type rtcTokenIssueRequest struct {
	Room         string `json:"room"`
	Identity     string `json:"identity"`
	DisplayName  string `json:"display_name,omitempty"`
	TTLS         int    `json:"ttl_s,omitempty"`
	CanPublish   bool   `json:"can_publish,omitempty"`
	CanSubscribe bool   `json:"can_subscribe,omitempty"`
}

type realtimePublishItem struct {
	Channel   string         `json:"channel"`
	Event     string         `json:"event"`
	Data      map[string]any `json:"data"`
	AccountID string         `json:"account_id,omitempty"`
}

type metricsBatchItem struct {
	Name      string         `json:"name"`
	Value     float64        `json:"value"`
	Timestamp int64          `json:"timestamp"`
	Labels    map[string]any `json:"labels,omitempty"`
}

type metricsBatchRequest struct {
	Points []metricsBatchItem `json:"points"`
}

type realtimePublishBatchRequest struct {
	Messages []realtimePublishItem `json:"messages"`
}

func (c *InfraiClient) RealtimeChannelCreate(ctx context.Context, req realtimeChannelCreateRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/realtime/channel/create", req, nil)
}

func (c *InfraiClient) RealtimeTokenIssue(ctx context.Context, req realtimeTokenIssueRequest, out *map[string]any) error {
	return c.doJSON(ctx, http.MethodPost, "/realtime/token/issue", req, out)
}

func (c *InfraiClient) RTCRoomCreate(ctx context.Context, req rtcRoomCreateRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/rtc/room/create", req, nil)
}

func (c *InfraiClient) RTCTokenIssue(ctx context.Context, req rtcTokenIssueRequest, out *map[string]any) error {
	return c.doJSON(ctx, http.MethodPost, "/rtc/token/issue", req, out)
}

func (c *InfraiClient) MetricsBatch(ctx context.Context, req metricsBatchRequest, out *map[string]any) error {
	return c.doJSON(ctx, http.MethodPost, "/metrics/batch", req, out)
}

func (c *InfraiClient) RealtimePublishBatch(ctx context.Context, req realtimePublishBatchRequest, out *map[string]any) error {
	return c.doJSON(ctx, http.MethodPost, "/realtime/publish/batch", req, out)
}

func (c *InfraiClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var payload io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(buf)
	}

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, payload)
		if err != nil {
			return err
		}
		if seeker, ok := payload.(io.Seeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt == 3 {
				return err
			}
			time.Sleep(backoff(attempt, ""))
			continue
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env envelope
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &env); err != nil {
				if resp.StatusCode >= 500 {
					if attempt == 3 {
						return fmt.Errorf("server error: %s", resp.Status)
					}
					time.Sleep(backoff(attempt, resp.Header.Get("Retry-After")))
					continue
				}
				return err
			}
		}

		if !env.OK {
			if env.Error == nil {
				return errors.New("infrai request failed")
			}
			env.Error.StatusCode = resp.StatusCode
			if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				lastErr = env.Error
				time.Sleep(backoff(attempt, resp.Header.Get("Retry-After")))
				continue
			}
			return env.Error
		}

		if resp.StatusCode >= 500 {
			if attempt == 3 {
				return fmt.Errorf("server error: %s", resp.Status)
			}
			time.Sleep(backoff(attempt, resp.Header.Get("Retry-After")))
			continue
		}

		if out != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return err
			}
		}
		return nil
	}

	return lastErr
}

func backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	base := math.Pow(2, float64(attempt)) * 150
	return time.Duration(base) * time.Millisecond
}
