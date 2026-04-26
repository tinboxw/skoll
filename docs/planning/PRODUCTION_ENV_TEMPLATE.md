# Production Environment Template

## Scope

This document provides a production-ready baseline environment template for Skoll deployments.

## Recommended Baseline

Use `.env.production.example` as the source template and apply values through your secret manager or deployment system.

Required baseline controls:

- `SKOLL_GO_ADMIN_MODE=prod`
- `SKOLL_ADMIN_AUTH_MODE=hmac-sha256`
- `SKOLL_ADMIN_AUTH_NONCE_STORE=redis`
- `SKOLL_ADMIN_AUTH_ALLOW_STATIC_TOKEN_IN_PROD=false`

## Template Source

- Root template file: `.env.production.example`

## Operator Checklist

1. Replace placeholder secrets before deployment.
2. Keep HMAC secret only in secret manager.
3. Ensure Redis nonce store endpoint is highly available.
4. Verify startup logs include expected runtime config values.
5. Run signed `/admin/ping` verification after rollout.
