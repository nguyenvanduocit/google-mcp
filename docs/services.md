# Services Layer Documentation

## Overview

The Services layer provides centralized OAuth authentication, HTTP client management, and service initialization for all Google API integrations in the MCP server. It implements a robust, thread-safe architecture for managing Google OAuth credentials and initializing API services with proper scope management.

## Module Metrics

| Metric | Value |
|--------|-------|
| **Total Lines of Code** | ~170 (across 3 files) |
| **Number of Services** | 5+ (Calendar, Gmail, YouTube, Chat, HTTP) |
| **Key Hub Components** | `GoogleHttpClient`, `ListGoogleScopes`, `ListChatScopes` |
| **Complexity** | Medium (OAuth flow, scope management) |

## Architecture

### Component Diagram

```mermaid
graph TB
    subgraph "Tool Modules"
        CalendarTools[Calendar Tools]
        GmailTools[Gmail Tools]
        YouTubeTools[YouTube Tools]
        ChatTools[Google Chat Tools]
    end

    subgraph "Service Layer"
        GoogleHttp[GoogleHttpClient<br/>OAuth Client Factory]
        Scopes[Scope Management<br/>ListGoogleScopes<br/>ListChatScopes]

        CalService[calendarService<br/>sync.OnceValue]
        GmailService[gmailService<br/>sync.OnceValue]
        YouTubeService[youtubeService<br/>sync.OnceValue]
        ChatService[DefaultGChatService<br/>sync.OnceValue]
        HttpService[DefaultHttpClient<br/>sync.OnceValue]
    end

    subgraph "OAuth Layer"
        TokenFile[Token File<br/>token.json]
        CredFile[Credentials File<br/>credentials.json]
        GoogleHttp --> TokenFile
        GoogleHttp --> CredFile
        GoogleHttp --> Scopes
    end

    subgraph "Google APIs"
        CalAPI[Calendar API v3]
        GmailAPI[Gmail API v1]
        YouTubeAPI[YouTube Data API v3]
        ChatAPI[Chat API v1]
    end

    CalendarTools --> CalService
    GmailTools --> GmailService
    YouTubeTools --> YouTubeService
    ChatTools --> ChatService

    CalService --> GoogleHttp
    GmailService --> GoogleHttp
    YouTubeService --> GoogleHttp
    ChatService --> GoogleHttp
    HttpService -.Proxy Config.-> GoogleHttp

    GoogleHttp --> CalAPI
    GoogleHttp --> GmailAPI
    GoogleHttp --> YouTubeAPI
    GoogleHttp --> ChatAPI
```

### OAuth Flow

```mermaid
sequenceDiagram
    participant Tool
    participant Service
    participant GoogleHttp
    participant TokenFile
    participant CredFile
    participant OAuth2
    participant GoogleAPI

    Note over Tool,GoogleAPI: Service Initialization Flow

    Tool->>Service: Request Service (First Call)
    Service->>Service: Check sync.OnceValue Cache
    alt Not Initialized
        Service->>GoogleHttp: GoogleHttpClient(tokenFile, credFile)
        GoogleHttp->>TokenFile: Read OAuth Token
        TokenFile-->>GoogleHttp: oauth2.Token
        GoogleHttp->>CredFile: Read Credentials
        CredFile-->>GoogleHttp: Client Config
        GoogleHttp->>GoogleHttp: ListGoogleScopes()
        GoogleHttp->>OAuth2: ConfigFromJSON(scopes)
        OAuth2-->>GoogleHttp: oauth2.Config
        GoogleHttp->>OAuth2: config.Client(token)
        OAuth2-->>GoogleHttp: http.Client
        GoogleHttp-->>Service: Authenticated Client
        Service->>GoogleAPI: NewService(httpClient)
        GoogleAPI-->>Service: API Service
        Service->>Service: Cache in sync.OnceValue
    end
    Service-->>Tool: Cached Service Instance

    Note over Tool,GoogleAPI: Subsequent Calls Use Cache

    Tool->>Service: Request Service (Nth Call)
    Service->>Service: Return Cached Instance
    Service-->>Tool: Same Service Instance
```

