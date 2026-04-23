package tools

import (
	"context"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

func RegisterContacts(s *mcp.Server, cfg config.Config) {
	_ = cfg // will be used when real DAV client is wired in

	// contacts_list
	s.AddTool(
		"contacts_list",
		"List all contacts from the CardDAV address book",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"addressbook": {Type: "string", Description: "Address book path (optional, defaults to primary)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Optional: []string{"addressbook"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: CardDAV PROPFIND depth:1 on address book collection
			return stub("contacts_list"), nil
		},
	)

	// contacts_search
	s.AddTool(
		"contacts_search",
		"Search contacts by name, email or phone",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"query":       {Type: "string", Description: "Search string"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
			},
			Required: []string{"query"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"query"},
				Optional: []string{"addressbook"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: addressbook-query REPORT with text-match filter
			return stub("contacts_search"), nil
		},
	)

	// contacts_get
	s.AddTool(
		"contacts_get",
		"Get a single contact by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Contact UID"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"addressbook"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: GET /{addressbook}/{uid}.vcf
			return stub("contacts_get"), nil
		},
	)

	// contacts_create
	s.AddTool(
		"contacts_create",
		"Create a new contact in the address book",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"name":        {Type: "string", Description: "Full name (FN)"},
				"email":       {Type: "string", Description: "Email address (optional)"},
				"phone":       {Type: "string", Description: "Phone number (optional)"},
				"org":         {Type: "string", Description: "Organization (optional)"},
				"notes":       {Type: "string", Description: "Notes (optional)"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
			},
			Required: []string{"name"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"name"},
				Optional: []string{"email", "phone", "org", "notes", "addressbook"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: generate UID, build vCard 3.0/4.0, PUT to server
			return stub("contacts_create"), nil
		},
	)
}
