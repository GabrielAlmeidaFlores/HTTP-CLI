package transport

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/user/http-cli/internal/models"
)

func ParseCurlCommand(curl string) (*models.Request, error) {
	curl = strings.TrimSpace(curl)
	if !strings.HasPrefix(curl, "curl") {
		return nil, fmt.Errorf("not a curl command")
	}

	req := &models.Request{
		Method:  models.MethodGET,
		Headers: make([]models.Header, 0),
		Body:    models.Body{Type: models.BodyNone},
		Auth:    models.Auth{Type: models.AuthNone},
	}

	tokens := tokenize(curl[4:])

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch {
		case token == "-X" || token == "--request":
			if i+1 < len(tokens) {
				i++
				req.Method = models.HTTPMethod(strings.ToUpper(tokens[i]))
			}
		case token == "-H" || token == "--header":
			if i+1 < len(tokens) {
				i++
				parts := strings.SplitN(tokens[i], ":", 2)
				if len(parts) == 2 {
					req.Headers = append(req.Headers, models.Header{
						Key:     strings.TrimSpace(parts[0]),
						Value:   strings.TrimSpace(parts[1]),
						Enabled: true,
					})
				}
			}
		case token == "-b" || token == "--cookie":
			if i+1 < len(tokens) {
				i++
				req.Headers = append(req.Headers, models.Header{
					Key:     "Cookie",
					Value:   tokens[i],
					Enabled: true,
				})
			}
		case token == "-d" || token == "--data" || token == "--data-raw" || token == "--data-binary":
			if i+1 < len(tokens) {
				i++
				req.Body = models.Body{Type: models.BodyRaw, Content: tokens[i]}
				if req.Method == models.MethodGET {
					req.Method = models.MethodPOST
				}
			}
		case token == "--data-urlencode":
			if i+1 < len(tokens) {
				i++
				req.Body = models.Body{Type: models.BodyURLEncoded, Content: tokens[i]}
				if req.Method == models.MethodGET {
					req.Method = models.MethodPOST
				}
			}
		case token == "-u" || token == "--user":
			if i+1 < len(tokens) {
				i++
				parts := strings.SplitN(tokens[i], ":", 2)
				if len(parts) == 2 {
					req.Auth = models.Auth{
						Type:     models.AuthBasic,
						Username: parts[0],
						Password: parts[1],
					}
				}
			}
		case token == "-A" || token == "--user-agent":
			if i+1 < len(tokens) {
				i++
				req.Headers = append(req.Headers, models.Header{
					Key:     "User-Agent",
					Value:   tokens[i],
					Enabled: true,
				})
			}
		case token == "--compressed" || token == "-s" || token == "--silent" ||
			token == "-L" || token == "--location" || token == "-k" || token == "--insecure" ||
			token == "-v" || token == "--verbose" || token == "-i" || token == "--include":
		case !strings.HasPrefix(token, "-"):
			cleaned := strings.Trim(token, "'\"")
			if _, err := url.ParseRequestURI(cleaned); err == nil {
				req.URL = cleaned
			}
		}
	}

	if req.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	if req.Name == "" {
		if u, err := url.Parse(req.URL); err == nil {
			req.Name = u.Path
			if req.Name == "" || req.Name == "/" {
				req.Name = u.Host
			}
		}
	}

	return req, nil
}

func tokenize(s string) []string {
	s = strings.ReplaceAll(s, "\\\n", " ")
	s = strings.ReplaceAll(s, "\\\r\n", " ")

	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for _, ch := range s {
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
