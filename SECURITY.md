# Security Policy

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's private vulnerability
reporting on the upstream repository:
<https://github.com/kentik/terraform-provider-kentik-cloudexport/security/advisories/new>

Do not open a public issue or pull request for a suspected vulnerability.
Include the provider version, Terraform/OpenTofu version, and the steps or
configuration needed to reproduce. Never include real Kentik API tokens.

## Supported versions

| Version | Supported |
|---------|-----------|
| 0.5.x (next release, `modernize/2026-11`) | Yes |
| 0.4.x and older | No: built with Go 1.17 and dependencies with known vulnerabilities. Upgrade to 0.5.x. |

## Dependency scanning

`make vuln` runs `govulncheck ./...`; CI runs it on every push and pull
request and Dependabot opens weekly grouped updates. There are currently no
known vulnerabilities that affect this module.
