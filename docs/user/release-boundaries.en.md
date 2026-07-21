# Release, Regulatory, and Support Boundaries

English | [简体中文](release-boundaries.md)

> Status: canonical English counterpart to the Chinese-default release and operator boundary. This document is not legal, regulatory, or professional advice.

## Release Position

Skoll is an open-source administration framework under active development. `pharma_oa` is an industry sample demonstrating plugin, permission, audit, workflow, master-data, and inventory capabilities. Acceptance evidence proves the recorded software contracts in the recorded environments; it is not certification of any production deployment, organizational process, or regulatory compliance.

## Pharmaceutical And Regulatory Boundary

- `pharma_oa` is not a regulator-validated GSP, GMP, medical-device, clinical, or pharmaceutical traceability system and provides no regulatory certification.
- Labels such as compliance, qualification, recall, cold chain, and audit describe sample capabilities only. They do not assert conformity with any jurisdiction, company policy, or regulator.
- The project makes no commitment to provide medical advice, clinical decisions, drug-quality determinations, legally binding record signatures, or regulatory submissions.
- Adopters must identify applicable law, industry rules, data residency, electronic record, validation, retention, and audit requirements. Their legal, quality, security, and business owners must approve production use.
- Do not place the sample in a regulated production process before requirements, risk assessment, computerized-system validation, access review, backup/restore rehearsal, and operating procedures are complete.

## Demo And Data Safety

Built-in `DEMO-*` identifiers, names, contacts, organizations, medicines, qualifications, and transactions are fictional test data. They do not represent real people, patients, customers, suppliers, drug approvals, or regulatory records. Do not apply the demo seed to production. Do not submit real personal data, health data, trade secrets, or production credentials to the repository, issues, logs, screenshots, or acceptance evidence.

| Responsibility | Operator requirement |
| --- | --- |
| Persistence | `mysql` / `postgres` persist business data; `memory` is for development acceptance and loses data when the process exits |
| Minimization | Collect only necessary fields and establish lawful access, authorization, and deletion processes |
| Permission and audit | Apply least privilege and review data scopes and critical audit events regularly |
| Encryption and secrets | Use TLS in transit and organizational encryption policies for databases, backups, and object storage; keep secrets in dedicated secret management |
| Retention and deletion | Define retention and deletion for database records, logs, audits, exports, and objects |
| Backup and recovery | Back up the database, `/app/data`, `/app/plugins`, and external object storage separately; verify isolated restore before release |

See [Deployment, Upgrade, and Recovery](deployment.md) for release commands and the [Operations Guide](operations.md) for routine checks.

## Security And Secrets

- Production must replace the default administrator password and `SKOLL_SECURITY_JWT_SECRET`. Database, plugin, and external-service credentials must not be stored in source, images, plaintext Compose/Kubernetes manifests, or release evidence.
- Pin dependency and image versions, review plugin source, signatures, permissions, menus, routes, and migration risks, and verify disable and failure states before release.
- Do not report exploit details publicly. Follow the repository [Security Policy](../../SECURITY.md) to request a private reporting channel.
- Project maintainers do not hold adopter production secrets and cannot replace the operator's incident response, forensics, or data recovery.

## License Boundary

- Skoll source code is provided under the repository [MIT License](../../LICENSE). Copies or substantial portions must retain the copyright and permission notice.
- The MIT License provides the software as-is without merchantability, fitness, or non-infringement warranties. `LICENSE` is authoritative.
- Go, Node.js, container base images, frontend libraries, and other third-party components retain their own licenses. `go.mod`, `go.sum`, `web/package.json`, and lockfiles are dependency inventories, not project relicensing of those components.
- A distributor must generate and review an SBOM and third-party license report for the actual build and satisfy applicable attribution, notice, source-offer, or other obligations.
- Unless a file says otherwise, repository sample plugins, scripts, and fictional demo data are distributed with the project under MIT. This grants no rights to third-party trademarks, real medicines, real organizations, or regulatory certification.

## Support Boundary

Community issues, discussions, and contribution review are handled on a best-effort maintainer-availability basis. There is no commercial support, response-time, availability, data-recovery, regulatory-validation, or security-on-call SLA. Reports should include version, environment, minimal reproduction, and redacted logs. General issues must not contain exploit details or real sensitive data. Security reports follow `SECURITY.md`; community behavior follows [CODE_OF_CONDUCT.md](../../CODE_OF_CONDUCT.md).

Production operators own capacity planning, monitoring, on-call coverage, upgrade windows, rollback decisions, disaster recovery, and vendor support. Contractual service needs should be arranged separately with a provider qualified to accept those responsibilities.

## Pre-Release Confirmation

1. Execute the [Release Checklist](../release-checklist.md) and review the [Deployment Rehearsal](../refactor/current/deployment_recovery_rehearsal_2026-07-21.md).
2. Confirm no default credentials or demo seed are enabled in the target environment and that real data classification and authorization reviews are complete.
3. Record licenses/SBOM, image digest, configuration, secret references, backup, restore point, rollback owner, and accepted risks.
4. Confirm product copy and contracts do not describe the sample as certified, a compliance guarantee, or a medical decision system.
5. Do not add legacy API, data, plugin-format, or page-route compatibility layers. Upgrade through current contracts and restore points.
