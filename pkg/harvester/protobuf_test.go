package harvester

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcubed3/okf/pkg/bundle"
)

func TestProtobufHarvester(t *testing.T) {
	protoContent := `
syntax = "proto3";

package test.v1;

// User represents a system user account.
message User {
  // Unique database identifier.
  string id = 1;
  string email = 2; // User email address
  
  /* Multiline comment
     representing age. */
  int32 age = 3;
}

// UserService coordinates user updates.
service UserService {
  // CreateUser registers a new user.
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
}

message CreateUserRequest {
  string email = 1;
}

message CreateUserResponse {
  User user = 1;
}
`

	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("failed to write mock proto: %v", err)
	}

	h := NewProtobufHarvester(protoFile)
	concepts, err := h.Harvest(context.Background())
	if err != nil {
		t.Fatalf("Harvest failed: %v", err)
	}

	// We expect 4 concepts:
	// 1. Message test.v1.User
	// 2. Service test.v1.UserService
	// 3. Message test.v1.CreateUserRequest
	// 4. Message test.v1.CreateUserResponse
	if len(concepts) != 4 {
		t.Fatalf("expected 4 concepts, got %d", len(concepts))
	}

	var userMsg *bundle.Concept
	var userService *bundle.Concept
	for _, c := range concepts {
		switch c.ID {
case "protobuf/test.v1.user":
			userMsg = c
		case "protobuf/test.v1.userservice":
			userService = c
		}
	}

	if userMsg == nil {
		t.Fatal("user message concept not found")
	}
	if userService == nil {
		t.Fatal("user service concept not found")
	}

	// Verify Message User Details
	if userMsg.Frontmatter.Type != "Protobuf Message" {
		t.Errorf("expected type 'Protobuf Message', got %q", userMsg.Frontmatter.Type)
	}
	if userMsg.Frontmatter.Desc != "User represents a system user account." {
		t.Errorf("expected desc 'User represents a system user account.', got %q", userMsg.Frontmatter.Desc)
	}
	if !strings.Contains(userMsg.Body, "id | string | 1 | Unique database identifier.") {
		t.Errorf("expected field id description details, got: %s", userMsg.Body)
	}
	if !strings.Contains(userMsg.Body, "email | string | 2 | User email address") {
		t.Errorf("expected field email inline description details, got: %s", userMsg.Body)
	}
	if !strings.Contains(userMsg.Body, "age | int32 | 3 |") || !strings.Contains(userMsg.Body, "representing age.") {
		t.Errorf("expected field age multiline comment details, got: %s", userMsg.Body)
	}

	// Verify Service UserService Details
	if userService.Frontmatter.Type != "Protobuf Service" {
		t.Errorf("expected type 'Protobuf Service', got %q", userService.Frontmatter.Type)
	}
	if userService.Frontmatter.Desc != "UserService coordinates user updates." {
		t.Errorf("expected desc 'UserService coordinates user updates.', got %q", userService.Frontmatter.Desc)
	}
	if !strings.Contains(userService.Body, "CreateUser") || !strings.Contains(userService.Body, "CreateUser registers a new user.") {
		t.Errorf("expected RPC CreateUser row, got: %s", userService.Body)
	}
}
