# Security policy

## Supported versions

Security fixes are applied to the latest released minor version. This provider is pre-1.0; users should pin a version and upgrade deliberately after reviewing the changelog and Terraform plan.

## Reporting a vulnerability

Do not open a public issue for vulnerabilities, leaked credentials, customer identifiers, or account data.

Use GitHub's **Security → Report a vulnerability** flow to open a private security advisory. If private reporting is unavailable, contact the maintainer through the contact method on the repository owner's GitHub profile and include only enough information to establish a private channel. Do not send live Google Ads credentials in the first message.

Include, when applicable:

- the affected provider version and platform;
- the smallest redacted configuration that reproduces the issue;
- whether the issue can expose credentials, account data, or cause destructive mutations;
- suggested remediation or a proof of concept that uses test accounts only.

The maintainer will acknowledge reports on a best-effort basis. No response-time or remediation SLA is offered.

## Credential safety

- Supply credentials through environment variables or a secrets manager, not committed `.tf`, `.tfvars`, state, logs, fixtures, or issue attachments.
- Treat OAuth client secrets, refresh tokens, service-account JSON, Terraform state, and generated account exports as sensitive.
- Redact customer IDs, resource names, request IDs, URLs, and ad content before posting diagnostics publicly.
- If a credential is committed, rotate it first. Removing it from the latest commit does not remove it from Git history.

