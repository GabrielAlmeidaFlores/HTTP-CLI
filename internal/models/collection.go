package models

import "time"

type Folder struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	RequestIDs []string `json:"request_ids"`
	Folders    []Folder `json:"folders,omitempty"`
}

type Collection struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Variables        map[string]string `json:"variables,omitempty"`
	RequestIDs       []string          `json:"request_ids"`
	Folders          []Folder          `json:"folders,omitempty"`
	PreRequestScript string            `json:"pre_request_script,omitempty"`
	TestsScript      string            `json:"tests_script,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type Environment struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
	IsActive  bool              `json:"is_active"`
	CreatedAt time.Time         `json:"created_at"`
}
