package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"ldapmerge/internal/nsx"
)

// Server is a mock NSX API server for testing
type Server struct {
	mux      *http.ServeMux
	mu       sync.RWMutex
	sources  map[string]*nsx.LDAPIdentitySource
	Username string
	Password string
}

// NewServer creates a new mock NSX server
func NewServer() *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		sources:  make(map[string]*nsx.LDAPIdentitySource),
		Username: "admin",
		Password: "secret",
	}

	s.setupRoutes()
	s.seedData()

	return s
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Basic auth check
	user, pass, ok := r.BasicAuth()
	if !ok || user != s.Username || pass != s.Password {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    401,
			"error_message": "Authentication required",
		})
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/policy/api/v1/aaa/ldap-identity-sources", s.handleLDAPIdentitySources)
	s.mux.HandleFunc("/policy/api/v1/aaa/ldap-identity-sources/", s.handleLDAPIdentitySource)
}

func (s *Server) seedData() {
	// Add sample LDAP identity sources
	s.sources["example.lab"] = &nsx.LDAPIdentitySource{
		ID:           "example.lab",
		DisplayName:  "Example Lab Domain",
		Description:  "Test LDAP identity source",
		ResourceType: "LdapIdentitySource",
		DomainName:   "example.lab",
		BaseDN:       "DC=example,DC=lab",
		AlternativeDomainNames: []string{
			"msk.example.lab",
			"nsk.example.lab",
		},
		LDAPServers: []nsx.LDAPServer{
			{
				URL:          "ldaps://ad-01.example.lab:636",
				UseStartTLS:  false,
				Enabled:      true,
				BindIdentity: "sync_to_ad@example.lab",
				Password:     "secret",
				Certificates: []string{
					"-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
				},
			},
			{
				URL:          "ldaps://ad-02.example.lab:636",
				UseStartTLS:  false,
				Enabled:      true,
				BindIdentity: "sync_to_ad@example.lab",
				Password:     "secret",
				Certificates: []string{
					"-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
				},
			},
		},
	}

	s.sources["example.org"] = &nsx.LDAPIdentitySource{
		ID:           "example.org",
		DisplayName:  "Example Org Domain",
		ResourceType: "LdapIdentitySource",
		DomainName:   "example.org",
		BaseDN:       "DC=example,DC=org",
		LDAPServers: []nsx.LDAPServer{
			{
				URL:          "ldaps://dc01.example.org:636",
				UseStartTLS:  false,
				Enabled:      true,
				BindIdentity: "ldap_reader@example.org",
				Password:     "secret",
			},
		},
	}
}

func (s *Server) handleLDAPIdentitySources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	action := r.URL.Query().Get("action")

	switch r.Method {
	case http.MethodGet:
		s.listSources(w, r)
	case http.MethodPost:
		switch action {
		case "probe_ldap_server":
			s.probeLDAPServer(w, r)
		case "probe_identity_source":
			s.probeIdentitySource(w, r)
		case "fetch_certificate":
			s.fetchCertificate(w, r)
		default:
			http.Error(w, "Unknown action", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLDAPIdentitySource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/policy/api/v1/aaa/ldap-identity-sources/")
	parts := strings.Split(path, "/")
	id := parts[0]

	// Check for action parameter
	action := r.URL.Query().Get("action")

	// Check for search endpoint
	if len(parts) > 1 && parts[1] == "search" {
		s.searchSource(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getSource(w, r, id)
	case http.MethodPut:
		s.putSource(w, r, id)
	case http.MethodPatch:
		s.patchSource(w, r, id)
	case http.MethodDelete:
		s.deleteSource(w, r, id)
	case http.MethodPost:
		if action == "probe" {
			s.probeConfiguredSource(w, r, id)
		} else {
			http.Error(w, "Unknown action", http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]nsx.LDAPIdentitySource, 0, len(s.sources))
	for _, source := range s.sources {
		results = append(results, *source)
	}

	response := nsx.LDAPIdentitySourceListResult{
		Results:     results,
		ResultCount: len(results),
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) getSource(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.RLock()
	source, ok := s.sources[id]
	s.mu.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    404,
			"error_message": fmt.Sprintf("LDAP identity source '%s' not found", id),
		})
		return
	}

	json.NewEncoder(w).Encode(source)
}

func (s *Server) putSource(w http.ResponseWriter, r *http.Request, id string) {
	var source nsx.LDAPIdentitySource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    400,
			"error_message": "Invalid JSON body",
		})
		return
	}

	source.ID = id
	if source.ResourceType == "" {
		source.ResourceType = "LdapIdentitySource"
	}

	s.mu.Lock()
	s.sources[id] = &source
	s.mu.Unlock()

	json.NewEncoder(w).Encode(source)
}

