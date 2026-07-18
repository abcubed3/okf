package cli

import (
	"strings"
	"testing"
)

func TestRunHarvest_NoType(t *testing.T) {
	err := RunHarvest([]string{})
	if err == nil {
		t.Errorf("Expected error for missing harvest type, got nil")
	}
	if err.Error() != "missing harvest type" {
		t.Errorf("Expected 'missing harvest type', got %v", err)
	}
}

func TestRunHarvest_UnknownType(t *testing.T) {
	err := RunHarvest([]string{"unknown-db"})
	if err == nil {
		t.Errorf("Expected error for unknown harvest type, got nil")
	}
}

func TestRunHarvestDB_MissingFlags(t *testing.T) {
	// Should fail because --driver and --conn are missing
	err := RunHarvest([]string{"db"})
	if err == nil {
		t.Errorf("Expected error for missing db flags, got nil")
	}
	if !strings.Contains(err.Error(), "--driver and --conn are required") {
		t.Errorf("Expected missing flags error, got %v", err)
	}
}

func TestRunHarvestWeb_MissingURL(t *testing.T) {
	err := RunHarvest([]string{"web"})
	if err == nil {
		t.Errorf("Expected error for missing web URL, got nil")
	}
	if !strings.Contains(err.Error(), "--url flag is required") {
		t.Errorf("Expected missing URL error, got %v", err)
	}
}

func TestRunHarvestProto_MissingPath(t *testing.T) {
	err := RunHarvest([]string{"proto"})
	if err == nil {
		t.Errorf("Expected error for missing proto path, got nil")
	}
	if !strings.Contains(err.Error(), "--path flag is required") {
		t.Errorf("Expected missing path error, got %v", err)
	}
}

func TestRunHarvestOpenAPI_MissingSpec(t *testing.T) {
	err := RunHarvest([]string{"openapi"})
	if err == nil {
		t.Errorf("Expected error for missing openapi spec, got nil")
	}
	if !strings.Contains(err.Error(), "--spec flag is required") {
		t.Errorf("Expected missing spec error, got %v", err)
	}
}
