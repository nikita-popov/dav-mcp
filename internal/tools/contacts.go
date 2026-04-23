package tools

import (
	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

func RegisterContacts(s *mcp.Server, cfg config.Config) {

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
		func(args map[string]any) (any, error) {
			// TODO: CardDAV PROPFIND depth:1 on address book collection
			_ = cfg
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
		func(args map[string]any) (any, error) {
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
		func(args map[string]any) (any, error) {
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
		func(args map[string]any) (any, error) {
			// TODO: generate UID, build vCard 3.0/4.0, PUT to server
			return stub("contacts_create"), nil
		},
	)
}
