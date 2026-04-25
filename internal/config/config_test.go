package config

import (
	"encoding/json"
	"testing"
)

// --- Load: single-account env vars ---

func TestLoad_SingleAccount(t *testing.T) {
	t.Setenv("DAV_URL", "https://dav.example.com")
	t.Setenv("DAV_USERNAME", "alice")
	t.Setenv("DAV_PASSWORD", "secret")
	t.Setenv("DAV_ACCOUNTS", "")
	t.Setenv("DAV_DEBUG", "")

	cfg := Load()
	if len(cfg.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(cfg.Accounts))
	}
	a := cfg.Accounts[0]
	if a.Name != "default" {
		t.Errorf("Name=%q", a.Name)
	}
	if a.URL != "https://dav.example.com" {
		t.Errorf("URL=%q", a.URL)
	}
	if a.Username != "alice" {
		t.Errorf("Username=%q", a.Username)
	}
	if cfg.Debug {
		t.Error("Debug should be false")
	}
}

func TestLoad_Debug(t *testing.T) {
	t.Setenv("DAV_DEBUG", "1")
	t.Setenv("DAV_URL", "https://dav.example.com")
	t.Setenv("DAV_ACCOUNTS", "")

	cfg := Load()
	if !cfg.Debug {
		t.Error("expected Debug=true")
	}
}

func TestLoad_NoEnv_EmptyAccounts(t *testing.T) {
	t.Setenv("DAV_URL", "")
	t.Setenv("DAV_ACCOUNTS", "")
	t.Setenv("DAV_DEBUG", "")

	cfg := Load()
	if len(cfg.Accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(cfg.Accounts))
	}
}

func TestLoad_MultiAccount(t *testing.T) {
	accounts := []Account{
		{Name: "work", URL: "https://work.example.com", Username: "bob", Password: "pw1"},
		{Name: "home", URL: "https://home.example.com", Username: "bob", Password: "pw2"},
	}
	raw, _ := json.Marshal(accounts)
	t.Setenv("DAV_ACCOUNTS", string(raw))
	t.Setenv("DAV_DEBUG", "")

	cfg := Load()
	if len(cfg.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(cfg.Accounts))
	}
	if cfg.Accounts[0].Name != "work" || cfg.Accounts[1].Name != "home" {
		t.Errorf("unexpected names: %v", cfg.AccountNames())
	}
}

func TestLoad_MultiAccount_AutoName(t *testing.T) {
	accounts := []Account{
		{URL: "https://a.example.com", Username: "u", Password: "p"},
		{URL: "https://b.example.com", Username: "u", Password: "p"},
	}
	raw, _ := json.Marshal(accounts)
	t.Setenv("DAV_ACCOUNTS", string(raw))
	t.Setenv("DAV_DEBUG", "")

	cfg := Load()
	if cfg.Accounts[0].Name != "account1" {
		t.Errorf("auto name[0]=%q", cfg.Accounts[0].Name)
	}
	if cfg.Accounts[1].Name != "account2" {
		t.Errorf("auto name[1]=%q", cfg.Accounts[1].Name)
	}
}

// --- Primary ---

func TestPrimary_PrefersDefault(t *testing.T) {
	cfg := Config{
		Accounts: []Account{
			{Name: "work", URL: "https://work.example.com"},
			{Name: "default", URL: "https://default.example.com"},
		},
	}
	p := cfg.Primary()
	if p.Name != "default" {
		t.Errorf("Primary=%q, want default", p.Name)
	}
}

func TestPrimary_FallsBackToFirst(t *testing.T) {
	cfg := Config{
		Accounts: []Account{
			{Name: "work", URL: "https://work.example.com"},
			{Name: "home", URL: "https://home.example.com"},
		},
	}
	p := cfg.Primary()
	if p.Name != "work" {
		t.Errorf("Primary=%q, want work", p.Name)
	}
}

func TestPrimary_EmptyAccounts(t *testing.T) {
	cfg := Config{}
	p := cfg.Primary()
	if p.Name != "" || p.URL != "" {
		t.Errorf("expected zero Account, got %+v", p)
	}
}

// --- Account ---

func TestAccount_ByName(t *testing.T) {
	cfg := Config{
		Accounts: []Account{
			{Name: "work", URL: "https://work.example.com"},
			{Name: "home", URL: "https://home.example.com"},
		},
	}
	a, err := cfg.Account("home")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.URL != "https://home.example.com" {
		t.Errorf("URL=%q", a.URL)
	}
}

func TestAccount_NotFound(t *testing.T) {
	cfg := Config{
		Accounts: []Account{{Name: "work", URL: "https://work.example.com"}},
	}
	_, err := cfg.Account("missing")
	if err == nil {
		t.Error("expected error for unknown account")
	}
}

func TestAccount_EmptyName_ReturnsPrimary(t *testing.T) {
	cfg := Config{
		Accounts: []Account{{Name: "only", URL: "https://only.example.com"}},
	}
	a, err := cfg.Account("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name != "only" {
		t.Errorf("Name=%q", a.Name)
	}
}

// --- AccountNames ---

func TestAccountNames(t *testing.T) {
	cfg := Config{
		Accounts: []Account{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}
	names := cfg.AccountNames()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("AccountNames=%v", names)
	}
}
