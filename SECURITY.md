# Security Policy

## Supported Scope

Security reports are accepted for the current `develop` branch and the latest published release candidate. Skoll is still in open-source foundation construction, so reports should target current contracts rather than legacy compatibility paths.

## Reporting a Vulnerability

Do not open a public issue for a suspected vulnerability. Send a private report to the maintainers with:

- affected commit or version;
- impacted surface, such as auth, RBAC, plugin install, file storage, audit export, or deployment config;
- reproduction steps;
- expected impact;
- any logs, requests, responses, or proof-of-concept details that are safe to share.

If no private security address is configured for the repository yet, open a public issue with the title `Security contact request` and no exploit details. A maintainer will arrange a private channel.

## Response Expectations

Maintainers should acknowledge reports within 5 business days, triage severity, and provide a remediation plan or status update. Critical auth, permission bypass, secret exposure, arbitrary file access, plugin execution, and data exfiltration issues take priority over feature work.

## Safe Harbor

Good-faith research is welcome when it avoids data destruction, persistence, lateral movement, social engineering, and public disclosure before maintainers have had time to respond.
