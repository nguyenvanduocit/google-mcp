# Google MCP Documentation

Complete technical documentation for the Google MCP (Model Context Protocol) server.

## 📚 Documentation Index

### Main Documentation
- **[google-mcp.md](google-mcp.md)** - System overview, architecture, configuration, and deployment guide

### Module Documentation

#### Tool Modules
| Module | LOC | Tools | Description |
|--------|-----|-------|-------------|
| **[calendar.md](calendar.md)** | 664 | 3 (11 actions) | Event management, scheduling, availability checking |
| **[gmail.md](gmail.md)** | 593 | 6 (8 actions) | Email operations, filters, labels, search |
| **[gchat.md](gchat.md)** | 606 | 10 | Space management, messaging, user operations |
| **[youtube.md](youtube.md)** | 517 | 4 (11 actions) | Video management, comments, captions |

#### Supporting Modules
| Module | Purpose | Key Components |
|--------|---------|----------------|
| **[services.md](services.md)** | Authentication & HTTP clients | OAuth, service initialization, scope management |
| **[utilities.md](utilities.md)** | Cross-cutting concerns | ErrorGuard (highest PageRank: 0.0587), panic recovery |

## 🎯 Quick Start

1. **System Overview**: Start with [google-mcp.md](google-mcp.md) for architecture and setup
2. **Choose Your Module**: Jump to specific tool documentation based on your needs:
   - Calendar operations → [calendar.md](calendar.md)
   - Email management → [gmail.md](gmail.md)
   - Chat/messaging → [gchat.md](gchat.md)
   - YouTube content → [youtube.md](youtube.md)
3. **Integration**: See [services.md](services.md) for OAuth setup and authentication
4. **Error Handling**: Review [utilities.md](utilities.md) for robust error handling patterns

## 📊 Documentation Statistics

- **Total Documentation**: 145KB across 7 files
- **Total Tools Documented**: 23 tools (46 actions/sub-commands)
- **Mermaid Diagrams**: 30+ architecture and flow diagrams
- **Code Examples**: 100+ complete examples
- **Lines of Documentation**: ~6,000 lines

## 🗺️ Documentation Structure

```
docs/
├── README.md              # This file - documentation index
├── google-mcp.md         # Main system documentation
├── calendar.md           # Google Calendar tools
├── gmail.md              # Gmail tools
├── gchat.md              # Google Chat tools
├── youtube.md            # YouTube tools
├── services.md           # Services layer (OAuth, HTTP)
├── utilities.md          # Error handling utilities
├── codebase_map.json     # Dependency graph with metrics
├── graph.html            # Interactive dependency visualization
└── module_tree.json      # Module hierarchy
```

## 🔑 Key Architectural Insights

### Hub Components (High PageRank)
Critical components with high centrality in the dependency graph:

| Component | PageRank | Fan-In | Module | Role |
|-----------|----------|--------|--------|------|
| **ErrorGuard** | 0.0587 | 5 | utilities | Universal error handler for all tools |
| **getAllUsersFromSpace** | 0.0287 | 1 | gchat | User aggregation with pagination |
| **ListChatScopes** | 0.0282 | 1 | services | Chat API scope definition |
| **parseTimeString** | 0.0247 | 1 | calendar | Time parsing for scheduling |
| **createOrGetLabel** | 0.0233 | 1 | gmail | Idempotent label creation |
| **GoogleHttpClient** | 0.0207 | 1 | services | OAuth HTTP client factory |
| **ListGoogleScopes** | 0.0200 | 2 | services | Comprehensive scope aggregation |

### Design Patterns

1. **Unified Tool Pattern**: Single tool with multiple actions (e.g., `calendar_event`, `gmail_filter`)
2. **Singleton Services**: `sync.OnceValue` for thread-safe lazy initialization
3. **Error Guard Pattern**: Universal panic recovery and error wrapping
4. **Hub-and-Spoke**: Central service layer with tool-specific handlers

## 📖 Documentation Features

Each module documentation includes:

### ✅ Architecture
- Component diagrams (Mermaid)
- Data flow diagrams
- Dependency relationships
- Metrics and complexity analysis

### ✅ Tool Descriptions
- Complete parameter documentation
- Request/response formats (YAML)
- Usage examples
- Error handling patterns

### ✅ Implementation Details
- Helper function documentation
- Algorithm explanations
- Performance considerations
- Edge case handling

### ✅ Integration Guidance
- Service initialization
- OAuth scopes required
- Environment variables
- API references

### ✅ Best Practices
- Error handling strategies
- Security considerations
- Performance optimizations
- Testing recommendations

