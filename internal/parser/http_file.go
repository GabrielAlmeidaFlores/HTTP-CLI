package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/user/http-cli/internal/models"
)

func ParseHTTPFile(path string) ([]*models.Request, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var requests []*models.Request
	var current *models.Request
	var bodyLines []string
	inBody := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "###") {
			if current != nil {
				current.Body.Content = strings.TrimSpace(strings.Join(bodyLines, "\n"))
				requests = append(requests, current)
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, "###"))
			current = &models.Request{
				Name:        name,
				Headers:     make([]models.Header, 0),
				QueryParams: make([]models.QueryParam, 0),
				Body:        models.Body{Type: models.BodyNone},
				Auth:        models.Auth{Type: models.AuthNone},
			}
			bodyLines = nil
			inBody = false
			continue
		}

		if current == nil {
			continue
		}

		if isRequestLine(line) {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				current.Method = models.HTTPMethod(strings.ToUpper(parts[0]))
				current.URL = parts[1]
				if current.Name == "" {
					current.Name = parts[1]
				}
			}
			continue
		}

		if !inBody && strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				current.Headers = append(current.Headers, models.Header{
					Key:     strings.TrimSpace(parts[0]),
					Value:   strings.TrimSpace(parts[1]),
					Enabled: true,
				})
				continue
			}
		}

		if !inBody && line == "" {
			inBody = true
			continue
		}

		if inBody {
			bodyLines = append(bodyLines, line)
		}
	}

	if current != nil {
		content := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		if content != "" {
			current.Body.Content = content
			current.Body.Type = detectBodyType(current.Headers, content)
		}
		requests = append(requests, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning file: %w", err)
	}

	return requests, nil
}

func isRequestLine(line string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "CONNECT", "TRACE"}
	upper := strings.ToUpper(line)
	for _, m := range methods {
		if strings.HasPrefix(upper, m+" ") {
			return true
		}
	}
	return false
}

func detectBodyType(headers []models.Header, content string) models.BodyType {
	for _, h := range headers {
		if strings.EqualFold(h.Key, "content-type") {
			val := strings.ToLower(h.Value)
			switch {
			case strings.Contains(val, "application/json"):
				return models.BodyJSON
			case strings.Contains(val, "application/x-www-form-urlencoded"):
				return models.BodyURLEncoded
			case strings.Contains(val, "multipart/form-data"):
				return models.BodyFormData
			}
		}
	}
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return models.BodyJSON
	}
	return models.BodyRaw
}
