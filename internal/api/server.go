package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humabunrouter"
	"github.com/uptrace/bunrouter"
	"github.com/uptrace/bunrouter/extra/reqlog"

	"ldapmerge/internal/merger"
	"ldapmerge/internal/models"
	"ldapmerge/internal/repository"
)

// Server represents the API server
type Server struct {
	addr   string
	router *bunrouter.Router
	merger *merger.Merger
	repo   *repository.Repository
}

// MergeInput is the request body for merge operation
type MergeInput struct {
	Body struct {
		Initial  []models.Domain            `json:"initial" doc:"Initial domain configurations"`
		Response models.CertificateResponse `json:"response" doc:"Certificate response data"`
	}
}

// MergeOutput is the response for merge operation
type MergeOutput struct {
	Body []models.Domain
}

// HealthOutput is the response for health check
type HealthOutput struct {
	Body struct {
		Status  string `json:"status" example:"ok" doc:"Health status"`
		Version string `json:"version" example:"1.0.0" doc:"API version"`
	}
}

// HistoryListOutput is the response for history list
type HistoryListOutput struct {
	Body []models.HistoryEntry
}

// HistoryInput is the path parameter for history entry
type HistoryInput struct {
	ID int64 `path:"id" doc:"History entry ID"`
}

// HistoryOutput is the response for single history entry
type HistoryOutput struct {
	Body models.HistoryEntry
}

// ConfigListOutput is the response for NSX configs list
type ConfigListOutput struct {
	Body []models.NSXConfig
}

// ConfigInput is the request for creating/updating NSX config
type ConfigInput struct {
	Body models.NSXConfig
}

// ConfigPathInput is the path parameter for config
type ConfigPathInput struct {
	ID int64 `path:"id" doc:"Config ID"`
}

// ConfigOutput is the response for single config
type ConfigOutput struct {
	Body models.NSXConfig
}

// NewServer creates a new API server
func NewServer(addr string, repo *repository.Repository) *Server {
	router := bunrouter.New(
		bunrouter.Use(reqlog.NewMiddleware()),
	)

	s := &Server{
		addr:   addr,
		router: router,
		merger: merger.New(),
		repo:   repo,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	config := huma.DefaultConfig("LDAP Merge API", "1.0.0")
	config.Info.Description = "API for merging LDAP configurations with certificate data from VMware NSX"
	config.Info.Contact = &huma.Contact{
		Name:  "API Support",
		Email: "support@example.com",
	}
	config.DocsPath = "/docs"
	config.Servers = []*huma.Server{
		{URL: "http://localhost:8080", Description: "Local development"},
	}

	api := humabunrouter.New(s.router, config)

	// Merge endpoints
	huma.Register(api, huma.Operation{
		OperationID: "merge",
		Method:      http.MethodPost,
		Path:        "/api/merge",
		Summary:     "Merge LDAP configs with certificates",
		Description: "Merge initial domain configurations with certificate response data. Result is saved to history.",
		Tags:        []string{"merge"},
	}, s.handleMerge)

	// Health endpoint
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/api/health",
		Summary:     "Health check",
		Description: "Returns the health status of the API",
		Tags:        []string{"system"},
	}, s.handleHealth)

	// History endpoints
	huma.Register(api, huma.Operation{
		OperationID:   "listHistory",
		Method:        http.MethodGet,
		Path:          "/api/history",
		Summary:       "List merge history",
		Description:   "Returns all merge operation history entries",
		Tags:          []string{"history"},
		DefaultStatus: http.StatusOK,
	}, s.handleListHistory)

	huma.Register(api, huma.Operation{
		OperationID:   "getHistory",
		Method:        http.MethodGet,
		Path:          "/api/history/{id}",
		Summary:       "Get history entry",
		Description:   "Returns a specific history entry by ID",
		Tags:          []string{"history"},
		DefaultStatus: http.StatusOK,
	}, s.handleGetHistory)

	// NSX Config endpoints
	huma.Register(api, huma.Operation{
		OperationID:   "listConfigs",
		Method:        http.MethodGet,
		Path:          "/api/configs",
		Summary:       "List NSX configurations",
		Description:   "Returns all saved NSX configurations",
		Tags:          []string{"config"},
		DefaultStatus: http.StatusOK,
	}, s.handleListConfigs)

	huma.Register(api, huma.Operation{
		OperationID:   "createConfig",
		Method:        http.MethodPost,
		Path:          "/api/configs",
		Summary:       "Create NSX configuration",
		Description:   "Save a new NSX configuration",
		Tags:          []string{"config"},
		DefaultStatus: http.StatusCreated,
	}, s.handleCreateConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "getConfig",
		Method:        http.MethodGet,
		Path:          "/api/configs/{id}",
		Summary:       "Get NSX configuration",
		Description:   "Returns a specific NSX configuration by ID",
		Tags:          []string{"config"},
		DefaultStatus: http.StatusOK,
	}, s.handleGetConfig)

	huma.Register(api, huma.Operation{
		OperationID:   "deleteConfig",
		Method:        http.MethodDelete,
		Path:          "/api/configs/{id}",
		Summary:       "Delete NSX configuration",
		Description:   "Delete a specific NSX configuration by ID",
		Tags:          []string{"config"},
		DefaultStatus: http.StatusNoContent,
	}, s.handleDeleteConfig)
}

