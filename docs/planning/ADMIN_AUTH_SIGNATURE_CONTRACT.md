# Admin Auth Signature Contract

## Scope

This document defines the request-signing contract for `admin-auth-mode=hmac-sha256`.

## Headers

Required headers on protected admin requests:

- `X-Admin-Timestamp`: Unix timestamp in seconds.
- `X-Admin-Nonce`: Client-generated nonce, unique within replay window.
- `X-Admin-Signature`: Hex-encoded HMAC-SHA256 signature.
- `X-Admin-Body-SHA256`: Optional lowercase hex SHA256 of request body.

## Signature Payload

Payload string format:

```text
{HTTP_METHOD}\n{REQUEST_PATH}\n{TIMESTAMP}\n{NONCE}\n{BODY_SHA256}
```

Example for `GET /admin/ping` and timestamp `1710000000`:

```text
GET
/admin/ping
1710000000
nonce-abc123

```

## Signature Algorithm

- HMAC algorithm: `HMAC-SHA256`
- Input key: value of `SKOLL_ADMIN_AUTH_HMAC_SECRET`
- Output encoding: lowercase hex string

## Time Window Validation

Server accepts timestamps within ±5 minutes of server time.

- Too old or too far in the future -> `401 Unauthorized`
- Missing/invalid timestamp or signature format -> `401 Unauthorized`
- Reused nonce in active window -> `401 Unauthorized`
- Provided body hash does not match actual request body -> `401 Unauthorized`

## Client Example (Shell + OpenSSL)

```bash
TS=$(date +%s)
PAYLOAD="GET\n/admin/ping\n${TS}"
NONCE="nonce-$(openssl rand -hex 8)"
PAYLOAD="GET\n/admin/ping\n${TS}\n${NONCE}\n"
SIG=$(printf "%b" "$PAYLOAD" | openssl dgst -sha256 -hmac "secret" -binary | xxd -p -c 256)

curl \
  -H "X-Admin-Timestamp: ${TS}" \
	-H "X-Admin-Nonce: ${NONCE}" \
  -H "X-Admin-Signature: ${SIG}" \
  http://localhost:8080/admin/ping
```

## Client Example (Go)

```go
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func sign(method, path, ts, nonce, bodySHA, secret string) string {
	payload := method + "\n" + path + "\n" + ts + "\n" + nonce + "\n" + bodySHA
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func main() {
	ts := "1710000000"
	nonce := "nonce-abc123"
	fmt.Println(sign("GET", "/admin/ping", ts, nonce, "", "secret"))
}
```

## Operational Notes

- Keep server/client clocks synchronized (NTP recommended).
- Rotate HMAC secret periodically.
- Use HTTPS to protect headers in transit.
- Avoid nonce reuse inside the replay window.
- Default replay store is process-local in-memory; multi-instance deployments should use a shared nonce store implementation.
