# Google MCP - Model Context Protocol (MCP) Server

The Model Context Protocol (MCP) implementation in Google MCP enables AI models to interact with Google services through a standardized interface.

## Prerequisites

- Go 1.23.2 or higher
- Google Cloud Platform account with the following APIs enabled:
  - Google Calendar API
  - Gmail API
  - Google Chat API
- Google Cloud credentials (OAuth 2.0 Client ID)

## Installation

### Installing via Go

1. Install the server:

```bash
go install github.com/nguyenvanduocit/google-mcp@latest
```

2. Create a `.env` file with your configuration:

```env
# Required for Google Services
GOOGLE_CREDENTIALS_FILE=  # Required: Path to Google Cloud credentials JSON file
GOOGLE_TOKEN_FILE=       # Required: Path to store Google OAuth tokens

# Optional configurations
ENABLE_TOOLS=           # Optional: Comma-separated list of tool groups to enable (empty = all enabled)
PROXY_URL=             # Optional: HTTP/HTTPS proxy URL if needed
```

https://developers.google.com/workspace/chat/authenticate-authorize-chat-user

3. Configure your Claude's config:

```json
{
  "mcpServers": {
    "google_mcp": {
      "command": "google-mcp",
      "args": ["-env", "/path/to/.env"]
    }
  }
}
```

## Enable Tools

The `ENABLE_TOOLS` environment variable is a comma-separated list of tool groups to enable. Available groups are:
- `calendar` - Google Calendar tools
- `gmail` - Gmail tools
- `gchat` - Google Chat tools

Leave it empty to enable all tools.

## Available Tools

### Group: calendar

#### calendar_create_event
Create a new event in Google Calendar with title, description, time, and attendees.

#### calendar_list_events
List upcoming events in Google Calendar with customizable time range and result limit.

#### calendar_update_event
Update an existing event's details including title, description, time, and attendees.

#### calendar_respond_to_event
Respond to an event invitation (accept, decline, or tentative).

### Group: gchat

#### gchat_list_spaces
List all available Google Chat spaces/rooms.

#### gchat_send_message
Send a message to a Google Chat space or direct message.

### Group: gmail

#### gmail_search
Search emails in Gmail using Gmail's search syntax.

#### gmail_move_to_spam
Move specific emails to spam folder in Gmail by message IDs.

#### gmail_create_filter
Create a Gmail filter with specified criteria and actions:
- Filter by sender, recipient, subject, or custom query
- Add labels
- Mark as important
- Mark as read
- Archive messages

#### gmail_list_filters
List all Gmail filters in the account.

#### gmail_list_labels
List all Gmail labels in the account.

#### gmail_delete_filter
Delete a Gmail filter by its ID.

#### gmail_delete_label
Delete a Gmail label by its ID.


## CLI Usage

In addition to the MCP server, `google-mcp` ships a standalone CLI binary (`google-cli`) for direct terminal use — no MCP client needed.

### Installation

```bash
just install-cli
# or
go install github.com/nguyenvanduocit/google-mcp/cmd/google-cli@latest
```

### Quick Start

```bash
export GOOGLE_AI_API_KEY=your-api-key
# or
google-cli --env .env <command> [flags]
```

### Commands

| Command | Description |
|---------|-------------|
| `list-events` | List Google Calendar events |
| `create-event` | Create a calendar event |
| `update-event` | Update a calendar event |
| `list-emails` | List Gmail emails |
| `send-email` | Send an email via Gmail |
| `get-email` | Get email details |
| `send-chat-message` | Send a Google Chat message |
| `list-chat-spaces` | List Chat spaces |
| `search-youtube` | Search YouTube videos |
| `get-video-details` | Get YouTube video details |

### Examples

```bash
# List calendar events
google-cli list-events --calendar-id primary

# Send an email
google-cli send-email --to recipient@example.com --subject "Hello" --body "World"

# Search YouTube
google-cli search-youtube --query "Go programming"

# JSON output
google-cli list-emails --output json | jq '.[].subject'
```

### Flags

Every command accepts:
- `--env string` — Path to `.env` file
- `--output string` — Output format: `text` (default) or `json`

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install nguyenvanduocit/tap/google-mcp
```
