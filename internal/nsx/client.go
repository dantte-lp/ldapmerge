package nsx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is an NSX API client.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// ClientConfig holds configuration for NSX client.
type ClientConfig struct {
	Host     string
	Username string
	Password string
	Insecure bool
	Timeout  time.Duration
}

// LDAPIdentitySource represents NSX LDAP identity source.
// Based on NSX 4.2 API: /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}.
type LDAPIdentitySource struct {
	ID                     string       `json:"id,omitempty"`
	DisplayName            string       `json:"display_name,omitempty"`
	Description            string       `json:"description,omitempty"`
	ResourceType           string       `json:"resource_type,omitempty"`
	DomainName             string       `json:"domain_name"`
	BaseDN                 string       `json:"base_dn"`
	AlternativeDomainNames []string     `json:"alternative_domain_names,omitempty"`
	LDAPServers            []LDAPServer `json:"ldap_servers"`
	Path                   string       `json:"path,omitempty"`
	RealizationID          string       `json:"realization_id,omitempty"`
	RelativePath           string       `json:"relative_path,omitempty"`
}

// LDAPServer represents an LDAP server in NSX.
type LDAPServer struct {
	URL          string   `json:"url"`
	UseStartTLS  bool     `json:"use_starttls,omitempty"`
	Enabled      bool     `json:"enabled,omitempty"`
	BindIdentity string   `json:"bind_identity,omitempty"`
	Password     string   `json:"password,omitempty"`
	Certificates []string `json:"certificates,omitempty"`
}

// LDAPIdentitySourceListResult represents list response.
type LDAPIdentitySourceListResult struct {
	Results     []LDAPIdentitySource `json:"results"`
	ResultCount int                  `json:"result_count"`
	Cursor      string               `json:"cursor,omitempty"`
}

// ProbeResult represents the result of a probe operation.
type ProbeResult struct {
	Results []ProbeResultItem `json:"results"`
}

// ProbeResultItem represents a single probe result.
type ProbeResultItem struct {
	LDAPServerURL string `json:"ldap_server_url"`
	Success       bool   `json:"success"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// FetchCertificateRequest represents request to fetch certificate.
type FetchCertificateRequest struct {
	LDAPServerURL string `json:"ldap_server_url"`
}

// FetchCertificateResult represents certificate fetch response.
type FetchCertificateResult struct {
	PEMEncoded string              `json:"pem_encoded"`
	Details    []CertificateDetail `json:"details,omitempty"`
}

// CertificateDetail contains certificate subject info.
type CertificateDetail struct {
	SubjectCN          string `json:"subject_cn,omitempty"`
	SubjectDN          string `json:"subject_dn,omitempty"`
	IssuerCN           string `json:"issuer_cn,omitempty"`
	IssuerDN           string `json:"issuer_dn,omitempty"`
	NotBefore          string `json:"not_before,omitempty"`
	NotAfter           string `json:"not_after,omitempty"`
	SerialNumber       string `json:"serial_number,omitempty"`
	SignatureAlgorithm string `json:"signature_algorithm,omitempty"`
}

// SearchRequest represents LDAP search request
type SearchRequest struct {
	FilterValue string `json:"filter_value"`
}

// SearchResult represents LDAP search response
type SearchResult struct {
	Results     []SearchResultItem `json:"results"`
	ResultCount int                `json:"result_count"`
}

// SearchResultItem represents a user or group from search
type SearchResultItem struct {
	DN          string `json:"dn"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "user" or "group"
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
}

