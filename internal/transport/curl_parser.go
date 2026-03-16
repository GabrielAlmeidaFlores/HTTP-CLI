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

	postProcessBody(req)

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

func postProcessBody(req *models.Request) {
	if req.Body.Type != models.BodyRaw || req.Body.Content == "" {
		return
	}

	for i, h := range req.Headers {
		if !strings.EqualFold(h.Key, "content-type") {
			continue
		}
		ct := h.Value
		lower := strings.ToLower(ct)

		if strings.Contains(lower, "multipart/form-data") {
			boundary := extractBoundary(ct)
			if boundary == "" {
				break
			}
			fields := parseMultipartBody(req.Body.Content, boundary)
			if len(fields) > 0 {
				req.Body = models.Body{Type: models.BodyFormData, FormData: fields}
				req.Headers = append(req.Headers[:i], req.Headers[i+1:]...)
			}
			return
		}

		if strings.Contains(lower, "application/json") {
			req.Body.Type = models.BodyJSON
			return
		}

		if strings.Contains(lower, "application/x-www-form-urlencoded") {
			req.Body.Type = models.BodyURLEncoded
			return
		}
	}
}

func extractBoundary(contentType string) string {
	for _, part := range strings.Split(contentType, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "boundary=") {
			return strings.TrimPrefix(strings.TrimPrefix(part[9:], `"`), `"`)
		}
	}
	return ""
}

func parseMultipartBody(rawBody, boundary string) []models.FormField {
	var fields []models.FormField
	delimiter := "--" + boundary

	parts := strings.Split(rawBody, delimiter)
	for _, part := range parts {
		part = strings.TrimPrefix(part, "\r\n")
		if part == "" || strings.HasPrefix(part, "--") {
			continue
		}

		headerEnd := strings.Index(part, "\r\n\r\n")
		if headerEnd < 0 {
			continue
		}
		headerSection := part[:headerEnd]
		bodyContent := strings.TrimSuffix(part[headerEnd+4:], "\r\n")

		var fieldName, fileName string
		for _, line := range strings.Split(headerSection, "\r\n") {
			if !strings.HasPrefix(strings.ToLower(line), "content-disposition:") {
				continue
			}
			disp := line[strings.Index(line, ":")+1:]
			if idx := strings.Index(disp, `name="`); idx >= 0 {
				rest := disp[idx+6:]
				if end := strings.Index(rest, `"`); end >= 0 {
					fieldName = rest[:end]
				}
			}
			if idx := strings.Index(disp, `filename="`); idx >= 0 {
				rest := disp[idx+10:]
				if end := strings.Index(rest, `"`); end >= 0 {
					fileName = rest[:end]
				}
			}
		}

		if fieldName == "" {
			continue
		}

		field := models.FormField{Key: fieldName, Enabled: true}
		if fileName != "" {
			field.Type = models.FormFieldFile
			field.Value = fileName
		} else {
			field.Type = models.FormFieldText
			field.Value = bodyContent
		}
		fields = append(fields, field)
	}
	return fields
}

func tokenize(s string) []string {
	s = strings.ReplaceAll(s, "\\\n", " ")
	s = strings.ReplaceAll(s, "\\\r\n", " ")

	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	runes := []rune(s)
	n := len(runes)

	for i := 0; i < n; i++ {
		ch := runes[i]
		switch {
		case ch == '$' && !inSingle && !inDouble && i+1 < n && runes[i+1] == '\'':
			i += 2
			for i < n {
				c := runes[i]
				if c == '\'' {
					break
				}
				if c == '\\' && i+1 < n {
					i++
					switch runes[i] {
					case 'n':
						current.WriteRune('\n')
					case 'r':
						current.WriteRune('\r')
					case 't':
						current.WriteRune('\t')
					case '\\':
						current.WriteRune('\\')
					case '\'':
						current.WriteRune('\'')
					case '"':
						current.WriteRune('"')
					default:
						current.WriteRune('\\')
						current.WriteRune(runes[i])
					}
				} else {
					current.WriteRune(c)
				}
				i++
			}
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