## Core Components

### google.go

**File**: `/Users/firegroup/projects/google-mcp/services/google.go`

#### GoogleHttpClient

**Signature**: `func GoogleHttpClient(tokenFile string, credentialsFile string) *http.Client`

**Description**: Factory function that creates an authenticated HTTP client for Google APIs using OAuth2 credentials.

**Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `tokenFile` | string | Path to OAuth token JSON file |
| `credentialsFile` | string | Path to OAuth credentials JSON file |

**Returns**: `*http.Client` configured with OAuth2 authentication and all required Google API scopes.

**Implementation Details**:
```go
func GoogleHttpClient(tokenFile string, credentialsFile string) *http.Client {
    // Read OAuth token
    tok, err := tokenFromFile(tokenFile)
    if err != nil {
        panic(fmt.Sprintf("failed to read token file: %v", err))
    }

    // Read credentials
    ctx := context.Background()
    b, err := os.ReadFile(credentialsFile)
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    // Create OAuth config with all scopes
    config, err := google.ConfigFromJSON(b, ListGoogleScopes()...)
    if err != nil {
        log.Fatalf("Unable to parse client secret file to config: %v", err)
    }

    // Return authenticated HTTP client
    return config.Client(ctx, tok)
}
```

**Features**:
- Reads OAuth token and credentials from files
- Configures all required Google API scopes
- Creates authenticated HTTP client with token refresh
- Panics on failure for fail-fast behavior

**Token File Format** (`token.json`):
```json
{
  "access_token": "ya29.a0AfH6SMBx...",
  "token_type": "Bearer",
  "refresh_token": "1//0gHxQK3...",
  "expiry": "2024-01-15T10:30:00Z"
}
```

**Credentials File Format** (`credentials.json`):
```json
{
  "installed": {
    "client_id": "123456789-abc.apps.googleusercontent.com",
    "project_id": "my-project",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
    "client_secret": "GOCSPX-...",
    "redirect_uris": ["urn:ietf:wg:oauth:2.0:oob", "http://localhost"]
  }
}
```

---

#### ListGoogleScopes (Hub Component)

**Signature**: `func ListGoogleScopes() []string`

**Description**: Returns comprehensive list of all OAuth scopes required for Google API integrations.

**Returns**: `[]string` containing all required OAuth scope URLs.

**Scopes Included**:
```go
func ListGoogleScopes() []string {
    scopes := []string{
        // Gmail Scopes
        gmail.GmailLabelsScope,           // Manage labels
        gmail.GmailModifyScope,           // Modify emails
        gmail.MailGoogleComScope,         // Full access
        gmail.GmailSettingsBasicScope,    // Settings access

        // Calendar Scopes
        calendar.CalendarScope,           // Full calendar access
        calendar.CalendarEventsScope,     // Events access

        // YouTube Scopes
        youtube.YoutubeScope,             // Full YouTube access
        youtube.YoutubeForceSslScope,     // Force SSL
        youtube.YoutubeUploadScope,       // Upload videos
        youtube.YoutubepartnerChannelAuditScope, // Channel audit
        youtube.YoutubepartnerScope,      // Content management
        youtube.YoutubeReadonlyScope,     // Read-only access
    }

    // Add Google Chat scopes
    scopes = append(scopes, ListChatScopes()...)

    return scopes
}
```

**Usage**:
```go
config, err := google.ConfigFromJSON(credentials, ListGoogleScopes()...)
```

**Scope Categories**:

**Gmail Scopes**:
- `https://www.googleapis.com/auth/gmail.labels`
- `https://www.googleapis.com/auth/gmail.modify`
- `https://mail.google.com/`
- `https://www.googleapis.com/auth/gmail.settings.basic`

