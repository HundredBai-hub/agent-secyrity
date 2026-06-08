package auth

import (
	"net/http"
	"testing"
)

func TestParseAPIKeys(t *testing.T) {
	t.Parallel()

	config, err := ParseAPIKeys("runtime:runtime-secret:tenant-a,tenant-b;admin:admin-secret:*")
	if err != nil {
		t.Fatalf("ParseAPIKeys() error = %v", err)
	}
	if !config.Enabled() {
		t.Fatalf("Enabled() = false, want true")
	}

	runtimeKey, ok := config.Authenticate("runtime-secret")
	if !ok {
		t.Fatalf("Authenticate(runtime-secret) failed")
	}
	if runtimeKey.ID != "runtime" {
		t.Fatalf("key ID = %q, want runtime", runtimeKey.ID)
	}
	if !runtimeKey.AllowsTenant("tenant-a") {
		t.Fatalf("runtime key should allow tenant-a")
	}
	if runtimeKey.AllowsTenant("tenant-c") {
		t.Fatalf("runtime key should not allow tenant-c")
	}

	adminKey, ok := config.Authenticate("admin-secret")
	if !ok {
		t.Fatalf("Authenticate(admin-secret) failed")
	}
	if !adminKey.AllowsTenant("any-tenant") {
		t.Fatalf("admin wildcard key should allow any tenant")
	}
}

func TestParseAPIKeysRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []string{
		"missing-parts",
		"id::tenant-a",
		"id:secret:",
		"id:secret:tenant-a;other:secret:tenant-b",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseAPIKeys(input); err == nil {
				t.Fatalf("ParseAPIKeys(%q) error = nil, want error", input)
			}
		})
	}
}

func TestEmptyConfigDisablesAuth(t *testing.T) {
	t.Parallel()

	config, err := ParseAPIKeys("")
	if err != nil {
		t.Fatalf("ParseAPIKeys(empty) error = %v", err)
	}
	if config.Enabled() {
		t.Fatalf("Enabled() = true, want false")
	}
}

func TestAuthenticateRequest(t *testing.T) {
	t.Parallel()

	config, err := ParseAPIKeys("runtime:runtime-secret:tenant-a")
	if err != nil {
		t.Fatalf("ParseAPIKeys() error = %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.test/v1/evaluate", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer runtime-secret")

	key, err := config.AuthenticateRequest(req)
	if err != nil {
		t.Fatalf("AuthenticateRequest() error = %v", err)
	}
	if key.ID != "runtime" {
		t.Fatalf("key ID = %q, want runtime", key.ID)
	}
}

func TestAuthenticateRequestRejectsMissingOrInvalidBearer(t *testing.T) {
	t.Parallel()

	config, err := ParseAPIKeys("runtime:runtime-secret:tenant-a")
	if err != nil {
		t.Fatalf("ParseAPIKeys() error = %v", err)
	}

	tests := map[string]string{
		"missing": "",
		"basic":   "Basic runtime-secret",
		"wrong":   "Bearer wrong-secret",
	}
	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(http.MethodGet, "http://example.test/v1/evaluate", nil)
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			if _, err := config.AuthenticateRequest(req); err == nil {
				t.Fatalf("AuthenticateRequest() error = nil, want error")
			}
		})
	}
}