func (s *Server) handleMerge(ctx context.Context, input *MergeInput) (*MergeOutput, error) {
	result := s.merger.Merge(input.Body.Initial, &input.Body.Response)

	// Save to history
	if s.repo != nil {
		_, err := s.repo.SaveHistory(ctx, input.Body.Initial, input.Body.Response, result)
		if err != nil {
			// Log error but don't fail the request
			// TODO: add proper logging
		}
	}

	return &MergeOutput{Body: result}, nil
}

func (s *Server) handleHealth(ctx context.Context, input *struct{}) (*HealthOutput, error) {
	output := &HealthOutput{}
	output.Body.Status = "ok"
	output.Body.Version = "1.0.0"
	return output, nil
}

func (s *Server) handleListHistory(ctx context.Context, input *struct{}) (*HistoryListOutput, error) {
	if s.repo == nil {
		return &HistoryListOutput{Body: []models.HistoryEntry{}}, nil
	}

	entries, err := s.repo.ListHistory(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list history", err)
	}

	return &HistoryListOutput{Body: entries}, nil
}

func (s *Server) handleGetHistory(ctx context.Context, input *HistoryInput) (*HistoryOutput, error) {
	if s.repo == nil {
		return nil, huma.Error404NotFound("history not available")
	}

	entry, err := s.repo.GetHistory(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("history entry not found")
	}

	return &HistoryOutput{Body: *entry}, nil
}

func (s *Server) handleListConfigs(ctx context.Context, input *struct{}) (*ConfigListOutput, error) {
	if s.repo == nil {
		return &ConfigListOutput{Body: []models.NSXConfig{}}, nil
	}

	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list configs", err)
	}

	return &ConfigListOutput{Body: configs}, nil
}

func (s *Server) handleCreateConfig(ctx context.Context, input *ConfigInput) (*ConfigOutput, error) {
	if s.repo == nil {
		return nil, huma.Error500InternalServerError("database not available", nil)
	}

	config, err := s.repo.SaveConfig(ctx, &input.Body)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to save config", err)
	}

	return &ConfigOutput{Body: *config}, nil
}

func (s *Server) handleGetConfig(ctx context.Context, input *ConfigPathInput) (*ConfigOutput, error) {
	if s.repo == nil {
		return nil, huma.Error404NotFound("config not available")
	}

	config, err := s.repo.GetConfig(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("config not found")
	}

	return &ConfigOutput{Body: *config}, nil
}

func (s *Server) handleDeleteConfig(ctx context.Context, input *ConfigPathInput) (*struct{}, error) {
	if s.repo == nil {
		return nil, huma.Error500InternalServerError("database not available", nil)
	}

	err := s.repo.DeleteConfig(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("config not found")
	}

	return nil, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return http.ListenAndServe(s.addr, s.router)
}
