package mcp

import (
	"strings"
	"testing"
)

func TestValidate_AllRequired(t *testing.T) {
	s := ArgSchema{Required: []string{"start", "end"}}
	if err := s.Validate(map[string]any{"start": "x", "end": "y"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	s := ArgSchema{Required: []string{"start", "end"}}
	err := s.Validate(map[string]any{"start": "x"})
	if err == nil || !strings.Contains(err.Error(), "end") {
		t.Fatalf("expected missing-required error for 'end', got %v", err)
	}
}

func TestValidate_UnknownField(t *testing.T) {
	s := ArgSchema{Required: []string{"name"}, Optional: []string{"email"}}
	err := s.Validate(map[string]any{"name": "x", "bogus": "y"})
	if err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("expected unknown-field error for 'bogus', got %v", err)
	}
}

func TestValidate_OptionalAllowed(t *testing.T) {
	s := ArgSchema{Required: []string{"name"}, Optional: []string{"email"}}
	if err := s.Validate(map[string]any{"name": "x", "email": "e@e"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_OptionalAbsent(t *testing.T) {
	s := ArgSchema{Required: []string{"name"}, Optional: []string{"email"}}
	if err := s.Validate(map[string]any{"name": "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateArgs_NilArgs(t *testing.T) {
	s := ArgSchema{Required: []string{}, Optional: []string{"x"}}
	if err := ValidateArgs(s, nil); err != nil {
		t.Fatal(err)
	}
}

func TestValidateArgs_NilArgsWithRequired(t *testing.T) {
	s := ArgSchema{Required: []string{"x"}}
	if err := ValidateArgs(s, nil); err == nil {
		t.Fatal("expected error for missing required on nil args")
	}
}
