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

	// contacts_list
	s.AddTool(
		"contacts_list",
		"List all contacts from the CardDAV address book.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"addressbook": {Type: "string", Description: "Address book path (optional, defaults to primary)"},
				"account":     {Type: "string", Description: "Account name (optional, defaults to primary account)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Optional: []string{"addressbook", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if result, ok := requireCardDAV(sess); !ok {
				return result, nil
			}
			abPath, err := resolveAB(ctx, sess, strArg(args, "addressbook"))
			if err != nil {
				return nil, err
			}
			contacts, err := loadContacts(ctx, sess.Client, abPath)
			if err != nil {
				return nil, err
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatContacts(contacts, abPath),
				}},
			}, nil
		},
	)

	// contacts_get
	s.AddTool(
		"contacts_get",
		"Get a single contact by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Contact UID"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"addressbook", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if result, ok := requireCardDAV(sess); !ok {
				return result, nil
			}
			abPath, err := resolveAB(ctx, sess, strArg(args, "addressbook"))
			if err != nil {
				return nil, err
			}
			uid := strArg(args, "uid")
			contacts, err := loadContacts(ctx, sess.Client, abPath)
			if err != nil {
				return nil, err
			}
			for _, c := range contacts {
				if c.UID == uid {
					return mcp.ToolResult{
						Content: []mcp.ContentItem{{
							Type: "text",
							Text: formatContact(c),
						}},
					}, nil
				}
			}
			return nil, fmt.Errorf("contact %q not found", uid)
		},
	)

	// contacts_search
	s.AddTool(
		"contacts_search",
		"Search contacts by name, email or phone (case-insensitive substring match).",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"query":       {Type: "string", Description: "Search string"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"query"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"query"},
				Optional: []string{"addressbook", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if result, ok := requireCardDAV(sess); !ok {
				return result, nil
			}
			abPath, err := resolveAB(ctx, sess, strArg(args, "addressbook"))
			if err != nil {
				return nil, err
			}
			q := strings.ToLower(strArg(args, "query"))
			all, err := loadContacts(ctx, sess.Client, abPath)
			if err != nil {
				return nil, err
			}
			var matched []vcard.Contact
			for _, c := range all {
				if contactMatches(c, q) {
					matched = append(matched, c)
				}
			}
			var b strings.Builder
			fmt.Fprintf(&b, "Search %q in %s: %d result(s)\n", q, abPath, len(matched))
			for _, c := range matched {
				b.WriteString(formatContact(c))
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{Type: "text", Text: b.String()}},
			}, nil
		},
	)

	// contacts_create
	s.AddTool(
		"contacts_create",
		"Create a new contact in the CardDAV address book.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"name":        {Type: "string", Description: "Full name (required)"},
				"email":       {Type: "string", Description: "Email address (optional)"},
				"phone":       {Type: "string", Description: "Phone number (optional)"},
				"org":         {Type: "string", Description: "Organisation (optional)"},
				"note":        {Type: "string", Description: "Note (optional)"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"name"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"name"},
				Optional: []string{"email", "phone", "org", "note", "addressbook", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if result, ok := requireCardDAV(sess); !ok {
				return result, nil
			}
			abPath, err := resolveAB(ctx, sess, strArg(args, "addressbook"))
			if err != nil {
				return nil, err
			}
			c := vcard.Contact{
				FN:    strArg(args, "name"),
				Email: strArg(args, "email"),
				Phone: strArg(args, "phone"),
				Org:   strArg(args, "org"),
				Notes: strArg(args, "note"),
			}
			vcf := vcard.Build(c)
			uid := vcard.ParseUID(vcf)
			if err := dav.PutContact(ctx, sess.Client, abPath, uid, vcf, ""); err != nil {
				return nil, fmt.Errorf("create contact: %w", err)
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Created contact %q (UID: %s) in %s", c.FN, uid, abPath),
				}},
			}, nil
		},
	)

	// contacts_update
	s.AddTool(
		"contacts_update",
		"Update an existing contact. Only the fields you provide are changed; omitted fields keep their current values.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Contact UID (required)"},
				"name":        {Type: "string", Description: "New full name (optional)"},
				"email":       {Type: "string", Description: "New email (optional)"},
				"phone":       {Type: "string", Description: "New phone (optional)"},
				"org":         {Type: "string", Description: "New organisation (optional)"},
				"note":        {Type: "string", Description: "New note (optional)"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"name", "email", "phone", "org", "note", "addressbook", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if result, ok := requireCardDAV(sess); !ok {
				return result, nil
			}
			abPath, err := resolveAB(ctx, sess, strArg(args, "addressbook"))
			if err != nil {
				return nil, err
			}
			uid := strArg(args, "uid")

			records, err := dav.QueryContactsFull(ctx, sess.Client, abPath)
			if err != nil {
				return nil, fmt.Errorf("contacts_update: fetch contacts: %w", err)
			}
			var rec *dav.ContactRecord
			for i := range records {
				if records[i].Contact.UID == uid {
					rec = &records[i]
					break
				}
			}
			if rec == nil {
				return nil, fmt.Errorf("contact %q not found in %s", uid, abPath)
			}

			c := rec.Contact
			if v := strArg(args, "name"); v != "" {
				c.FN = v
			}
			if _, ok := args["email"]; ok {
				c.Email = strArg(args, "email")
			}
			if _, ok := args["phone"]; ok {
				c.Phone = strArg(args, "phone")
			}
			if _, ok := args["org"]; ok {
				c.Org = strArg(args, "org")
			}
			if _, ok := args["note"]; ok {
				c.Notes = strArg(args, "note")
			}

			vcf := vcard.Build(c)
			if err := dav.PutContactHref(ctx, sess.Client, rec.Href, vcf, rec.ETag); err != nil {
				return nil, fmt.Errorf("contacts_update: put: %w", err)
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Updated contact %q (UID: %s)", c.FN, uid),
				}},
			}, nil
		},
	)

	// contacts_delete
	s.AddTool(
		"contacts_delete",
		"Delete a contact by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Contact UID"},
				"addressbook": {Type: "string", Description: "Address book path (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"addressbook", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if result, ok := requireCardDAV(sess); !ok {
				return result, nil
			}
			abPath, err := resolveAB(ctx, sess, strArg(args, "addressbook"))
			if err != nil {
				return nil, err
			}
			uid := strArg(args, "uid")

			// Resolve UID → Href so we can send a conditional DELETE.
			records, err := dav.QueryContactsFull(ctx, sess.Client, abPath)
			if err != nil {
				return nil, fmt.Errorf("contacts_delete: fetch contacts: %w", err)
			}
			var rec *dav.ContactRecord
			for i := range records {
				if records[i].Contact.UID == uid {
					rec = &records[i]
					break
				}
			}
			if rec == nil {
				return nil, fmt.Errorf("contact %q not found in %s", uid, abPath)
			}

			if err := sess.Client.Delete(ctx, rec.Href, rec.ETag); err != nil {
				return nil, fmt.Errorf("contacts_delete: %w", err)
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Deleted contact UID=%s from %s", uid, abPath),
				}},
			}, nil
		},
	)
}

