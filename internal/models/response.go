package models

import "time"

type Response struct {
	RequestID  string            `json:"request_id"`
	Status     int               `json:"status"`
	StatusText string            `json:"status_text"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Size       int64             `json:"size"`
	Duration   time.Duration     `json:"duration"`
	Timestamp  time.Time         `json:"timestamp"`
	Error      string            `json:"error,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	Protocol   string            `json:"protocol,omitempty"`
}

func (r *Response) IsSuccess() bool {
	return r.Status >= 200 && r.Status < 300
}

func (r *Response) IsClientError() bool {
	return r.Status >= 400 && r.Status < 500
}

func (r *Response) IsServerError() bool {
	return r.Status >= 500
}

func (r *Response) ContentType() string {
	if ct, ok := r.Headers["Content-Type"]; ok {
		return ct
	}
	if ct, ok := r.Headers["content-type"]; ok {
		return ct
	}
	return ""
}
