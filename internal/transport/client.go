package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/user/http-cli/internal/models"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(timeoutSeconds int, followRedirects bool, verifySSL bool) *Client {
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	_ = verifySSL
	return &Client{httpClient: client}
}

func (c *Client) Execute(ctx context.Context, req *models.Request, envVars map[string]string) (*models.Response, error) {
	url := interpolate(req.URL, envVars)
	body := interpolate(req.Body.Content, envVars)

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, string(req.Method), url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for _, h := range req.Headers {
		if h.Enabled {
			httpReq.Header.Set(interpolate(h.Key, envVars), interpolate(h.Value, envVars))
		}
	}

	q := httpReq.URL.Query()
	for _, p := range req.QueryParams {
		if p.Enabled {
			q.Set(interpolate(p.Key, envVars), interpolate(p.Value, envVars))
		}
	}
	httpReq.URL.RawQuery = q.Encode()

	applyAuth(httpReq, req.Auth)

	start := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return &models.Response{
			RequestID: req.ID,
			Error:     err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}, nil
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	headers := make(map[string]string)
	for k, v := range httpResp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &models.Response{
		RequestID:  req.ID,
		Status:     httpResp.StatusCode,
		StatusText: httpResp.Status,
		Headers:    headers,
		Body:       string(respBody),
		Size:       int64(len(respBody)),
		Duration:   duration,
		Timestamp:  time.Now(),
	}, nil
}

func applyAuth(req *http.Request, auth models.Auth) {
	switch auth.Type {
	case models.AuthBasic:
		req.SetBasicAuth(auth.Username, auth.Password)
	case models.AuthBearer:
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case models.AuthAPIKey:
		if auth.In == "header" {
			req.Header.Set(auth.Key, auth.Value)
		} else {
			q := req.URL.Query()
			q.Set(auth.Key, auth.Value)
			req.URL.RawQuery = q.Encode()
		}
	}
}

func interpolate(s string, vars map[string]string) string {
	if vars == nil {
		return s
	}
	result := s
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

func BuildCurlCommand(req *models.Request) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("curl -X %s", req.Method))
	for _, h := range req.Headers {
		if h.Enabled {
			buf.WriteString(fmt.Sprintf(" -H '%s: %s'", h.Key, h.Value))
		}
	}
	if req.Body.Content != "" {
		buf.WriteString(fmt.Sprintf(" -d '%s'", req.Body.Content))
	}
	buf.WriteString(fmt.Sprintf(" '%s'", req.URL))
	return buf.String()
}
