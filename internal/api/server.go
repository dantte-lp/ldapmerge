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
	"ldapmerge/internal/version"
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

// DatabaseInfo contains database information for health check
type DatabaseInfo struct {
	Path         string `json:"path" doc:"Database file path" example:"/home/user/.ldapmerge/data.db"`
	Size         int64  `json:"size" doc:"Database size in bytes" example:"45056"`
	SizeHuman    string `json:"size_human" doc:"Human-readable database size" example:"44.0 KB"`
	Version      string `json:"version" doc:"SQLite version" example:"3.46.0"`
	Tables       int    `json:"tables" doc:"Number of application tables" example:"2"`
	WALMode      bool   `json:"wal_mode" doc:"Write-Ahead Logging enabled" example:"true"`
	HistoryCount int64  `json:"history_count" doc:"Number of history entries" example:"10"`
	ConfigCount  int64  `json:"config_count" doc:"Number of saved NSX configs" example:"2"`
}

// HealthOutput is the response for health check
type HealthOutput struct {
	Body struct {
		Status   string        `json:"status" example:"ok" doc:"Health status"`
		Version  string        `json:"version" example:"1.0.0" doc:"API version"`
		Database *DatabaseInfo `json:"database,omitempty" doc:"Database information"`
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
	config := huma.DefaultConfig("ldapmerge", version.Short())

	// OpenAPI 3.x Info Object
	config.Info.Title = "ldapmerge API"
	config.Info.Version = version.Short()
	config.Info.Description = `**LDAP Configuration Merger for VMware NSX 4.2**

# ldapmerge API

REST API for merging LDAP server configurations with SSL certificates and synchronizing with VMware NSX.

## Overview

This API provides endpoints for:
- **Merging** LDAP configurations with certificate data from Ansible
- **Storing** merge operation history in SQLite
- **Managing** NSX connection configurations

## Workflow

1. Fetch LDAP configuration from NSX (or provide JSON file)
2. Obtain SSL certificates from LDAP servers (via Ansible)
3. Use this API to merge configurations with certificates
4. Push the result back to NSX

## Authentication

> **Note:** This API does not implement authentication.
> Use a reverse proxy (nginx, traefik) for production deployments.

## Related Resources

- [VMware NSX 4.2 LDAP Identity Sources API](https://developer.broadcom.com/xapis/nsx-t-data-center-rest-api/4.2/)
- [GitHub Repository](https://github.com/dantte-lp/ldapmerge)
`
	config.Info.Contact = &huma.Contact{
		Name:  "Pavel Lavrukhin",
		URL:   "https://github.com/dantte-lp/ldapmerge",
		Email: "admin@lavrukhin.net",
	}
	config.Info.License = &huma.License{
		Name: "MIT",
		URL:  "https://opensource.org/licenses/MIT",
	}
	config.Info.TermsOfService = "https://github.com/dantte-lp/ldapmerge/blob/main/LICENSE"

	// Servers
	config.Servers = []*huma.Server{
		{URL: "http://localhost:8080", Description: "Local development server"},
		{URL: "https://api.example.com", Description: "Production server (example)"},
	}

	// External Documentation
	config.Extensions = map[string]any{
		"externalDocs": map[string]string{
			"description": "Full documentation on GitHub",
			"url":         "https://github.com/dantte-lp/ldapmerge/blob/main/docs/API.md",
		},
	}

	// Tags with descriptions
	config.Tags = []*huma.Tag{
		{
			Name:        "merge",
			Description: "Operations for merging LDAP configurations with SSL certificates",
		},
		{
			Name:        "history",
			Description: "Merge operation history stored in SQLite database",
		},
		{
			Name:        "config",
			Description: "NSX Manager connection configuration management",
		},
		{
			Name:        "system",
			Description: "System endpoints for health checks and monitoring",
		},
	}

	// Disable default docs, we'll add Scalar manually
	config.DocsPath = ""

	api := humabunrouter.New(s.router, config)

	// Scalar API Documentation
	s.router.GET("/docs", func(w http.ResponseWriter, r bunrouter.Request) error {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := w.Write([]byte(scalarHTML))
		return err
	})

	// Merge endpoints
	huma.Register(api, huma.Operation{
		OperationID: "merge",
		Method:      http.MethodPost,
		Path:        "/api/merge",
		Summary:     "Merge LDAP configs with certificates",
		Description: `Merges initial LDAP domain configurations with SSL certificate data.

## Request Body

The request body must contain two fields:
- **initial**: Array of domain configurations (from NSX or JSON file)
- **response**: Certificate response data (from Ansible)

## Merge Logic

Certificates are matched to LDAP servers by exact URL match.
Each certificate from the response is added to the corresponding server's ` + "`certificates`" + ` array.

## Side Effects

The merge result is automatically saved to the history database for auditing purposes.`,
		Tags: []string{"merge"},
	}, s.handleMerge)

	// Health endpoint
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/api/health",
		Summary:     "Health check",
		Description: `Returns the health status of the API server and database information.

## Response includes:

- **status**: Server health status
- **version**: API version
- **database**: SQLite database information
  - path, size, SQLite version
  - WAL mode status
  - record counts (history, configs)

## Use cases:

- Kubernetes liveness/readiness probes
- Load balancer health checks
- Monitoring and alerting systems
- Database diagnostics`,
		Tags: []string{"system"},
	}, s.handleHealth)

	// History endpoints
	huma.Register(api, huma.Operation{
		OperationID: "listHistory",
		Method:      http.MethodGet,
		Path:        "/api/history",
		Summary:     "List merge history",
		Description: `Returns all merge operation history entries.

Each entry contains:
- **id**: Unique identifier
- **created_at**: Timestamp of the merge operation
- **initial**: Original configuration before merge
- **response**: Certificate data used for merge
- **result**: Final merged configuration`,
		Tags:          []string{"history"},
		DefaultStatus: http.StatusOK,
	}, s.handleListHistory)

	huma.Register(api, huma.Operation{
		OperationID: "getHistory",
		Method:      http.MethodGet,
		Path:        "/api/history/{id}",
		Summary:     "Get history entry",
		Description: `Returns a specific history entry by ID.

The entry includes full data for:
- Initial configuration
- Certificate response
- Merged result`,
		Tags:          []string{"history"},
		DefaultStatus: http.StatusOK,
	}, s.handleGetHistory)

	// NSX Config endpoints
	huma.Register(api, huma.Operation{
		OperationID: "listConfigs",
		Method:      http.MethodGet,
		Path:        "/api/configs",
		Summary:     "List NSX configurations",
		Description: `Returns all saved NSX Manager connection configurations.

> **Security Note:** Passwords are never returned in API responses.`,
		Tags:          []string{"config"},
		DefaultStatus: http.StatusOK,
	}, s.handleListConfigs)

	huma.Register(api, huma.Operation{
		OperationID: "createConfig",
		Method:      http.MethodPost,
		Path:        "/api/configs",
		Summary:     "Create NSX configuration",
		Description: `Saves a new NSX Manager connection configuration.

## Required Fields

- **name**: Unique name for this configuration
- **host**: NSX Manager URL (e.g., ` + "`https://nsx.example.com`" + `)
- **username**: API username

## Optional Fields

- **password**: API password (stored securely)
- **description**: Human-readable description
- **insecure**: Skip TLS certificate verification`,
		Tags:          []string{"config"},
		DefaultStatus: http.StatusCreated,
	}, s.handleCreateConfig)

	huma.Register(api, huma.Operation{
		OperationID: "getConfig",
		Method:      http.MethodGet,
		Path:        "/api/configs/{id}",
		Summary:     "Get NSX configuration",
		Description: `Returns a specific NSX configuration by ID.

> **Security Note:** Password field is never included in the response.`,
		Tags:          []string{"config"},
		DefaultStatus: http.StatusOK,
	}, s.handleGetConfig)

	huma.Register(api, huma.Operation{
		OperationID: "deleteConfig",
		Method:      http.MethodDelete,
		Path:        "/api/configs/{id}",
		Summary:     "Delete NSX configuration",
		Description: `Permanently deletes an NSX configuration by ID.

This action cannot be undone.`,
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
	output.Body.Version = version.Short()

	// Add database info if available
	if s.repo != nil {
		if dbInfo, err := s.repo.GetDBInfo(ctx); err == nil {
			output.Body.Database = &DatabaseInfo{
				Path:         dbInfo.Path,
				Size:         dbInfo.Size,
				SizeHuman:    dbInfo.SizeHuman,
				Version:      dbInfo.Version,
				Tables:       dbInfo.Tables,
				WALMode:      dbInfo.WALMode,
				HistoryCount: dbInfo.HistoryCount,
				ConfigCount:  dbInfo.ConfigCount,
			}
		}
	}

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

// Scalar API Documentation HTML
const scalarHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ldapmerge API Documentation</title>
    <meta name="description" content="LDAP Configuration Merger for VMware NSX 4.2 - API Documentation">
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>🔀</text></svg>">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <script
        id="api-reference"
        data-url="/openapi.json"
        data-configuration='{
            "theme": "kepler",
            "layout": "modern",
            "darkMode": true,
            "hiddenClients": ["unirest"],
            "defaultHttpClient": {
                "targetKey": "shell",
                "clientKey": "curl"
            },
            "metaData": {
                "title": "ldapmerge API",
                "description": "LDAP Configuration Merger for VMware NSX 4.2",
                "ogDescription": "REST API for merging LDAP configurations with SSL certificates"
            },
            "searchHotKey": "k"
        }'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
