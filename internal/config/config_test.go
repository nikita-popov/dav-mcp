package config

import (
	"os"
	"testing"
)

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("DAV_URL", "https://dav.example.com")
	os.Setenv("DAV_USERNAME", "alice")
	os.Setenv("DAV_PASSWORD", "secret")
	defer func() {
		os.Unsetenv("DAV_URL")
		os.Unsetenv("DAV_USERNAME")
		os.Unsetenv("DAV_PASSWORD")
	}()

	c := Load()
	if c.DAVURL != "https://dav.example.com" {
		t.Errorf("DAVURL: got %q", c.DAVURL)
	}
	if c.Username != "alice" {
		t.Errorf("Username: got %q", c.Username)
	}
	if c.Password != "secret" {
		t.Errorf("Password: got %q", c.Password)
	}
}

func TestLoad_EmptyEnv(t *testing.T) {
	os.Unsetenv("DAV_URL")
	os.Unsetenv("DAV_USERNAME")
	os.Unsetenv("DAV_PASSWORD")
	c := Load()
	if c.DAVURL != "" || c.Username != "" || c.Password != "" {
		t.Errorf("expected empty config, got %+v", c)
	}
}
