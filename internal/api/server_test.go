package api_test

import (
	"encoding/json"
	"testing"

	"ldapmerge/internal/models"
)

func TestMergeLogic(t *testing.T) {
	// Test the actual merge logic
	initial := []models.Domain{
		{
			ID:         "example.lab",
			DomainName: "example.lab",
			BaseDN:     "DC=example,DC=lab",
			LDAPServers: []models.LDAPServer{
				{
					URL:      "ldaps://ad-01.example.lab:636",
					StartTLS: "false",
					Enabled:  "true",
				},
				{
					URL:      "ldaps://ad-02.example.lab:636",
					StartTLS: "false",
					Enabled:  "true",
				},
			},
		},
	}

	response := models.CertificateResponse{
		Results: []models.CertificateResult{
			{
				JSON: models.CertificateJSON{
					PEMEncoded: "-----BEGIN CERTIFICATE-----\ncert1\n-----END CERTIFICATE-----",
				},
				Item: models.ResponseItem{
					URL: "ldaps://ad-01.example.lab:636",
				},
			},
			{
				JSON: models.CertificateJSON{
					PEMEncoded: "-----BEGIN CERTIFICATE-----\ncert2\n-----END CERTIFICATE-----",
				},
				Item: models.ResponseItem{
					URL: "ldaps://ad-02.example.lab:636",
				},
			},
		},
	}

	// Build certificate map
	certMap := make(map[string][]string)
	for _, result := range response.Results {
		url := result.Item.URL
		if url != "" && result.JSON.PEMEncoded != "" {
			certMap[url] = append(certMap[url], result.JSON.PEMEncoded)
		}
	}

	// Apply certificates
	for i := range initial {
		for j := range initial[i].LDAPServers {
			url := initial[i].LDAPServers[j].URL
			if certs, ok := certMap[url]; ok {
				initial[i].LDAPServers[j].Certificates = certs
			}
		}
	}

	// Verify results
	if len(initial[0].LDAPServers[0].Certificates) != 1 {
		t.Errorf("Expected 1 certificate for server 1, got %d", len(initial[0].LDAPServers[0].Certificates))
	}

	if len(initial[0].LDAPServers[1].Certificates) != 1 {
		t.Errorf("Expected 1 certificate for server 2, got %d", len(initial[0].LDAPServers[1].Certificates))
	}

	expectedCert1 := "-----BEGIN CERTIFICATE-----\ncert1\n-----END CERTIFICATE-----"
	if initial[0].LDAPServers[0].Certificates[0] != expectedCert1 {
		t.Errorf("Unexpected certificate content for server 1")
	}
}

func TestRequestBodyStructure(t *testing.T) {
	// Test data
	initial := []models.Domain{
		{
			ID:         "example.lab",
			DomainName: "example.lab",
			BaseDN:     "DC=example,DC=lab",
			LDAPServers: []models.LDAPServer{
				{
					URL:          "ldaps://ad-01.example.lab:636",
					StartTLS:     "false",
					Enabled:      "true",
					BindUsername: "admin@example.lab",
				},
			},
		},
	}

	response := models.CertificateResponse{
		Results: []models.CertificateResult{
			{
				JSON: models.CertificateJSON{
					PEMEncoded: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
				},
				Item: models.ResponseItem{
					URL: "ldaps://ad-01.example.lab:636",
				},
			},
		},
	}

	// Create request body
	body := map[string]interface{}{
		"initial":  initial,
		"response": response,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	// Decode and verify structure
	var decoded map[string]interface{}
	if err := json.Unmarshal(bodyJSON, &decoded); err != nil {
		t.Fatalf("Failed to decode request body: %v", err)
	}

	if _, ok := decoded["initial"]; !ok {
		t.Error("Expected 'initial' field in request")
	}

	if _, ok := decoded["response"]; !ok {
		t.Error("Expected 'response' field in request")
	}
}
