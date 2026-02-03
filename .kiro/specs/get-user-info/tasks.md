# Implementation Plan

## Google Chat User Information Tool

- [x] 1. Implement MCP tool definition and registration
  - Add `gchat_get_user_info` tool definition in `tools/gchat.go` with proper description and parameter validation
  - Register the new tool in the existing `RegisterGChatTool` function following established patterns
  - Follow existing MCP tool naming conventions (snake_case) and parameter structure
  - Ensure tool description clearly explains the user ID format requirement (`users/123456789`)
  - _Requirements: User needs interface to retrieve user information by ID_

- [x] 2. Implement core handler function with input validation  
  - Create `gChatGetUserInfoHandler` function in `tools/gchat.go` following existing handler patterns
  - Add input validation to ensure user_id parameter starts with `users/` prefix
  - Implement error responses for invalid input format using `mcp.NewToolResultError`
  - Follow existing error handling patterns and return appropriate error messages
  - _Requirements: System must validate user ID format and handle invalid inputs_

- [x] 3. Implement user search logic across Google Chat spaces
  - Create `findUserInSpaces` helper function to search for target user across accessible spaces
  - Use existing `services.DefaultGChatService().Spaces.List()` to get accessible spaces
  - Iterate through spaces and use `services.DefaultGChatService().Spaces.Members.List()` to search for user
  - Implement early exit strategy - stop searching once user is found in first space
  - Return structured user information (name, displayName, type) when found
  - _Requirements: Core functionality to locate user across accessible chat spaces_

- [x] 4. Add response formatting and error handling
  - Format successful user information response using YAML marshaling with `gopkg.in/yaml.v3`
  - Handle "user not found" scenario with clear error message
  - Add proper error handling for Google Chat API failures using existing patterns
  - Ensure response format matches other Google Chat tools in the project (YAML output)
  - Apply `util.ErrorGuard` wrapper when registering the tool handler
  - _Requirements: System must provide clear feedback and handle all error scenarios_