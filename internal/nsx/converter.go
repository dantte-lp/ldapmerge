package nsx

import (
	"strconv"

	"ldapmerge/internal/models"
)

// DomainToLDAPIdentitySource converts internal Domain model to NSX LDAPIdentitySource
func DomainToLDAPIdentitySource(d models.Domain) LDAPIdentitySource {
	servers := make([]LDAPServer, len(d.LDAPServers))
	for i, s := range d.LDAPServers {
		enabled, _ := strconv.ParseBool(s.Enabled)
		startTLS, _ := strconv.ParseBool(s.StartTLS)

		servers[i] = LDAPServer{
			URL:          s.URL,
			UseStartTLS:  startTLS,
			Enabled:      enabled,
			BindIdentity: s.BindUsername,
			Password:     s.BindPassword,
			Certificates: s.Certificates,
		}
	}

	return LDAPIdentitySource{
		ID:                     d.ID,
		DisplayName:            d.DomainName,
		DomainName:             d.DomainName,
		BaseDN:                 d.BaseDN,
		AlternativeDomainNames: d.AlternativeDomainNames,
		LDAPServers:            servers,
		ResourceType:           "LdapIdentitySource",
	}
}

// LDAPIdentitySourceToDomain converts NSX LDAPIdentitySource to internal Domain model
func LDAPIdentitySourceToDomain(s LDAPIdentitySource) models.Domain {
	servers := make([]models.LDAPServer, len(s.LDAPServers))
	for i, srv := range s.LDAPServers {
		servers[i] = models.LDAPServer{
			URL:          srv.URL,
			StartTLS:     strconv.FormatBool(srv.UseStartTLS),
			Enabled:      strconv.FormatBool(srv.Enabled),
			BindUsername: srv.BindIdentity,
			BindPassword: srv.Password,
			Certificates: srv.Certificates,
		}
	}

	return models.Domain{
		ID:                     s.ID,
		DomainName:             s.DomainName,
		BaseDN:                 s.BaseDN,
		AlternativeDomainNames: s.AlternativeDomainNames,
		LDAPServers:            servers,
	}
}

// DomainsToLDAPIdentitySources converts slice of Domains to LDAPIdentitySources
func DomainsToLDAPIdentitySources(domains []models.Domain) []LDAPIdentitySource {
	result := make([]LDAPIdentitySource, len(domains))
	for i, d := range domains {
		result[i] = DomainToLDAPIdentitySource(d)
	}
	return result
}

// LDAPIdentitySourcesToDomains converts slice of LDAPIdentitySources to Domains
func LDAPIdentitySourcesToDomains(sources []LDAPIdentitySource) []models.Domain {
	result := make([]models.Domain, len(sources))
	for i, s := range sources {
		result[i] = LDAPIdentitySourceToDomain(s)
	}
	return result
}
