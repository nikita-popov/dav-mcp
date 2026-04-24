# dav-mcp

An [MCP](https://modelcontextprotocol.io) server for CalDAV and CardDAV.
Exposes calendar and contact management as tools callable by LLM clients
(Claude Desktop, Cursor, etc.).

Transport: **stdio**. No HTTP server, no daemon — the client spawns the
process and communicates over stdin/stdout per the MCP spec.

## Requirements

- Go 1.22+
- A CalDAV / CardDAV server (Nextcloud, Radicale, iCloud, Fastmail, …)

## Build

```sh
make          # tidy deps + build → bin/dav-mcp
make test     # run tests with race detector + coverage
make check    # fmt + vet + test
```

## Configuration

All configuration is via environment variables:

| Variable       | Required | Description                        |
|----------------|----------|------------------------------------|
| `DAV_URL`      | yes      | Base URL of the DAV server         |
| `DAV_USERNAME` | yes      | Username for Basic Auth            |
| `DAV_PASSWORD` | yes      | Password for Basic Auth            |
| `DAV_DEBUG`    | no       | Set to `1` for verbose HTTP logging |

With `DAV_URL`, `DAV_USERNAME`, and `DAV_PASSWORD` set, the server
auto-connects on the first tool call — no explicit `calendar_connect`
needed.

## MCP Client Setup

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "dav": {
      "command": "/path/to/bin/dav-mcp",
      "env": {
        "DAV_URL": "https://dav.example.com",
        "DAV_USERNAME": "alice",
        "DAV_PASSWORD": "secret"
      }
    }
  }
}
```

### Cursor (`~/.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "dav": {
      "command": "/path/to/bin/dav-mcp",
      "env": {
        "DAV_URL": "https://dav.example.com",
        "DAV_USERNAME": "alice",
        "DAV_PASSWORD": "secret"
      }
    }
  }
}
```

## Tools

### Calendar (CalDAV)

| Tool | Status | Description |
|------|--------|-------------|
| `calendar_connect` | ✅ | Connect to a CalDAV server and discover calendars |
| `calendar_reconnect` | ✅ | Reconnect using credentials from environment variables |
| `calendar_get_events` | ✅ | List events in a time range |
| `calendar_create_event` | ✅ | Create a new event |
| `calendar_create_recurring_event` | 🚧 | Create a recurring event with RRULE |
| `calendar_update_event` | 🚧 | Update an existing event |
| `calendar_delete_event` | 🚧 | Delete an event by UID |

### Contacts (CardDAV)

| Tool | Status | Description |
|------|--------|-------------|
| `contacts_list` | ✅ | List all contacts in an address book |
| `contacts_get` | ✅ | Get a single contact by UID |
| `contacts_search` | ✅ | Search contacts by name, email, phone, or org |
| `contacts_create` | ✅ | Create a new contact (vCard 4.0) |
| `contacts_update` | 🚧 | Update an existing contact |
| `contacts_delete` | 🚧 | Delete a contact by UID |

✅ implemented · 🚧 stub (not yet implemented)

## Debugging

Enable verbose HTTP logging to stderr:

```sh
DAV_DEBUG=1 DAV_URL=https://dav.example.com DAV_USERNAME=alice DAV_PASSWORD=secret ./bin/dav-mcp
```

With `DAV_DEBUG=1` the server logs every HTTP request and response body
(truncated at 2 KB / 4 KB respectively) to stderr.

## Project Layout

```
cmd/dav-mcp/      entry point
internal/
  config/         env-based configuration
  dav/            WebDAV HTTP client (Propfind, Report, Put, Delete)
  ical/           iCalendar builder and parser
  mcp/            MCP protocol (stdio transport, JSON-RPC 2.0)
  tools/          tool handlers (calendar, contacts)
  vcard/          vCard 4.0 builder and parser
```

## License

MIT
