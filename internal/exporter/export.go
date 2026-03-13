package exporter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/user/http-cli/internal/models"
)

func ToCurl(req *models.Request) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("curl -X %s", req.Method))

	for _, h := range req.Headers {
		if h.Enabled {
			sb.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", h.Key, h.Value))
		}
	}

	switch req.Auth.Type {
	case models.AuthBasic:
		sb.WriteString(fmt.Sprintf(" \\\n  -u '%s:%s'", req.Auth.Username, req.Auth.Password))
	case models.AuthBearer:
		sb.WriteString(fmt.Sprintf(" \\\n  -H 'Authorization: Bearer %s'", req.Auth.Token))
	case models.AuthAPIKey:
		if req.Auth.In == "header" {
			sb.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", req.Auth.Key, req.Auth.Value))
		}
	}

	if req.Body.Content != "" {
		escaped := strings.ReplaceAll(req.Body.Content, "'", "'\\''")
		sb.WriteString(fmt.Sprintf(" \\\n  -d '%s'", escaped))
	}

	url := req.URL
	if len(req.QueryParams) > 0 {
		var params []string
		for _, p := range req.QueryParams {
			if p.Enabled {
				params = append(params, p.Key+"="+p.Value)
			}
		}
		if len(params) > 0 {
			if strings.Contains(url, "?") {
				url += "&" + strings.Join(params, "&")
			} else {
				url += "?" + strings.Join(params, "&")
			}
		}
	}

	sb.WriteString(fmt.Sprintf(" \\\n  '%s'", url))

	return sb.String()
}

type postmanExportCollection struct {
	Info postmanExportInfo   `json:"info"`
	Item []postmanExportItem `json:"item"`
}

type postmanExportInfo struct {
	Name       string `json:"name"`
	Schema     string `json:"schema"`
	ExportedAt string `json:"_postman_id,omitempty"`
}

type postmanExportItem struct {
	Name    string           `json:"name"`
	Request postmanExportReq `json:"request"`
}

type postmanExportReq struct {
	Method string                `json:"method"`
	Header []postmanExportHeader `json:"header"`
	URL    postmanExportURL      `json:"url"`
	Body   *postmanExportBody    `json:"body,omitempty"`
}

type postmanExportHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanExportURL struct {
	Raw string `json:"raw"`
}

type postmanExportBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw,omitempty"`
}

func ToPostmanCollection(name string, requests []*models.Request) ([]byte, error) {
	col := postmanExportCollection{
		Info: postmanExportInfo{
			Name:       name,
			Schema:     "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
			ExportedAt: fmt.Sprintf("%d", time.Now().UnixNano()),
		},
		Item: make([]postmanExportItem, 0, len(requests)),
	}

	for _, req := range requests {
		item := postmanExportItem{
			Name: req.Name,
			Request: postmanExportReq{
				Method: string(req.Method),
				Header: make([]postmanExportHeader, 0),
				URL:    postmanExportURL{Raw: req.URL},
			},
		}

		for _, h := range req.Headers {
			if h.Enabled {
				item.Request.Header = append(item.Request.Header, postmanExportHeader{
					Key:   h.Key,
					Value: h.Value,
				})
			}
		}

		if req.Body.Content != "" {
			item.Request.Body = &postmanExportBody{
				Mode: "raw",
				Raw:  req.Body.Content,
			}
		}

		col.Item = append(col.Item, item)
	}

	return json.MarshalIndent(col, "", "  ")
}