// APIError represents NSX API error response
type APIError struct {
	HTTPStatus   int    `json:"http_status"`
	ErrorCode    int    `json:"error_code"`
	ModuleName   string `json:"module_name"`
	ErrorMessage string `json:"error_message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("NSX API error %d: %s (code: %d)", e.HTTPStatus, e.ErrorMessage, e.ErrorCode)
}

// NewClient creates a new NSX API client.
func NewClient(cfg ClientConfig) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Insecure, //nolint:gosec // G402: Intentionally configurable for self-signed certs
		},
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:  cfg.Host,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

// doRequest performs an HTTP request to NSX API.
//
//nolint:unparam // statusCode return value used for future error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.ErrorMessage != "" {
			apiErr.HTTPStatus = resp.StatusCode
			return nil, resp.StatusCode, &apiErr
		}
		return nil, resp.StatusCode, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// ListLDAPIdentitySources retrieves all LDAP identity sources
// GET /policy/api/v1/aaa/ldap-identity-sources
func (c *Client) ListLDAPIdentitySources(ctx context.Context) (*LDAPIdentitySourceListResult, error) {
	data, _, err := c.doRequest(ctx, http.MethodGet, "/policy/api/v1/aaa/ldap-identity-sources", nil)
	if err != nil {
		return nil, err
	}

	var result LDAPIdentitySourceListResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetLDAPIdentitySource retrieves a specific LDAP identity source by ID
// GET /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}
func (c *Client) GetLDAPIdentitySource(ctx context.Context, id string) (*LDAPIdentitySource, error) {
	path := fmt.Sprintf("/policy/api/v1/aaa/ldap-identity-sources/%s", url.PathEscape(id))
	data, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result LDAPIdentitySource
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// CreateOrUpdateLDAPIdentitySource creates or updates an LDAP identity source (PATCH)
// PATCH /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}
func (c *Client) CreateOrUpdateLDAPIdentitySource(ctx context.Context, source *LDAPIdentitySource) (*LDAPIdentitySource, error) {
	path := fmt.Sprintf("/policy/api/v1/aaa/ldap-identity-sources/%s", url.PathEscape(source.ID))
	data, _, err := c.doRequest(ctx, http.MethodPatch, path, source)
	if err != nil {
		return nil, err
	}

	var result LDAPIdentitySource
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// PutLDAPIdentitySource creates or replaces an LDAP identity source (PUT - full update)
// PUT /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}
func (c *Client) PutLDAPIdentitySource(ctx context.Context, source *LDAPIdentitySource) (*LDAPIdentitySource, error) {
	path := fmt.Sprintf("/policy/api/v1/aaa/ldap-identity-sources/%s", url.PathEscape(source.ID))
	data, _, err := c.doRequest(ctx, http.MethodPut, path, source)
	if err != nil {
		return nil, err
	}

	var result LDAPIdentitySource
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DeleteLDAPIdentitySource deletes an LDAP identity source
// DELETE /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}
func (c *Client) DeleteLDAPIdentitySource(ctx context.Context, id string) error {
	path := fmt.Sprintf("/policy/api/v1/aaa/ldap-identity-sources/%s", url.PathEscape(id))
	_, _, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// ProbeLDAPServer tests connection to an LDAP server
// POST /policy/api/v1/aaa/ldap-identity-sources?action=probe_ldap_server
func (c *Client) ProbeLDAPServer(ctx context.Context, source *LDAPIdentitySource) (*ProbeResult, error) {
	path := "/policy/api/v1/aaa/ldap-identity-sources?action=probe_ldap_server"
	data, _, err := c.doRequest(ctx, http.MethodPost, path, source)
	if err != nil {
		return nil, err
	}

	var result ProbeResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ProbeIdentitySource verifies LDAP identity source configuration before creation
// POST /policy/api/v1/aaa/ldap-identity-sources?action=probe_identity_source
func (c *Client) ProbeIdentitySource(ctx context.Context, source *LDAPIdentitySource) (*ProbeResult, error) {
	path := "/policy/api/v1/aaa/ldap-identity-sources?action=probe_identity_source"
	data, _, err := c.doRequest(ctx, http.MethodPost, path, source)
	if err != nil {
		return nil, err
	}

	var result ProbeResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// FetchCertificate retrieves the SSL certificate from an LDAP server
// POST /policy/api/v1/aaa/ldap-identity-sources?action=fetch_certificate
func (c *Client) FetchCertificate(ctx context.Context, ldapServerURL string) (*FetchCertificateResult, error) {
	path := "/policy/api/v1/aaa/ldap-identity-sources?action=fetch_certificate"
	req := FetchCertificateRequest{LDAPServerURL: ldapServerURL}

	data, _, err := c.doRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	var result FetchCertificateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ProbeConfiguredSource tests an existing LDAP identity source
// POST /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}?action=probe
func (c *Client) ProbeConfiguredSource(ctx context.Context, id string) (*ProbeResult, error) {
	path := fmt.Sprintf("/policy/api/v1/aaa/ldap-identity-sources/%s?action=probe", url.PathEscape(id))
	data, _, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	var result ProbeResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Search searches for users and groups in an LDAP identity source
// POST /policy/api/v1/aaa/ldap-identity-sources/{ldap-identity-source-id}/search
func (c *Client) Search(ctx context.Context, id string, filterValue string) (*SearchResult, error) {
	path := fmt.Sprintf("/policy/api/v1/aaa/ldap-identity-sources/%s/search", url.PathEscape(id))
	req := SearchRequest{FilterValue: filterValue}

	data, _, err := c.doRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
