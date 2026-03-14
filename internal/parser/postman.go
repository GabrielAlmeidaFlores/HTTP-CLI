package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/user/http-cli/internal/models"
)

var parseIDCounter int64

func nextParseID() string {
	n := atomic.AddInt64(&parseIDCounter, 1)
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), n)
}

type postmanCollection struct {
	Info     postmanInfo       `json:"info"`
	Item     []postmanItem     `json:"item"`
	Variable []postmanVariable `json:"variable"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type postmanVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type postmanItem struct {
	Name    string        `json:"name"`
	Item    []postmanItem `json:"item"`
	Request *postmanReq   `json:"request"`
}

type postmanReq struct {
	Method string          `json:"method"`
	URL    postmanURL      `json:"url"`
	Header []postmanHeader `json:"header"`
	Body   *postmanBody    `json:"body"`
	Auth   *postmanAuth    `json:"auth"`
}

type postmanURL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host"`
	Path  []string `json:"path"`
	Query []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"query"`
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type postmanBody struct {
	Mode       string `json:"mode"`
	Raw        string `json:"raw"`
	URLEncoded []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"urlencoded"`
	FormData []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"formdata"`
}

type postmanAuth struct {
	Type   string            `json:"type"`
	Basic  []postmanAuthItem `json:"basic"`
	Bearer []postmanAuthItem `json:"bearer"`
	Apikey []postmanAuthItem `json:"apikey"`
}

type postmanAuthItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func ParsePostmanCollection(path string) ([]*models.Request, *models.Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading file: %w", err)
	}

	var col postmanCollection
	if err := json.Unmarshal(data, &col); err != nil {
		return nil, nil, fmt.Errorf("parsing postman collection: %w", err)
	}

	var allRequests []*models.Request
	var topLevelIDs []string
	var folders []models.Folder

	for _, item := range col.Item {
		if len(item.Item) > 0 {
			folder, subReqs := buildFolder(item)
			folders = append(folders, folder)
			allRequests = append(allRequests, subReqs...)
		} else if item.Request != nil {
			req := convertPostmanRequest(item.Name, item.Request)
			allRequests = append(allRequests, req)
			topLevelIDs = append(topLevelIDs, req.ID)
		}
	}

	collection := &models.Collection{
		Name:       col.Info.Name,
		Variables:  make(map[string]string),
		RequestIDs: topLevelIDs,
		Folders:    folders,
	}

	for _, v := range col.Variable {
		if v.Key != "" {
			collection.Variables[v.Key] = v.Value
		}
	}

	return allRequests, collection, nil
}

func buildFolder(item postmanItem) (models.Folder, []*models.Request) {
	folder := models.Folder{
		ID:         nextParseID(),
		Name:       item.Name,
		RequestIDs: make([]string, 0),
		Folders:    make([]models.Folder, 0),
	}

	var allReqs []*models.Request
	for _, child := range item.Item {
		if len(child.Item) > 0 {
			subFolder, subReqs := buildFolder(child)
			folder.Folders = append(folder.Folders, subFolder)
			allReqs = append(allReqs, subReqs...)
		} else if child.Request != nil {
			req := convertPostmanRequest(child.Name, child.Request)
			allReqs = append(allReqs, req)
			folder.RequestIDs = append(folder.RequestIDs, req.ID)
		}
	}
	return folder, allReqs
}

func convertPostmanRequest(name string, pr *postmanReq) *models.Request {
	req := &models.Request{
		ID:          nextParseID(),
		Name:        name,
		Method:      models.HTTPMethod(strings.ToUpper(pr.Method)),
		URL:         pr.URL.Raw,
		Headers:     make([]models.Header, 0),
		QueryParams: make([]models.QueryParam, 0),
		Body:        models.Body{Type: models.BodyNone},
		Auth:        models.Auth{Type: models.AuthNone},
	}

	for _, h := range pr.Header {
		req.Headers = append(req.Headers, models.Header{
			Key:     h.Key,
			Value:   h.Value,
			Enabled: !h.Disabled,
		})
	}

	for _, q := range pr.URL.Query {
		req.QueryParams = append(req.QueryParams, models.QueryParam{
			Key:     q.Key,
			Value:   q.Value,
			Enabled: true,
		})
	}

	if pr.Body != nil {
		switch pr.Body.Mode {
		case "raw":
			bodyType := models.BodyRaw
			for _, h := range req.Headers {
				if strings.EqualFold(h.Key, "content-type") && strings.Contains(strings.ToLower(h.Value), "json") {
					bodyType = models.BodyJSON
				}
			}
			req.Body = models.Body{Type: bodyType, Content: pr.Body.Raw}
		case "urlencoded":
			var parts []string
			for _, p := range pr.Body.URLEncoded {
				parts = append(parts, p.Key+"="+p.Value)
			}
			req.Body = models.Body{Type: models.BodyURLEncoded, Content: strings.Join(parts, "&")}
		case "formdata":
			var parts []string
			for _, p := range pr.Body.FormData {
				parts = append(parts, p.Key+"="+p.Value)
			}
			req.Body = models.Body{Type: models.BodyFormData, Content: strings.Join(parts, "&")}
		}
	}

	if pr.Auth != nil {
		req.Auth = convertPostmanAuth(pr.Auth)
	}

	return req
}

func convertPostmanAuth(a *postmanAuth) models.Auth {
	findValue := func(items []postmanAuthItem, key string) string {
		for _, item := range items {
			if item.Key == key {
				return item.Value
			}
		}
		return ""
	}

	switch strings.ToLower(a.Type) {
	case "basic":
		return models.Auth{
			Type:     models.AuthBasic,
			Username: findValue(a.Basic, "username"),
			Password: findValue(a.Basic, "password"),
		}
	case "bearer":
		return models.Auth{
			Type:  models.AuthBearer,
			Token: findValue(a.Bearer, "token"),
		}
	case "apikey":
		return models.Auth{
			Type:  models.AuthAPIKey,
			Key:   findValue(a.Apikey, "key"),
			Value: findValue(a.Apikey, "value"),
			In:    findValue(a.Apikey, "in"),
		}
	}

	return models.Auth{Type: models.AuthNone}
}