// resolveAB returns the address book path: explicit arg or auto-discovered primary.
func resolveAB(ctx context.Context, sess *dav.Session, abPath string) (string, error) {
	if abPath != "" {
		return abPath, nil
	}
	if sess.AddressbookHome == "" {
		return "", fmt.Errorf("no addressbook home in session; server may not support CardDAV")
	}
	abs, err := dav.DiscoverCollections(ctx, sess.Client, sess.AddressbookHome)
	if err != nil {
		return "", fmt.Errorf("discover addressbooks: %w", err)
	}
	if len(abs) == 0 {
		return "", fmt.Errorf("no address books found under %s", sess.AddressbookHome)
	}
	return abs[0].Href, nil
}

// loadContacts fetches and parses all contacts from an address book path.
func loadContacts(ctx context.Context, c *dav.Client, abPath string) ([]vcard.Contact, error) {
	raw, err := dav.QueryContacts(ctx, c, abPath)
	if err != nil {
		return nil, err
	}
	var out []vcard.Contact
	for _, r := range raw {
		out = append(out, vcard.ParseContacts(r)...)
	}
	return out, nil
}

// contactMatches returns true if q appears in any field of c.
func contactMatches(c vcard.Contact, q string) bool {
	return strings.Contains(strings.ToLower(c.FN), q) ||
		strings.Contains(strings.ToLower(c.Email), q) ||
		strings.Contains(strings.ToLower(c.Phone), q) ||
		strings.Contains(strings.ToLower(c.Org), q) ||
		strings.Contains(strings.ToLower(c.Notes), q)
}

func strArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func formatContact(c vcard.Contact) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n[%s]\n", c.UID)
	fmt.Fprintf(&b, "  Name:  %s\n", c.FN)
	if c.Email != "" {
		fmt.Fprintf(&b, "  Email: %s\n", c.Email)
	}
	if c.Phone != "" {
		fmt.Fprintf(&b, "  Phone: %s\n", c.Phone)
	}
	if c.Org != "" {
		fmt.Fprintf(&b, "  Org:   %s\n", c.Org)
	}
	if c.Notes != "" {
		fmt.Fprintf(&b, "  Note:  %s\n", c.Notes)
	}
	return b.String()
}

func formatContacts(contacts []vcard.Contact, abPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contacts in %s (%d found):\n", abPath, len(contacts))
	for _, c := range contacts {
		b.WriteString(formatContact(c))
	}
	return b.String()
}