**Calendar Scopes**:
- `https://www.googleapis.com/auth/calendar`
- `https://www.googleapis.com/auth/calendar.events`

**YouTube Scopes**:
- `https://www.googleapis.com/auth/youtube`
- `https://www.googleapis.com/auth/youtube.force-ssl`
- `https://www.googleapis.com/auth/youtube.upload`
- `https://www.googleapis.com/auth/youtubepartner-channel-audit`
- `https://www.googleapis.com/auth/youtubepartner`
- `https://www.googleapis.com/auth/youtube.readonly`

---

#### ListChatScopes (Hub Component)

**Signature**: `func ListChatScopes() []string`

**Description**: Returns comprehensive list of all OAuth scopes required for Google Chat API.

**Returns**: `[]string` containing all Google Chat OAuth scope URLs.

**Scopes Included**:
```go
func ListChatScopes() []string {
    return []string{
        // Admin Scopes
        "https://www.googleapis.com/auth/chat.admin.memberships",
        "https://www.googleapis.com/auth/chat.admin.memberships.readonly",
        "https://www.googleapis.com/auth/chat.admin.spaces",
        "https://www.googleapis.com/auth/chat.admin.spaces.readonly",

        // Membership Scopes
        "https://www.googleapis.com/auth/chat.memberships",
        "https://www.googleapis.com/auth/chat.memberships.app",
        "https://www.googleapis.com/auth/chat.memberships.readonly",

        // Message Scopes
        "https://www.googleapis.com/auth/chat.messages",
        "https://www.googleapis.com/auth/chat.messages.create",
        "https://www.googleapis.com/auth/chat.messages.reactions",
        "https://www.googleapis.com/auth/chat.messages.reactions.create",
        "https://www.googleapis.com/auth/chat.messages.reactions.readonly",
        "https://www.googleapis.com/auth/chat.messages.readonly",

        // Space Scopes
        "https://www.googleapis.com/auth/chat.spaces",
        "https://www.googleapis.com/auth/chat.spaces.create",
        "https://www.googleapis.com/auth/chat.spaces.readonly",

        // User State Scopes
        "https://www.googleapis.com/auth/chat.users.readstate",
        "https://www.googleapis.com/auth/chat.users.readstate.readonly",
    }
}
```

**Categories**:
- **Admin Scopes**: Full admin access to memberships and spaces
- **Membership Scopes**: Manage space memberships
- **Message Scopes**: Create, read, and react to messages
- **Space Scopes**: Create and manage spaces
- **User State Scopes**: Read user state and read status

---

#### tokenFromFile

**Signature**: `func tokenFromFile(file string) (*oauth2.Token, error)`

**Description**: Helper function to read OAuth token from JSON file.

**Implementation**:
```go
func tokenFromFile(file string) (*oauth2.Token, error) {
    f, err := os.Open(file)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    tok := &oauth2.Token{}
    err = json.NewDecoder(f).Decode(tok)
    return tok, err
}
```

**Returns**: `*oauth2.Token` with access token, refresh token, and expiry.

---

### gchat.go

**File**: `/Users/firegroup/projects/google-mcp/services/gchat.go`

#### NewGChatService

**Signature**: `func NewGChatService() (*chat.Service, error)`

**Description**: Creates and initializes a new Google Chat service instance.

**Implementation**:
```go
func NewGChatService() (*chat.Service, error) {
    ctx := context.Background()

    // Get environment variables
    credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
    if credentialsFile == "" {
        panic("GOOGLE_CREDENTIALS_FILE environment variable must be set")
    }

    tokenFile := os.Getenv("GOOGLE_TOKEN_FILE")
    if tokenFile == "" {
        panic("GOOGLE_TOKEN_FILE environment variable must be set")
    }

    // Create authenticated HTTP client
    client := GoogleHttpClient(tokenFile, credentialsFile)

    // Initialize Google Chat API service
    srv, err := chat.NewService(ctx, option.WithHTTPClient(client))
    if err != nil {
        return nil, fmt.Errorf("failed to create chat service: %v", err)
    }

    return srv, nil
}
```