## 🔍 Finding Information

### By Task
- **Create calendar event** → [calendar.md#calendar_event](calendar.md#1-calendar_event)
- **Search emails** → [gmail.md#gmail_search](gmail.md#1-gmail_search)
- **Send chat message** → [gchat.md#gchat_send_message](gchat.md#2-gchat_send_message)
- **Manage YouTube videos** → [youtube.md#youtube_video](youtube.md#1-youtube_video)
- **Setup OAuth** → [services.md#oauth-flow](services.md#oauth-flow)
- **Handle errors** → [utilities.md#errorguard](utilities.md#errorguard)

### By Concept
- **Authentication** → [services.md#authentication-flow](services.md#authentication-flow)
- **Error Handling** → [utilities.md](utilities.md)
- **Service Initialization** → [services.md#service-initialization](services.md#service-initialization)
- **Pagination** → [gchat.md#pagination-handling](gchat.md#pagination-handling)
- **MIME Parsing** → [gmail.md#extractmessagebody](gmail.md#extractmessagebody)
- **Time Slot Finding** → [calendar.md#calendar_find_time_slot](calendar.md#2-calendar_find_time_slot)

### By Component Type
- **Tools/Handlers** → Tool module docs (calendar, gmail, gchat, youtube)
- **Services** → [services.md](services.md)
- **Utilities** → [utilities.md](utilities.md)
- **Configuration** → [google-mcp.md#configuration](google-mcp.md#configuration)

## 🎨 Diagram Legend

Documentation uses Mermaid diagrams for visualization:

- **Component Diagrams**: Show module structure and dependencies
- **Sequence Diagrams**: Illustrate request/response flows
- **Flowcharts**: Explain complex algorithms and decision logic

## 🛠️ Generated Artifacts

Additional analysis artifacts available:

- **codebase_map.json**: Complete dependency graph with metrics
  - PageRank scores
  - Cyclomatic complexity
  - Fan-in/fan-out analysis
  - Community detection

- **graph.html**: Interactive D3.js visualization
  - Visual dependency graph
  - Community clusters
  - Node metrics

- **module_tree.json**: Hierarchical module structure

## 📝 Documentation Conventions

### Code Examples
- JSON for API requests
- YAML for API responses
- Go for implementation examples

### Tool Names
- Format: `module_action` (e.g., `gmail_search`, `calendar_event`)
- Actions specified via `action` parameter for unified tools

### File Paths
- Absolute paths in examples: `/Users/.../google-mcp/...`
- Environment variables: `GOOGLE_TOKEN_FILE`, `GOOGLE_CREDENTIALS_FILE`

### Links
- Internal: `[text](module.md#section)`
- External: `[text](https://...)`
- All docs in same directory (flat structure)

## 🔗 External References

- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [Google Calendar API v3](https://developers.google.com/calendar/api/v3/reference)
- [Gmail API v1](https://developers.google.com/gmail/api/reference/rest)
- [Google Chat API v1](https://developers.google.com/chat/api/reference/rest)
- [YouTube Data API v3](https://developers.google.com/youtube/v3/docs)
- [OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)

## 📊 Module Metrics Summary

| Metric | Total |
|--------|-------|
| **Total Lines of Code** | 2,380 (across all tool modules) |
| **Total Tools** | 23 tools |
| **Total Actions** | 46 sub-actions |
| **Average Tool Complexity** | Medium |
| **Service Dependencies** | 4 (Calendar, Gmail, Chat, YouTube) |
| **OAuth Scopes Required** | 30+ |
| **API Endpoints Used** | 50+ |

## 🚀 Next Steps

1. **For Developers**:
   - Review [google-mcp.md](google-mcp.md) for system architecture
   - Study [services.md](services.md) for OAuth integration
   - Examine [utilities.md](utilities.md) for error handling patterns

2. **For API Users**:
   - Start with tool documentation for your use case
   - Review usage examples in each module
   - Check OAuth scopes in [services.md](services.md)

3. **For Contributors**:
   - Follow patterns in existing tool implementations
   - Use ErrorGuard for all tool handlers
   - Add comprehensive tool descriptions
   - Update documentation when adding features

## 📄 License

MIT License - See [LICENSE](../LICENSE) for details

## 🤝 Contributing

When contributing:
1. Follow existing documentation structure
2. Add Mermaid diagrams for complex flows
3. Include usage examples
4. Document error cases
5. Update metrics and statistics

---

**Generated**: 2026-02-27
**Version**: 1.0
**Coverage**: 100% of codebase documented
