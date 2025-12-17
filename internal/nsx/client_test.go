package nsx_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"ldapmerge/internal/nsx"
	"ldapmerge/internal/nsx/mock"
)

func setupTestServer() (*httptest.Server, *nsx.Client) {
	mockServer := mock.NewServer()
	ts := httptest.NewServer(mockServer)

	client := nsx.NewClient(nsx.ClientConfig{
		Host:     ts.URL,
		Username: "admin",
		Password: "secret",
		Insecure: true,
	})

	return ts, client
}

func TestListLDAPIdentitySources(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()
	result, err := client.ListLDAPIdentitySources(ctx)
	if err != nil {
		t.Fatalf("ListLDAPIdentitySources failed: %v", err)
	}

	if result.ResultCount < 1 {
		t.Error("Expected at least one LDAP identity source")
	}

	// Check that example.lab exists
	found := false
	for _, source := range result.Results {
		if source.ID == "example.lab" {
			found = true
			if source.DomainName != "example.lab" {
				t.Errorf("Expected domain_name 'example.lab', got '%s'", source.DomainName)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find 'example.lab' in results")
	}
}

func TestGetLDAPIdentitySource(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()

	// Test existing source
	source, err := client.GetLDAPIdentitySource(ctx, "example.lab")
	if err != nil {
		t.Fatalf("GetLDAPIdentitySource failed: %v", err)
	}

	if source.ID != "example.lab" {
		t.Errorf("Expected ID 'example.lab', got '%s'", source.ID)
	}

	if len(source.LDAPServers) < 1 {
		t.Error("Expected at least one LDAP server")
	}

	// Test non-existing source
	_, err = client.GetLDAPIdentitySource(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existing source")
	}
}

func TestPutLDAPIdentitySource(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()

	newSource := &nsx.LDAPIdentitySource{
		ID:         "test.domain",
		DomainName: "test.domain",
		BaseDN:     "DC=test,DC=domain",
		LDAPServers: []nsx.LDAPServer{
			{
				URL:          "ldaps://dc1.test.domain:636",
				Enabled:      true,
				BindIdentity: "admin@test.domain",
			},
		},
	}

	result, err := client.PutLDAPIdentitySource(ctx, newSource)
	if err != nil {
		t.Fatalf("PutLDAPIdentitySource failed: %v", err)
	}

	if result.ID != "test.domain" {
		t.Errorf("Expected ID 'test.domain', got '%s'", result.ID)
	}

	// Verify it was created
	source, err := client.GetLDAPIdentitySource(ctx, "test.domain")
	if err != nil {
		t.Fatalf("GetLDAPIdentitySource after PUT failed: %v", err)
	}

	if source.DomainName != "test.domain" {
		t.Errorf("Expected domain_name 'test.domain', got '%s'", source.DomainName)
	}
}

func TestDeleteLDAPIdentitySource(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()

	// First create a source to delete
	newSource := &nsx.LDAPIdentitySource{
		ID:         "to-delete",
		DomainName: "to-delete.domain",
		BaseDN:     "DC=to-delete,DC=domain",
		LDAPServers: []nsx.LDAPServer{
			{URL: "ldaps://dc1.to-delete.domain:636"},
		},
	}

	_, err := client.PutLDAPIdentitySource(ctx, newSource)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Delete it
	err = client.DeleteLDAPIdentitySource(ctx, "to-delete")
	if err != nil {
		t.Fatalf("DeleteLDAPIdentitySource failed: %v", err)
	}

	// Verify it's gone
	_, err = client.GetLDAPIdentitySource(ctx, "to-delete")
	if err == nil {
		t.Error("Expected error when getting deleted source")
	}
}

func TestProbeConfiguredSource(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()

	result, err := client.ProbeConfiguredSource(ctx, "example.lab")
	if err != nil {
		t.Fatalf("ProbeConfiguredSource failed: %v", err)
	}

	if len(result.Results) < 1 {
		t.Error("Expected at least one probe result")
	}

	for _, item := range result.Results {
		if !item.Success {
			t.Errorf("Expected probe success for %s", item.LDAPServerURL)
		}
	}
}

func TestFetchCertificate(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()

	result, err := client.FetchCertificate(ctx, "ldaps://ad01.example.com:636")
	if err != nil {
		t.Fatalf("FetchCertificate failed: %v", err)
	}

	if result.PEMEncoded == "" {
		t.Error("Expected non-empty PEM certificate")
	}

	if len(result.Details) < 1 {
		t.Error("Expected certificate details")
	}
}

func TestSearch(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	ctx := context.Background()

	result, err := client.Search(ctx, "example.lab", "john")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.ResultCount < 1 {
		t.Error("Expected at least one search result")
	}

	// Check that we got both user and group
	foundUser := false
	foundGroup := false
	for _, item := range result.Results {
		if item.Type == "user" {
			foundUser = true
		}
		if item.Type == "group" {
			foundGroup = true
		}
	}

	if !foundUser {
		t.Error("Expected to find a user in results")
	}
	if !foundGroup {
		t.Error("Expected to find a group in results")
	}
}

func TestAuthenticationFailure(t *testing.T) {
	mockServer := mock.NewServer()
	ts := httptest.NewServer(mockServer)
	defer ts.Close()

	// Client with wrong credentials
	client := nsx.NewClient(nsx.ClientConfig{
		Host:     ts.URL,
		Username: "wrong",
		Password: "wrong",
		Insecure: true,
	})

	ctx := context.Background()
	_, err := client.ListLDAPIdentitySources(ctx)
	if err == nil {
		t.Error("Expected authentication error")
	}
}
