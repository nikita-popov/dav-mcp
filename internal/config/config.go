package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Account holds credentials for a single DAV server.
type Account struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Config is the top-level application configuration.
type Config struct {
	// Accounts is the list of configured DAV accounts.
	// Always has at least one entry after Load() succeeds.
	Accounts []Account

	// Debug enables verbose HTTP logging when true.
	Debug bool

	// Deprecated single-account fields kept for internal use only.
	// Use Accounts[0] for the primary account.
	DAVURL   string
	Username string
	Password string
}

// Primary returns the default account.
// Prefers an account named "default"; falls back to the first in the list.
func (c Config) Primary() Account {
	for _, a := range c.Accounts {
		if a.Name == "default" {
			return a
		}
	}
	if len(c.Accounts) == 0 {
		return Account{}
	}
	return c.Accounts[0]
}

// Account returns the named account, or an error if not found.
// An empty name returns the primary account.
func (c Config) Account(name string) (Account, error) {
	if name == "" {
		return c.Primary(), nil
	}
	for _, a := range c.Accounts {
		if a.Name == name {
			return a, nil
		}
	}
	return Account{}, fmt.Errorf("account %q not found; available: %v", name, c.AccountNames())
}

// AccountNames returns a slice of all configured account names.
func (c Config) AccountNames() []string {
	names := make([]string, len(c.Accounts))
	for i, a := range c.Accounts {
		names[i] = a.Name
	}
	return names
}

// Load reads configuration from environment variables.
//
// Priority:
//  1. DAV_ACCOUNTS (JSON array) — enables multi-account mode.
//  2. DAV_URL + DAV_USERNAME + DAV_PASSWORD — single-account fallback.
func Load() Config {
	cfg := Config{
		Debug: os.Getenv("DAV_DEBUG") == "1",
	}

	if raw := os.Getenv("DAV_ACCOUNTS"); raw != "" {
		var accounts []Account
		if err := json.Unmarshal([]byte(raw), &accounts); err != nil {
			fmt.Fprintf(os.Stderr, "dav-mcp: invalid DAV_ACCOUNTS JSON: %v\n", err)
			os.Exit(1)
		}
		for i, a := range accounts {
			if a.Name == "" {
				accounts[i].Name = fmt.Sprintf("account%d", i+1)
			}
			if a.URL == "" {
				fmt.Fprintf(os.Stderr, "dav-mcp: account %q has no url\n", accounts[i].Name)
				os.Exit(1)
			}
		}
		cfg.Accounts = accounts
		return cfg
	}

	// Single-account fallback.
	url := os.Getenv("DAV_URL")
	username := os.Getenv("DAV_USERNAME")
	password := os.Getenv("DAV_PASSWORD")

	// Keep legacy fields populated for backward compat with any direct callers.
	cfg.DAVURL = url
	cfg.Username = username
	cfg.Password = password

	if url != "" {
		cfg.Accounts = []Account{{
			Name:     "default",
			URL:      url,
			Username: username,
			Password: password,
		}}
	}

	return cfg
}