**Returns**: `*chat.Service` for Google Chat API operations.

**Environment Variables Required**:
- `GOOGLE_CREDENTIALS_FILE`: Path to OAuth credentials JSON
- `GOOGLE_TOKEN_FILE`: Path to OAuth token JSON

---

#### DefaultGChatService

**Type**: `sync.OnceValue[*chat.Service]`

**Description**: Thread-safe singleton for Google Chat service.

**Implementation**:
```go
var DefaultGChatService = sync.OnceValue[*chat.Service](func() *chat.Service {
    srv, err := NewGChatService()
    if err != nil {
        panic(fmt.Sprintf("failed to create chat service: %v", err))
    }
    return srv
})
```

**Features**:
- Lazy initialization on first access
- Thread-safe singleton using `sync.OnceValue`
- Panics on initialization failure
- Cached for application lifetime

**Usage**:
```go
chatService := services.DefaultGChatService()
spaces, err := chatService.Spaces.List().Do()
```

---

### httpclient.go

**File**: `/Users/firegroup/projects/google-mcp/services/httpclient.go`

#### DefaultHttpClient

**Type**: `sync.OnceValue[*http.Client]`

**Description**: Thread-safe singleton for general HTTP client with optional proxy support.

**Implementation**:
```go
var DefaultHttpClient = sync.OnceValue(func() *http.Client {
    transport := &http.Transport{}

    // Configure proxy if environment variable set
    proxyURL := os.Getenv("PROXY_URL")
    if proxyURL != "" {
        proxy, err := url.Parse(proxyURL)
        if err != nil {
            panic(fmt.Sprintf("Failed to parse PROXY_URL: %v", err))
        }
        transport.Proxy = http.ProxyURL(proxy)
        transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    }

    return &http.Client{Transport: transport}
})
```

**Features**:
- Optional proxy support via `PROXY_URL` environment variable
- Insecure TLS verification when using proxy (for corporate proxies)
- Thread-safe singleton pattern
- Lazy initialization

**Environment Variables**:
- `PROXY_URL` (optional): HTTP/HTTPS proxy URL (e.g., `http://proxy.example.com:8080`)

**Usage**:
```go
httpClient := services.DefaultHttpClient()
resp, err := httpClient.Get("https://api.example.com")
```

**Security Note**:
- When proxy is configured, TLS verification is disabled (`InsecureSkipVerify: true`)
- This is common for corporate proxies with self-signed certificates
- Use with caution in production environments

---

## Service Initialization Patterns

### Pattern 1: sync.OnceValue Singleton

All services use the `sync.OnceValue` pattern for thread-safe lazy initialization:

```go
var ServiceName = sync.OnceValue[*ServiceType](func() *ServiceType {
    // Initialization logic
    service, err := createService()
    if err != nil {
        panic(fmt.Sprintf("failed to create service: %v", err))
    }
    return service
})
```

**Benefits**:
- Thread-safe initialization
- Lazy loading (only initialized when needed)
- Single instance per application lifecycle
- No mutex required for access
- Automatic cleanup on panic

### Pattern 2: Environment Variable Configuration

All services read configuration from environment variables:

```go
tokenFile := os.Getenv("GOOGLE_TOKEN_FILE")
if tokenFile == "" {
    panic("GOOGLE_TOKEN_FILE environment variable must be set")
}

credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
if credentialsFile == "" {
    panic("GOOGLE_CREDENTIALS_FILE environment variable must be set")
}
```

**Required Environment Variables**:
- `GOOGLE_TOKEN_FILE`: Path to OAuth token JSON
- `GOOGLE_CREDENTIALS_FILE`: Path to OAuth credentials JSON
- `PROXY_URL` (optional): HTTP/HTTPS proxy URL

