package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"feishu-codex-runner/internal/model"
)

const baseURL = "https://open.feishu.cn/open-apis"

type Client struct {
	appID     string
	appSecret string
	http      *http.Client

	mu          sync.Mutex
	token       string
	tokenExpire time.Time
}

func NewClient(appID, appSecret string) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		http:      &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) FetchMessages(ctx context.Context, startTime time.Time, pageToken string) ([]model.Message, string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, "", err
	}
	u, _ := url.Parse(baseURL + "/im/v1/messages")
	q := u.Query()
	q.Set("page_size", "20")
	q.Set("sort_type", "ByCreateTimeAsc")
	q.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	u.RawQuery = q.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch messages: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch messages status=%d body=%s", res.StatusCode, string(body))
	}

	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				MessageID string `json:"message_id"`
				ChatID    string `json:"chat_id"`
				Sender    struct {
					ID struct {
						OpenID string `json:"open_id"`
					} `json:"sender_id"`
				} `json:"sender"`
				CreateTime string `json:"create_time"`
				Body       struct {
					Content string `json:"content"`
				} `json:"body"`
			} `json:"items"`
			PageToken string `json:"page_token"`
			HasMore   bool   `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", fmt.Errorf("decode fetch messages: %w", err)
	}
	if payload.Code != 0 {
		return nil, "", fmt.Errorf("fetch messages api error code=%d msg=%s", payload.Code, payload.Msg)
	}

	out := make([]model.Message, 0, len(payload.Data.Items))
	for _, item := range payload.Data.Items {
		txt := extractText(item.Body.Content)
		if strings.TrimSpace(txt) == "" {
			continue
		}
		ms, _ := strconv.ParseInt(item.CreateTime, 10, 64)
		if ms > 1e12 {
			ms = ms / 1000
		}
		out = append(out, model.Message{
			MessageID:    item.MessageID,
			ChatID:       item.ChatID,
			SenderOpenID: item.Sender.ID.OpenID,
			Text:         txt,
			CreateTime:   time.Unix(ms, 0),
		})
	}
	next := ""
	if payload.Data.HasMore {
		next = payload.Data.PageToken
	}
	return out, next, nil
}

func (c *Client) SendText(ctx context.Context, chatID, text string) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    fmt.Sprintf(`{"text":%q}`, text),
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/im/v1/messages?receive_id_type=chat_id", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("send message status=%d body=%s", res.StatusCode, string(body))
	}
	return nil
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.tokenExpire.Add(-1*time.Minute)) {
		t := c.token
		c.mu.Unlock()
		return t, nil
	}
	c.mu.Unlock()

	payload := map[string]string{"app_id": c.appID, "app_secret": c.appSecret}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/auth/v3/tenant_access_token/internal", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("get token status=%d body=%s", res.StatusCode, string(body))
	}
	var r struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if r.Code != 0 {
		return "", fmt.Errorf("get token api error code=%d msg=%s", r.Code, r.Msg)
	}
	c.mu.Lock()
	c.token = r.TenantAccessToken
	c.tokenExpire = time.Now().Add(time.Duration(r.Expire) * time.Second)
	t := c.token
	c.mu.Unlock()
	return t, nil
}

func extractText(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		return content
	}
	if text, ok := v["text"].(string); ok {
		return text
	}
	return content
}
