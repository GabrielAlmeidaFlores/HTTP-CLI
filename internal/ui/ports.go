package ui

import (
	"context"

	"github.com/user/http-cli/internal/models"
)

type RequestStore interface {
	SaveRequest(ctx context.Context, req *models.Request) error
	DeleteRequest(ctx context.Context, id string) error
	ListRequests(ctx context.Context) ([]*models.Request, error)
	AddHistory(ctx context.Context, resp *models.Response) error
}

type HTTPExecutor interface {
	Execute(ctx context.Context, req *models.Request, envVars map[string]string) (*models.Response, error)
}
