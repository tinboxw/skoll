package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type JWTClaims struct {
	Subject          string   `json:"sub"`
	OrganizationID   string   `json:"organizationId,omitempty"`
	OrganizationPath []string `json:"organizationPath,omitempty"`
	Role             string   `json:"role"`
	Roles            []string `json:"roles"`
	IssuedAt         int64    `json:"iat"`
	ExpiresAt        int64    `json:"exp"`
}

type JWTIdentity struct {
	Subject          string
	OrganizationID   string
	OrganizationPath []string
	Role             string
	Roles            []string
}

func (c JWTClaims) HasRole(role string) bool {
	target := strings.TrimSpace(role)
	if target == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(c.Role), target) {
		return true
	}
	for _, item := range c.Roles {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func SignJWT(secret string, identity JWTIdentity, ttl time.Duration, now time.Time) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is empty")
	}
	if ttl <= 0 {
		return "", errors.New("jwt ttl must be > 0")
	}
	claims, err := buildJWTClaims(identity, ttl, now)
	if err != nil {
		return "", err
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}

	headerRaw, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	head := base64.RawURLEncoding.EncodeToString(headerRaw)
	body := base64.RawURLEncoding.EncodeToString(claimsRaw)
	message := head + "." + body

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return message + "." + sig, nil
}

func ParseJWT(secret, tokenString string) (*JWTClaims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}
	message := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, errors.New("invalid token signature")
	}

	bodyRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode token payload: %w", err)
	}
	var claims JWTClaims
	if err := json.Unmarshal(bodyRaw, &claims); err != nil {
		return nil, fmt.Errorf("parse token payload: %w", err)
	}
	if err := validateJWTClaims(claims); err != nil {
		return nil, err
	}
	if claims.ExpiresAt > 0 && time.Now().Unix() >= claims.ExpiresAt {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func buildJWTClaims(identity JWTIdentity, ttl time.Duration, now time.Time) (JWTClaims, error) {
	claims := JWTClaims{
		Subject:          strings.TrimSpace(identity.Subject),
		OrganizationID:   strings.TrimSpace(identity.OrganizationID),
		OrganizationPath: normalizeClaimValues(identity.OrganizationPath),
		Role:             strings.TrimSpace(identity.Role),
		Roles:            normalizeClaimValues(identity.Roles),
		IssuedAt:         now.Unix(),
		ExpiresAt:        now.Add(ttl).Unix(),
	}
	if claims.Role == "" && len(claims.Roles) > 0 {
		claims.Role = claims.Roles[0]
	}
	if err := validateJWTClaims(claims); err != nil {
		return JWTClaims{}, err
	}
	return claims, nil
}

func validateJWTClaims(claims JWTClaims) error {
	if strings.TrimSpace(claims.Subject) == "" {
		return errors.New("jwt subject is empty")
	}
	if claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return errors.New("jwt time claims are invalid")
	}
	if len(claims.Roles) == 0 {
		return errors.New("jwt roles are empty")
	}
	if !containsClaimValue(claims.Roles, claims.Role) {
		return errors.New("jwt primary role is not present in roles")
	}
	if claims.OrganizationID == "" {
		if len(claims.OrganizationPath) != 0 {
			return errors.New("jwt organization path requires an organization")
		}
		return nil
	}
	if len(claims.OrganizationPath) == 0 || claims.OrganizationPath[len(claims.OrganizationPath)-1] != claims.OrganizationID {
		return errors.New("jwt organization path does not end at organization")
	}
	return nil
}

func normalizeClaimValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func containsClaimValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
