// Package auth provides API authentication and tenant authorization helpers.
package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
)

// ErrUnauthorized is returned when a request has no valid API key.
var ErrUnauthorized = fmt.Errorf("missing or invalid API key")

// APIKey represents one configured API key and its tenant access scope.
type APIKey struct {
	ID              string
	secret          string
	allowedTenants  map[string]struct{}
	allowAllTenants bool
}

// AllowsTenant reports whether this API key can access a tenant.
func (k APIKey) AllowsTenant(tenantID string) bool {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return false
	}
	if k.allowAllTenants {
		return true
	}
	_, ok := k.allowedTenants[tenantID]
	return ok
}

// APIKeyConfig contains all configured API keys.
type APIKeyConfig struct {
	keys []APIKey
}

// ParseAPIKeys parses AGENT_SECURITY_API_KEYS.
func ParseAPIKeys(raw string) (APIKeyConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return APIKeyConfig{}, nil
	}

	seenSecrets := make(map[string]struct{})
	entries := strings.Split(raw, ";")
	keys := make([]APIKey, 0, len(entries))
	for _, entry := range entries {
		key, err := parseAPIKey(entry)
		if err != nil {
			return APIKeyConfig{}, err
		}
		if _, ok := seenSecrets[key.secret]; ok {
			return APIKeyConfig{}, fmt.Errorf("duplicate API key secret")
		}
		seenSecrets[key.secret] = struct{}{}
		keys = append(keys, key)
	}
	return APIKeyConfig{keys: keys}, nil
}

// Enabled reports whether API key authentication should be enforced.
func (c APIKeyConfig) Enabled() bool {
	return len(c.keys) > 0
}

// Authenticate returns the matching API key for a bearer token.
func (c APIKeyConfig) Authenticate(token string) (APIKey, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return APIKey{}, false
	}
	for _, key := range c.keys {
		if constantTimeStringEqual(key.secret, token) {
			return key, true
		}
	}
	return APIKey{}, false
}

// AuthenticateRequest authenticates a request using Authorization: Bearer.
func (c APIKeyConfig) AuthenticateRequest(r *http.Request) (APIKey, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return APIKey{}, ErrUnauthorized
	}
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return APIKey{}, ErrUnauthorized
	}
	key, ok := c.Authenticate(token)
	if !ok {
		return APIKey{}, ErrUnauthorized
	}
	return key, nil
}

func parseAPIKey(entry string) (APIKey, error) {
	parts := strings.Split(strings.TrimSpace(entry), ":")
	if len(parts) != 3 {
		return APIKey{}, fmt.Errorf("API key entry must use key-id:secret:tenant-list format")
	}
	id := strings.TrimSpace(parts[0])
	secret := strings.TrimSpace(parts[1])
	tenantList := strings.TrimSpace(parts[2])
	if id == "" {
		return APIKey{}, fmt.Errorf("API key id is required")
	}
	if secret == "" {
		return APIKey{}, fmt.Errorf("API key secret is required")
	}
	if tenantList == "" {
		return APIKey{}, fmt.Errorf("API key tenant list is required")
	}

	key := APIKey{
		ID:             id,
		secret:         secret,
		allowedTenants: make(map[string]struct{}),
	}
	for _, tenant := range strings.Split(tenantList, ",") {
		tenant = strings.TrimSpace(tenant)
		if tenant == "" {
			return APIKey{}, fmt.Errorf("API key tenant id is required")
		}
		if tenant == "*" {
			key.allowAllTenants = true
			continue
		}
		key.allowedTenants[tenant] = struct{}{}
	}
	if !key.allowAllTenants && len(key.allowedTenants) == 0 {
		return APIKey{}, fmt.Errorf("API key must allow at least one tenant")
	}
	return key, nil
}

func constantTimeStringEqual(expected string, actual string) bool {
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
