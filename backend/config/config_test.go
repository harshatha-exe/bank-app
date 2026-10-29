package config

import "testing"

func TestLoadTrustedProxies(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/bank")
	t.Setenv("CSRF_SECRET", "secret")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 192.0.2.10")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.TrustedProxies) != 2 {
		t.Fatalf("trusted proxies = %#v", config.TrustedProxies)
	}
}

func TestLoadRejectsUnrestrictedTrustedProxy(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/bank")
	t.Setenv("CSRF_SECRET", "secret")
	t.Setenv("TRUSTED_PROXIES", "0.0.0.0/0")
	if _, err := Load(); err == nil {
		t.Fatal("expected unrestricted proxy to be rejected")
	}
}

func TestLoadRejectsInvalidTrustedProxy(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/bank")
	t.Setenv("CSRF_SECRET", "secret")
	t.Setenv("TRUSTED_PROXIES", "not-an-ip")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid proxy to be rejected")
	}
}

func TestLoadSwaggerEnabledDefaultsAndCanBeDisabled(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/bank")
	t.Setenv("CSRF_SECRET", "secret")
	t.Setenv("TRUSTED_PROXIES", "")
	t.Setenv("SWAGGER_ENABLED", "")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !config.SwaggerEnabled {
		t.Fatal("swagger should be enabled by default")
	}
	t.Setenv("SWAGGER_ENABLED", "false")
	config, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.SwaggerEnabled {
		t.Fatal("swagger should be disabled explicitly")
	}
}