func (s *Server) patchSource(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.sources[id]
	if !ok {
		existing = &nsx.LDAPIdentitySource{ID: id, ResourceType: "LdapIdentitySource"}
	}

	var patch nsx.LDAPIdentitySource
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    400,
			"error_message": "Invalid JSON body",
		})
		return
	}

	// Apply patch (simplified)
	if patch.DisplayName != "" {
		existing.DisplayName = patch.DisplayName
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}
	if patch.DomainName != "" {
		existing.DomainName = patch.DomainName
	}
	if patch.BaseDN != "" {
		existing.BaseDN = patch.BaseDN
	}
	if len(patch.AlternativeDomainNames) > 0 {
		existing.AlternativeDomainNames = patch.AlternativeDomainNames
	}
	if len(patch.LDAPServers) > 0 {
		existing.LDAPServers = patch.LDAPServers
	}

	s.sources[id] = existing
	json.NewEncoder(w).Encode(existing)
}

func (s *Server) deleteSource(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sources[id]; !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    404,
			"error_message": fmt.Sprintf("LDAP identity source '%s' not found", id),
		})
		return
	}

	delete(s.sources, id)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) probeLDAPServer(w http.ResponseWriter, r *http.Request) {
	var source nsx.LDAPIdentitySource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	results := make([]nsx.ProbeResultItem, len(source.LDAPServers))
	for i, server := range source.LDAPServers {
		results[i] = nsx.ProbeResultItem{
			LDAPServerURL: server.URL,
			Success:       true,
		}
	}

	json.NewEncoder(w).Encode(nsx.ProbeResult{Results: results})
}

func (s *Server) probeIdentitySource(w http.ResponseWriter, r *http.Request) {
	var source nsx.LDAPIdentitySource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	results := make([]nsx.ProbeResultItem, len(source.LDAPServers))
	for i, server := range source.LDAPServers {
		results[i] = nsx.ProbeResultItem{
			LDAPServerURL: server.URL,
			Success:       true,
		}
	}

	json.NewEncoder(w).Encode(nsx.ProbeResult{Results: results})
}

func (s *Server) probeConfiguredSource(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.RLock()
	source, ok := s.sources[id]
	s.mu.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    404,
			"error_message": fmt.Sprintf("LDAP identity source '%s' not found", id),
		})
		return
	}

	results := make([]nsx.ProbeResultItem, len(source.LDAPServers))
	for i, server := range source.LDAPServers {
		results[i] = nsx.ProbeResultItem{
			LDAPServerURL: server.URL,
			Success:       true,
		}
	}

	json.NewEncoder(w).Encode(nsx.ProbeResult{Results: results})
}

func (s *Server) fetchCertificate(w http.ResponseWriter, r *http.Request) {
	var req nsx.FetchCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generate mock certificate
	result := nsx.FetchCertificateResult{
		PEMEncoded: fmt.Sprintf("-----BEGIN CERTIFICATE-----\nMock certificate for %s\n-----END CERTIFICATE-----", req.LDAPServerURL),
		Details: []nsx.CertificateDetail{
			{
				SubjectCN:          extractHostFromURL(req.LDAPServerURL),
				SubjectDN:          fmt.Sprintf("CN=%s", extractHostFromURL(req.LDAPServerURL)),
				IssuerCN:           "Mock CA",
				NotBefore:          "2024-01-01T00:00:00Z",
				NotAfter:           "2025-12-31T23:59:59Z",
				SignatureAlgorithm: "SHA256-RSA",
			},
		},
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) searchSource(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.RLock()
	_, ok := s.sources[id]
	s.mu.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error_code":    404,
			"error_message": fmt.Sprintf("LDAP identity source '%s' not found", id),
		})
		return
	}

	var req nsx.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Return mock search results
	result := nsx.SearchResult{
		Results: []nsx.SearchResultItem{
			{
				DN:          fmt.Sprintf("CN=%s,OU=Users,DC=example,DC=lab", req.FilterValue),
				Name:        req.FilterValue,
				Type:        "user",
				DisplayName: fmt.Sprintf("Test User %s", req.FilterValue),
				Email:       fmt.Sprintf("%s@example.lab", req.FilterValue),
			},
			{
				DN:          fmt.Sprintf("CN=%s,OU=Groups,DC=example,DC=lab", req.FilterValue),
				Name:        fmt.Sprintf("%s-group", req.FilterValue),
				Type:        "group",
				DisplayName: fmt.Sprintf("Group for %s", req.FilterValue),
			},
		},
		ResultCount: 2,
	}

	json.NewEncoder(w).Encode(result)
}

func extractHostFromURL(urlStr string) string {
	// Simple extraction of host from URL like ldaps://host:port
	urlStr = strings.TrimPrefix(urlStr, "ldaps://")
	urlStr = strings.TrimPrefix(urlStr, "ldap://")
	if idx := strings.Index(urlStr, ":"); idx > 0 {
		return urlStr[:idx]
	}
	return urlStr
}

// GetSources returns all sources (for testing)
func (s *Server) GetSources() map[string]*nsx.LDAPIdentitySource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*nsx.LDAPIdentitySource)
	for k, v := range s.sources {
		result[k] = v
	}
	return result
}

// SetSource sets a source (for testing)
func (s *Server) SetSource(source *nsx.LDAPIdentitySource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources[source.ID] = source
}

// ClearSources removes all sources (for testing)
func (s *Server) ClearSources() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources = make(map[string]*nsx.LDAPIdentitySource)
}
