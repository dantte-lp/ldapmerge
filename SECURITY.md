# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within ldapmerge, please report it responsibly.

### How to Report

1. **Do NOT** open a public GitHub issue for security vulnerabilities
2. Send details to the project maintainer via private channels
3. Include as much information as possible:
   - Type of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- Acknowledgment within 48 hours
- Status update within 7 days
- Fix timeline based on severity

### Security Best Practices

When using ldapmerge:

1. **Credentials**: Never commit NSX credentials to version control
2. **TLS**: Use `--insecure` flag only in development environments
3. **Database**: Protect the SQLite database file with appropriate permissions
4. **API Server**: Run behind a reverse proxy with TLS in production
5. **Logs**: Review log files for sensitive information before sharing

## Vulnerability Disclosure

We follow responsible disclosure practices:

1. Reporter notifies maintainer privately
2. Maintainer confirms and assesses the vulnerability
3. Fix is developed and tested
4. Security advisory is prepared
5. Fix is released with advisory
6. Public disclosure after patch availability
