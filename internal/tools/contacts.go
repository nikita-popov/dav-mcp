package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/vcard"
)

func RegisterContacts(s *mcp.Server, cfg config.Config) {
	_ = cfg

	// contacts_list
	s.AddTool(
		"contacts_list",
		"List all contacts from the CardDAV address book. Requires an active session (call calendar_connect first).",
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

			session := dav.Get()
			if session == nil {
				return nil, fmt.Errorf("not connected: call calendar_connect first")
			}

			abPath, _ := args["addressbook"].(string)
			if abPath == "" {
				if session.AddressbookHome == "" {
					return nil, fmt.Errorf("no addressbook home in session; server may not support CardDAV")
				}
				// list collections under addressbook home, take first
				abs, err := dav.DiscoverCollections(ctx, session.Client, session.AddressbookHome)
				if err != nil {
					return nil, fmt.Errorf("discover addressbooks: %w", err)
				}
				if len(abs) == 0 {
					return nil, fmt.Errorf("no address books found under %s", session.AddressbookHome)
				}
				abPath = abs[0].Href
			}

			vCards, err := dav.QueryContacts(ctx, session.Client, abPath)
			if err != nil {
				return nil, err
			}

			var all []vcard.Contact
			for _, raw := range vCards {
				all = append(all, vcard.ParseContacts(raw)...)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatContacts(all, abPath),
				}},
			}, nil
		},
	)

	// contacts_search
	s.AddTool(
		"contacts_search",
		"Search contacts by name, email or phone",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"query":       {Type: "string", Description: "Search string (case-insensitive)"},
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
				"uid": {Type: "string", Description: "Contact UID"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
			}, args); err != nil {
				return nil, err
			}
			return stub("contacts_get"), nil
		},
	)

	// contacts_create
	s.AddTool(
		"contacts_create",
		"Create a new contact",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"name":        {Type: "string", Description: "Full name"},
				"email":       {Type: "string", Description: "Email address (optional)"},
				"phone":       {Type: "string", Description: "Phone number (optional)"},
				"org":         {Type: "string", Description: "Organisation (optional)"},
				"note":        {Type: "string", Description: "Note (optional)"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
			},
			Required: []string{"name"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"name"},
				Optional: []string{"email", "phone", "org", "note", "addressbook"},
			}, args); err != nil {
				return nil, err
			}
			return stub("contacts_create"), nil
		},
	)

	// contacts_update
	s.AddTool(
		"contacts_update",
		"Update an existing contact",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":   {Type: "string", Description: "Contact UID"},
				"name":  {Type: "string", Description: "New full name (optional)"},
				"email": {Type: "string", Description: "New email (optional)"},
				"phone": {Type: "string", Description: "New phone (optional)"},
				"org":   {Type: "string", Description: "New organisation (optional)"},
				"note":  {Type: "string", Description: "New note (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"name", "email", "phone", "org", "note"},
			}, args); err != nil {
				return nil, err
			}
			return stub("contacts_update"), nil
		},
	)

	// contacts_delete
	s.AddTool(
		"contacts_delete",
		"Delete a contact by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid": {Type: "string", Description: "Contact UID"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
			}, args); err != nil {
				return nil, err
			}
			return stub("contacts_delete"), nil
		},
	)
}

// formatContacts renders a contact list as human-readable text for the LLM.
func formatContacts(contacts []vcard.Contact, abPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contacts in %s (%d found):\n", abPath, len(contacts))
	for _, c := range contacts {
		fmt.Fprintf(&b, "\n[%s]\n", c.UID)
		fmt.Fprintf(&b, "  Name:  %s\n", c.FN)
		if len(c.Email) > 0 {
			fmt.Fprintf(&b, "  Email: %s\n", strings.Join(c.Email, ", "))
		}
		if len(c.Phone) > 0 {
			fmt.Fprintf(&b, "  Phone: %s\n", strings.Join(c.Phone, ", "))
		}
		if c.Org != "" {
			fmt.Fprintf(&b, "  Org:   %s\n", c.Org)
		}
		if c.Note != "" {
			fmt.Fprintf(&b, "  Note:  %s\n", c.Note)
		}
	}
	return b.String()
}
