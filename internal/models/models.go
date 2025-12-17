package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// LDAPServer represents an LDAP server configuration
type LDAPServer struct {
	URL          string   `json:"url"`
	StartTLS     string   `json:"starttls"`
	Enabled      string   `json:"enabled"`
	BindUsername string   `json:"bind_username,omitempty"`
	BindPassword string   `json:"bind_password,omitempty"`
	Certificates []string `json:"certificates,omitempty"`
}

// Domain represents a domain configuration with LDAP servers
type Domain struct {
	ID                     string       `json:"id"`
	DomainName             string       `json:"domain_name"`
	BaseDN                 string       `json:"base_dn"`
	AlternativeDomainNames []string     `json:"alternative_domain_names"`
	LDAPServers            []LDAPServer `json:"ldap_servers"`
}

// CertificateDetail contains certificate subject info
type CertificateDetail struct {
	SubjectCN string `json:"subject_cn"`
}

// CertificateJSON contains the certificate data from response
type CertificateJSON struct {
	PEMEncoded string              `json:"pem_encoded"`
	Details    []CertificateDetail `json:"details"`
}

// ResponseItem represents the item from response (matching LDAP server)
type ResponseItem struct {
	URL      string `json:"url"`
	StartTLS string `json:"starttls"`
	Enabled  string `json:"enabled"`
}

// CertificateResult represents a single result from the response JSON
type CertificateResult struct {
	JSON           CertificateJSON `json:"json"`
	Item           ResponseItem    `json:"item"`
	AnsibleLoopVar string          `json:"ansible_loop_var"`
}

// CertificateResponse represents the full response JSON structure
type CertificateResponse struct {
	Results []CertificateResult `json:"results"`
}

// MergeRequest is the API request for merging operation
type MergeRequest struct {
	Initial  []Domain            `json:"initial"`
	Response CertificateResponse `json:"response"`
}

// MergeResponse is the API response with merged data
type MergeResponse struct {
	Body []Domain
}

// JSON is a wrapper type for storing JSON in SQLite as TEXT
type JSON[T any] struct {
	Data T
}

// Value implements driver.Valuer for database storage
func (j JSON[T]) Value() (driver.Value, error) {
	return json.Marshal(j.Data)
}

// Scan implements sql.Scanner for database retrieval
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

// HistoryEntry represents a merge operation history record
type HistoryEntry struct {
	ID        int64               `json:"id" doc:"Unique identifier"`
	CreatedAt time.Time           `json:"created_at" doc:"Creation timestamp"`
	Initial   JSON[[]Domain]      `json:"initial" doc:"Initial input data"`
	Response  JSON[CertificateResponse] `json:"response" doc:"Certificate response data"`
	Result    JSON[[]Domain]      `json:"result" doc:"Merged result data"`
}

// NSXConfig represents a saved NSX configuration
type NSXConfig struct {
	ID          int64     `json:"id,omitempty" doc:"Unique identifier"`
	Name        string    `json:"name" doc:"Configuration name" minLength:"1" maxLength:"255"`
	Description string    `json:"description,omitempty" doc:"Configuration description"`
	Host        string    `json:"host" doc:"NSX Manager host URL" format:"uri"`
	Username    string    `json:"username" doc:"NSX API username"`
	Password    string    `json:"password,omitempty" doc:"NSX API password (write-only)"`
	Insecure    bool      `json:"insecure" doc:"Skip TLS verification"`
	CreatedAt   time.Time `json:"created_at,omitempty" doc:"Creation timestamp"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" doc:"Last update timestamp"`
}
