package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/golog"
)

const (
	pathWhoami            = "/plugin_repo/v1/whoami"
	pathFilesUpload       = "/plugin_repo/v1/files/upload"
	pathReleasesPublish   = "/plugin_repo/v1/releases/publish"
	pathSubmissionsList   = "/plugin_repo/v1/submissions/list"
	pathSubmissionsCancel = "/plugin_repo/v1/submissions/cancel"
	pathPluginsList       = "/plugin_repo/v1/plugins/list"
	pathPluginsSetActive  = "/plugin_repo/v1/plugins/set-active"
	pathPluginsDelete     = "/plugin_repo/v1/plugins/delete"
	pathReleasesDelete    = "/plugin_repo/v1/releases/delete"
	pathReleasesPrune     = "/plugin_repo/v1/releases/prune"
	pathKeysUpdateEmail   = "/plugin_repo/v1/keys/update-email"
)

// Client is an HMAC-authenticated client for the plugin_repo write endpoints.
// It signs every request the same way the Terminal externalapi middleware
// verifies: HMAC-SHA256 over timestamp + method + path + rawBody.
type Client struct {
	BaseURL   string
	APIKey    string
	APISecret string
	HTTP      *http.Client
	UserAgent string
}

func New(baseURL, apiKey, apiSecret string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		APISecret: apiSecret,
		HTTP:      &http.Client{Timeout: 2 * time.Minute},
		UserAgent: "plusev-cli",
	}
}

// Whoami calls the /whoami endpoint to validate credentials and return
// the repo metadata associated with the dev key. Used by `registry add`
// to derive a unique label from the repo slug.
func (c *Client) Whoami(ctx context.Context) (*WhoamiResult, error) {
	var out WhoamiResult

	if err := c.Post(ctx, pathWhoami, nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Post sends a JSON POST and decodes the wrapped response into result.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}

		bodyBytes = b
	}

	return c.do(ctx, http.MethodPost, path, bodyBytes, "application/json", result)
}

// Upload sends a multipart file upload and decodes the wrapped response.
func (c *Client) Upload(ctx context.Context, path, fieldName, filename string, content []byte, result any) error {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}

	if _, err := fw.Write(content); err != nil {
		return fmt.Errorf("write form file: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	contentType := w.FormDataContentType()

	return c.do(ctx, http.MethodPost, path, buf.Bytes(), contentType, result)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, contentType string, result any) error {
	url := c.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	timestamp := time.Now().Unix()

	signPath := req.URL.Path
	if req.URL.RawQuery != "" {
		signPath += "?" + req.URL.RawQuery
	}

	sig := c.sign(method, signPath, timestamp, body)

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Signature", sig)
	req.Header.Set("User-Agent", c.UserAgent)

	golog.Debugf("→ %s %s (signature=%s…)", method, url, sig[:12])
	if len(body) > 0 && len(body) < 4096 {
		golog.Debugf("  body: %s", string(body))
	} else if len(body) >= 4096 {
		golog.Debugf("  body: <%d bytes — too large to log>", len(body))
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	golog.Debugf("← %d %s (body=%d bytes)", resp.StatusCode, url, len(raw))
	if len(raw) > 0 && len(raw) < 4096 {
		golog.Debugf("  body: %s", string(raw))
	} else if len(raw) >= 4096 {
		golog.Debugf("  body: <%d bytes — too large to log>", len(raw))
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return &AuthError{StatusCode: resp.StatusCode, Body: string(raw)}
	case resp.StatusCode == http.StatusTooManyRequests:
		return &RateLimitError{Body: string(raw)}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(raw))
	}

	var wrapped struct {
		Result bool            `json:"result"`
		Data   json.RawMessage `json:"data"`
		Code   int             `json:"code"`
		Error  string          `json:"error"`
	}

	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if !wrapped.Result {
		msg := wrapped.Error
		if msg == "" {
			if s, ok := parseString(wrapped.Data); ok {
				msg = s
			}
		}

		if msg == "" {
			msg = fmt.Sprintf("request failed (code %d)", wrapped.Code)
		}

		return &APIError{Code: wrapped.Code, Message: msg}
	}

	if result != nil && len(wrapped.Data) > 0 {
		if err := json.Unmarshal(wrapped.Data, result); err != nil {
			return fmt.Errorf("parse response data: %w", err)
		}
	}

	return nil
}

func (c *Client) sign(method, path string, timestamp int64, body []byte) string {
	message := strconv.FormatInt(timestamp, 10) + method + path + string(body)

	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(message))

	return hex.EncodeToString(h.Sum(nil))
}

// UploadFile uploads a wasm binary and returns its sha256.
func (c *Client) UploadFile(ctx context.Context, filename string, content []byte) (*UploadResult, error) {
	var out UploadResult

	if err := c.Upload(ctx, pathFilesUpload, "file", filename, content, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *Client) PublishRelease(ctx context.Context, r PublishRelease) error {
	return c.Post(ctx, pathReleasesPublish, r, nil)
}

func (c *Client) ListSubmissions(ctx context.Context, req SubmissionListReq) ([]Submission, error) {
	var out []Submission

	if err := c.Post(ctx, pathSubmissionsList, req, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) CancelSubmission(ctx context.Context, id uint64) error {
	return c.Post(ctx, pathSubmissionsCancel, SubmissionCancelReq{SubmissionID: id}, nil)
}

func (c *Client) ListPlugins(ctx context.Context) ([]PluginListEntry, error) {
	var out []PluginListEntry

	if err := c.Post(ctx, pathPluginsList, nil, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) SetPluginActive(ctx context.Context, pluginID string, active bool) error {
	return c.Post(ctx, pathPluginsSetActive, SetActiveReq{PluginID: pluginID, Active: active}, nil)
}

func (c *Client) DeletePlugin(ctx context.Context, pluginID string) error {
	return c.Post(ctx, pathPluginsDelete, DeletePluginReq{PluginID: pluginID}, nil)
}

func (c *Client) DeleteRelease(ctx context.Context, pluginID, version string) error {
	return c.Post(ctx, pathReleasesDelete, DeleteReleaseReq{PluginID: pluginID, Version: version}, nil)
}

func (c *Client) PruneReleases(ctx context.Context, pluginID, olderThan string) error {
	return c.Post(ctx, pathReleasesPrune, PruneReleasesReq{PluginID: pluginID, OlderThan: olderThan}, nil)
}

func (c *Client) UpdateEmail(ctx context.Context, email string) error {
	return c.Post(ctx, pathKeysUpdateEmail, UpdateEmailReq{Email: email}, nil)
}

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

type AuthError struct {
	StatusCode int
	Body       string
}

func (e *AuthError) Error() string {
	msg := e.Body
	if msg == "" {
		msg = "authentication failed"
	}

	return fmt.Sprintf("auth error (status %d): %s", e.StatusCode, truncate([]byte(msg)))
}

type RateLimitError struct {
	Body string
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded: " + truncate([]byte(e.Body))
}

func parseString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}

	var s string

	if err := json.Unmarshal(raw, &s); err == nil {
		return s, true
	}

	return "", false
}

func truncate(b []byte) string {
	const max = 500

	if len(b) > max {
		return string(b[:max]) + "..."
	}

	return string(b)
}
