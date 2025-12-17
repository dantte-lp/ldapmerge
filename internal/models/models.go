package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// LDAPServer represents an LDAP server configuration.
type LDAPServer struct {
	URL          string   `json:"url" doc:"LDAP server URL" example:"ldaps://ad-01.example.lab:636"`
	StartTLS     string   `json:"starttls" doc:"Use StartTLS" example:"false"`
	Enabled      string   `json:"enabled" doc:"Server enabled status" example:"true"`
	BindUsername string   `json:"bind_username,omitempty" doc:"Bind username for LDAP authentication" example:"sync@example.lab"`
	BindPassword string   `json:"bind_password,omitempty" doc:"Bind password (write-only)"`
	Certificates []string `json:"certificates,omitempty" doc:"PEM-encoded SSL certificates"`
}

// Domain represents a domain configuration with LDAP servers.
type Domain struct {
	ID                     string       `json:"id" doc:"Unique domain identifier" example:"example.lab"`
	DomainName             string       `json:"domain_name" doc:"Domain name" example:"example.lab"`
	BaseDN                 string       `json:"base_dn" doc:"LDAP base distinguished name" example:"DC=example,DC=lab"`
	AlternativeDomainNames []string     `json:"alternative_domain_names" doc:"Alternative domain names for this domain"`
	LDAPServers            []LDAPServer `json:"ldap_servers" doc:"List of LDAP servers for this domain"`
}

// CertificateDetail contains certificate subject info.
type CertificateDetail struct {
	SubjectCN string `json:"subject_cn" doc:"Certificate subject common name" example:"ad-01.example.lab"`
}

// CertificateJSON contains the certificate data from response.
type CertificateJSON struct {
	PEMEncoded string              `json:"pem_encoded" doc:"PEM-encoded certificate" example:"-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----"`
	Details    []CertificateDetail `json:"details" doc:"Certificate details"`
}

// ResponseItem represents the item from response (matching LDAP server).
type ResponseItem struct {
	URL      string `json:"url" doc:"LDAP server URL used for matching" example:"ldaps://ad-01.example.lab:636"`
	StartTLS string `json:"starttls" doc:"StartTLS flag" example:"false"`
	Enabled  string `json:"enabled" doc:"Server enabled flag" example:"true"`
}

// CertificateResult represents a single result from the response JSON.
type CertificateResult struct {
	JSON           CertificateJSON `json:"json" doc:"Certificate data"`
	Item           ResponseItem    `json:"item" doc:"Server identifier used for URL matching"`
	AnsibleLoopVar string          `json:"ansible_loop_var,omitempty" doc:"Ansible loop variable name"`
}

// CertificateResponse represents the full response JSON structure from Ansible.
type CertificateResponse struct {
	Results []CertificateResult `json:"results" doc:"Array of certificate results from Ansible"`
}

// MergeRequest is the API request for merging operation.
type MergeRequest struct {
	Initial  []Domain            `json:"initial"`
	Response CertificateResponse `json:"response"`
}

// MergeResponse is the API response with merged data.
type MergeResponse struct {
	Body []Domain
}

// JSON is a wrapper type for storing JSON in SQLite as TEXT.
type JSON[T any] struct {
	Data T
}

// Value implements driver.Valuer for database storage.
func (j JSON[T]) Value() (driver.Value, error) {
	return json.Marshal(j.Data)
}

// Scan implements sql.Scanner for database retrieval.
func (j *JSON[T]) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}

	return json.Unmarshal(bytes, &j.Data)
}

// HistoryEntry represents a merge operation history record.
type HistoryEntry struct {
	ID        int64                     `json:"id" doc:"Unique identifier" example:"1"`
	CreatedAt time.Time                 `json:"created_at" doc:"Timestamp when merge was performed" format:"date-time"`
	Initial   JSON[[]Domain]            `json:"initial" doc:"Original domain configurations before merge"`
	Response  JSON[CertificateResponse] `json:"response" doc:"Certificate response data used for merge"`
	Result    JSON[[]Domain]            `json:"result" doc:"Final merged domain configurations with certificates"`
}

// NSXConfig represents a saved NSX configuration.
type NSXConfig struct {
	ID          int64     `json:"id,omitempty" doc:"Unique identifier" example:"1"`
	Name        string    `json:"name" doc:"Configuration name" minLength:"1" maxLength:"255" example:"production-nsx"`
	Description string    `json:"description,omitempty" doc:"Human-readable configuration description" example:"Production NSX Manager"`
	Host        string    `json:"host" doc:"NSX Manager URL" format:"uri" example:"https://nsx.example.com"`
	Username    string    `json:"username" doc:"NSX API username" example:"admin"`
	Password    string    `json:"password,omitempty" doc:"NSX API password (write-only, never returned in responses)"`
	Insecure    bool      `json:"insecure" doc:"Skip TLS certificate verification" example:"false"`
	CreatedAt   time.Time `json:"created_at,omitempty" doc:"Creation timestamp" format:"date-time"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" doc:"Last update timestamp" format:"date-time"`
}