### Pattern 3: Fail-Fast Initialization

Services panic on initialization failure:

```go
if credentialsFile == "" {
    panic("GOOGLE_CREDENTIALS_FILE environment variable must be set")
}

srv, err := chat.NewService(ctx, option.WithHTTPClient(client))
if err != nil {
    panic(fmt.Sprintf("failed to create chat service: %v", err))
}
```

**Rationale**:
- Configuration errors should be caught at startup
- Services are critical dependencies
- Fail-fast prevents partial initialization
- Clear error messages for debugging

## OAuth Flow

### Initial Authentication

The OAuth flow must be completed before using the services:

1. **Create OAuth Credentials**:
   - Go to [Google Cloud Console](https://console.cloud.google.com)
   - Create OAuth 2.0 Client ID (Desktop Application)
   - Download credentials JSON

2. **Run Initial Authentication**:
   ```go
   // External authentication script (not in services layer)
   config, _ := google.ConfigFromJSON(credentials, ListGoogleScopes()...)
   authURL := config.AuthCodeURL("state-token")
   // User visits authURL and authorizes
   token := config.Exchange(context.Background(), authCode)
   // Save token to file
   ```

3. **Save Token**:
   ```go
   tokenJSON, _ := json.Marshal(token)
   os.WriteFile("token.json", tokenJSON, 0600)
   ```

4. **Use Services**:
   ```go
   // Services layer reads token.json automatically
   client := GoogleHttpClient("token.json", "credentials.json")
   ```

### Token Refresh

OAuth tokens are automatically refreshed by the `oauth2` library:

```go
// config.Client() creates client with automatic token refresh
client := config.Client(ctx, token)

// Token is refreshed automatically when expired
resp, err := client.Get("https://www.googleapis.com/...")
```

**Token Refresh Flow**:
1. Client detects token expiry
2. Uses refresh token to request new access token
3. Updates token in memory (does NOT update file)
4. Continues with request

**Important**: Token file is read-only. Refreshed tokens are not persisted to disk automatically.

## Error Handling

### Initialization Errors

Services use panic for initialization errors:

```go
if credentialsFile == "" {
    panic("GOOGLE_CREDENTIALS_FILE environment variable must be set")
}
```

**Categories**:

1. **Environment Variable Errors**:
   - Missing `GOOGLE_TOKEN_FILE`
   - Missing `GOOGLE_CREDENTIALS_FILE`
   - Invalid `PROXY_URL` format

2. **File Errors**:
   - Token file not found
   - Credentials file not found
   - Invalid JSON format

3. **OAuth Errors**:
   - Invalid credentials format
   - Failed to create OAuth config
   - Failed to create HTTP client

4. **Service Creation Errors**:
   - Failed to initialize API service
   - Network connectivity issues

### Runtime Errors

Tools handle runtime errors gracefully:

```go
spaces, err := services.DefaultGChatService().Spaces.List().Do()
if err != nil {
    return mcp.NewToolResultError(fmt.Sprintf("failed to list spaces: %v", err)), nil
}
```

**Common Runtime Errors**:
- Token expired (automatically refreshed)
- Network connectivity issues
- API quota exceeded
- Insufficient permissions
- Invalid API parameters

## Usage Examples

### Example 1: Initialize Google Chat Service

```go
package main

import (
    "github.com/nguyenvanduocit/google-mcp/services"
)

func main() {
    // Set environment variables
    os.Setenv("GOOGLE_TOKEN_FILE", "/path/to/token.json")
    os.Setenv("GOOGLE_CREDENTIALS_FILE", "/path/to/credentials.json")

    // Get Chat service (lazy initialization)
    chatService := services.DefaultGChatService()

    // Use service
    spaces, err := chatService.Spaces.List().Do()
    if err != nil {
        log.Fatal(err)
    }

    for _, space := range spaces.Spaces {
        fmt.Printf("Space: %s\n", space.DisplayName)
    }
}
```

### Example 2: Create Custom Service with GoogleHttpClient

```go
package main

import (
    "context"
    "github.com/nguyenvanduocit/google-mcp/services"
    "google.golang.org/api/drive/v3"
    "google.golang.org/api/option"
)

func main() {
    // Get authenticated HTTP client
    client := services.GoogleHttpClient(
        "/path/to/token.json",
        "/path/to/credentials.json",
    )

    // Create custom service (e.g., Google Drive)
    ctx := context.Background()
    driveService, err := drive.NewService(ctx, option.WithHTTPClient(client))
    if err != nil {
        log.Fatal(err)
    }

    // Use service
    files, err := driveService.Files.List().Do()
    // ...
}
```

### Example 3: Use Proxy Configuration

```go
package main

import (
    "os"
    "github.com/nguyenvanduocit/google-mcp/services"
)

func main() {
    // Configure proxy
    os.Setenv("PROXY_URL", "http://proxy.company.com:8080")

    // Get HTTP client with proxy
    httpClient := services.DefaultHttpClient()

    // Use client for general HTTP requests
    resp, err := httpClient.Get("https://api.example.com/data")
    // ...
}
```

### Example 4: List All Required Scopes

```go
package main

import (
    "fmt"
    "github.com/nguyenvanduocit/google-mcp/services"
)

func main() {
    // Get all required scopes
    scopes := services.ListGoogleScopes()

    fmt.Println("Required OAuth Scopes:")
    for _, scope := range scopes {
        fmt.Printf("  - %s\n", scope)
    }

    // Get Chat-specific scopes
    chatScopes := services.ListChatScopes()

    fmt.Println("\nGoogle Chat Scopes:")
    for _, scope := range chatScopes {
        fmt.Printf("  - %s\n", scope)
    }
}
```

## Best Practices

### 1. Environment Variable Management

**Development**:
```bash
# .env file
export GOOGLE_TOKEN_FILE="$HOME/.config/google-mcp/token.json"
export GOOGLE_CREDENTIALS_FILE="$HOME/.config/google-mcp/credentials.json"
```

**Production**:
```bash
# Use absolute paths
export GOOGLE_TOKEN_FILE="/etc/google-mcp/token.json"
export GOOGLE_CREDENTIALS_FILE="/etc/google-mcp/credentials.json"

# Secure file permissions
chmod 600 /etc/google-mcp/token.json
chmod 600 /etc/google-mcp/credentials.json
```

### 2. Credential Security

**DO**:
- Store credentials in secure locations
- Use restrictive file permissions (0600)
- Rotate tokens regularly
- Use service accounts for production
- Keep credentials out of version control

**DON'T**:
- Commit credentials to git
- Share credentials via unsecured channels
- Use overly broad scopes
- Store credentials in code
- Use personal credentials in production

### 3. Service Access

**Efficient Access**:
```go
// Good: Single access to singleton
chatService := services.DefaultGChatService()
spaces, _ := chatService.Spaces.List().Do()
members, _ := chatService.Spaces.Members.List(spaceName).Do()

// Avoid: Multiple accesses (though safe, slightly less efficient)
spaces, _ := services.DefaultGChatService().Spaces.List().Do()
members, _ := services.DefaultGChatService().Spaces.Members.List(spaceName).Do()
```

### 4. Error Handling

**Initialization Errors**:
```go
// Let initialization panics propagate
// Services are critical dependencies
func main() {
    defer func() {
        if r := recover(); r != nil {
            log.Fatalf("Failed to initialize services: %v", r)
        }
    }()

    service := services.DefaultGChatService()
    // ...
}
```

**Runtime Errors**:
```go
// Handle gracefully in tools
spaces, err := chatService.Spaces.List().Do()
if err != nil {
    // Return user-friendly error
    return mcp.NewToolResultError(fmt.Sprintf("failed to list spaces: %v", err)), nil
}
```

## Troubleshooting

### Common Issues

**1. "GOOGLE_TOKEN_FILE environment variable must be set"**
- Environment variable not set
- Typo in variable name
- Variable not exported in shell

**Solution**:
```bash
export GOOGLE_TOKEN_FILE="/path/to/token.json"
# Verify
echo $GOOGLE_TOKEN_FILE
```

**2. "failed to read token file"**
- File path incorrect
- File doesn't exist
- Insufficient permissions

**Solution**:
```bash
ls -la /path/to/token.json
chmod 600 /path/to/token.json
```

**3. "Unable to parse client secret file to config"**
- Credentials file format incorrect
- File corrupted
- Wrong credentials type (needs Desktop Application)

**Solution**:
- Re-download credentials from Google Cloud Console
- Verify JSON format with `jq . < credentials.json`

**4. "failed to create chat service"**
- Network connectivity issues
- OAuth token expired and refresh failed
- API not enabled in Google Cloud Console

**Solution**:
- Check network connectivity
- Regenerate token if refresh token expired
- Enable Google Chat API in Cloud Console

**5. Proxy-related issues**
- Invalid proxy URL format
- Proxy requires authentication
- TLS certificate issues

**Solution**:
```bash
# Test proxy
curl -x http://proxy:8080 https://www.googleapis.com

# Use authenticated proxy
export PROXY_URL="http://user:pass@proxy:8080"
```

## Performance Considerations

### Service Initialization

**Lazy Loading**:
- Services initialized only when first accessed
- Startup time minimized
- No overhead for unused services

**Caching**:
- Each service initialized once per application lifecycle
- Thread-safe access without locks
- No re-initialization on subsequent calls

**Memory Usage**:
- Single service instance per type
- Minimal memory footprint
- HTTP clients reuse connections

### Connection Pooling

The `http.Client` includes connection pooling:
```go
// Default transport configuration
&http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}
```

**Benefits**:
- Connection reuse across requests
- Reduced latency for subsequent requests
- Automatic cleanup of idle connections

## Security Considerations

### OAuth Token Storage

**File Permissions**:
```bash
# Restrictive permissions for token files
chmod 600 token.json
chmod 600 credentials.json

# Verify
ls -la token.json
# Should show: -rw------- (600)
```

**Token Rotation**:
- Refresh tokens can expire
- Access tokens expire every ~1 hour (auto-refreshed)
- Manual rotation recommended every 90 days

### Scope Minimization

Consider creating separate credentials for different use cases:

```go
// Read-only credentials for reporting
readOnlyScopes := []string{
    gmail.GmailReadonlyScope,
    calendar.CalendarReadonlyScope,
    youtube.YoutubeReadonlyScope,
}

// Full access for management tools
fullScopes := services.ListGoogleScopes()
```

### Service Account Usage

For production deployments, consider service accounts:

1. **Create Service Account**:
   - Google Cloud Console → IAM & Admin → Service Accounts
   - Create key and download JSON

2. **Domain-Wide Delegation** (for Google Workspace):
   - Enable domain-wide delegation
   - Configure OAuth scopes
   - Impersonate users as needed

3. **Use Service Account**:
   ```go
   // Alternative to GoogleHttpClient for service accounts
   config, _ := google.JWTConfigFromJSON(
       serviceAccountKey,
       services.ListGoogleScopes()...,
   )
   client := config.Client(context.Background())
   ```

## Related Documentation

- **OAuth 2.0**: [Google OAuth Documentation](https://developers.google.com/identity/protocols/oauth2)
- **API Scopes**: [Google API Scopes](https://developers.google.com/identity/protocols/oauth2/scopes)
- **Service Accounts**: [Service Account Documentation](https://cloud.google.com/iam/docs/service-accounts)
- **Tool Modules**: calendar.md, gmail.md, gchat.md, youtube.md
- **Utilities**: utilities.md (ErrorGuard wrapper)
