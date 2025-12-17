package merger

import (
	"encoding/json"
	"fmt"
	"os"

	"ldapmerge/internal/models"
)

// Merger handles the merging of initial and response data
type Merger struct{}

// New creates a new Merger instance
func New() *Merger {
	return &Merger{}
}

// LoadInitialFromFile loads the initial domains from a JSON file
func (m *Merger) LoadInitialFromFile(path string) ([]models.Domain, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial file: %w", err)
	}

	var domains []models.Domain
	if err := json.Unmarshal(data, &domains); err != nil {
		return nil, fmt.Errorf("failed to parse initial JSON: %w", err)
	}

	return domains, nil
}

// LoadResponseFromFile loads the certificate response from a JSON file
func (m *Merger) LoadResponseFromFile(path string) (*models.CertificateResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read response file: %w", err)
	}

	var response models.CertificateResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return &response, nil
}

// buildCertificateMap creates a map from URL to certificates
func (m *Merger) buildCertificateMap(response *models.CertificateResponse) map[string][]string {
	certMap := make(map[string][]string)

	for _, result := range response.Results {
		url := result.Item.URL
		if url == "" {
			continue
		}

		if _, exists := certMap[url]; !exists {
			certMap[url] = []string{}
		}

		if result.JSON.PEMEncoded != "" {
			certMap[url] = append(certMap[url], result.JSON.PEMEncoded)
		}
	}

	return certMap
}

// Merge combines the initial domains with certificates from the response
func (m *Merger) Merge(domains []models.Domain, response *models.CertificateResponse) []models.Domain {
	certMap := m.buildCertificateMap(response)

	result := make([]models.Domain, len(domains))

	for i, domain := range domains {
		result[i] = models.Domain{
			ID:                     domain.ID,
			DomainName:             domain.DomainName,
			BaseDN:                 domain.BaseDN,
			AlternativeDomainNames: domain.AlternativeDomainNames,
			LDAPServers:            make([]models.LDAPServer, len(domain.LDAPServers)),
		}

		for j, server := range domain.LDAPServers {
			result[i].LDAPServers[j] = models.LDAPServer{
				URL:          server.URL,
				StartTLS:     server.StartTLS,
				Enabled:      server.Enabled,
				BindUsername: server.BindUsername,
				BindPassword: server.BindPassword,
			}

			if certs, exists := certMap[server.URL]; exists && len(certs) > 0 {
				result[i].LDAPServers[j].Certificates = certs
			}
		}
	}

	return result
}

// MergeFromFiles loads files and performs the merge operation
func (m *Merger) MergeFromFiles(initialPath, responsePath string) ([]models.Domain, error) {
	domains, err := m.LoadInitialFromFile(initialPath)
	if err != nil {
		return nil, err
	}

	response, err := m.LoadResponseFromFile(responsePath)
	if err != nil {
		return nil, err
	}

	return m.Merge(domains, response), nil
}

// ToJSON converts the result to formatted JSON
func (m *Merger) ToJSON(domains []models.Domain, indent bool) ([]byte, error) {
	if indent {
		return json.MarshalIndent(domains, "", "    ")
	}
	return json.Marshal(domains)
}
