package models

import "time"

type HTTPMethod string

const (
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodPATCH   HTTPMethod = "PATCH"
	MethodOPTIONS HTTPMethod = "OPTIONS"
	MethodHEAD    HTTPMethod = "HEAD"
)

var AllMethods = []HTTPMethod{
	MethodGET, MethodPOST, MethodPUT, MethodDELETE,
	MethodPATCH, MethodOPTIONS, MethodHEAD,
}

type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBasic  AuthType = "basic"
	AuthBearer AuthType = "bearer"
	AuthAPIKey AuthType = "apikey"
)

type Auth struct {
	Type     AuthType `json:"type"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Token    string   `json:"token,omitempty"`
	Key      string   `json:"key,omitempty"`
	Value    string   `json:"value,omitempty"`
	In       string   `json:"in,omitempty"`
}

type Header struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type QueryParam struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type BodyType string

const (
	BodyNone       BodyType = "none"
	BodyRaw        BodyType = "raw"
	BodyJSON       BodyType = "json"
	BodyFormData   BodyType = "form-data"
	BodyURLEncoded BodyType = "urlencoded"
)

type Body struct {
	Type    BodyType `json:"type"`
	Content string   `json:"content"`
}

type Request struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Method       HTTPMethod   `json:"method"`
	URL          string       `json:"url"`
	Headers      []Header     `json:"headers"`
	QueryParams  []QueryParam `json:"query_params"`
	Body         Body         `json:"body"`
	Auth         Auth         `json:"auth"`
	CollectionID string       `json:"collection_id,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
