package app

import (
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestNormalizeScopeUpdateName(t *testing.T) {
	name := "  Demo Scope  "
	normalized, ok, err := normalizeScopeUpdateName(&name)
	if err != nil {
		t.Fatalf("normalizeScopeUpdateName returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected name to be marked as present")
	}
	if normalized != "Demo Scope" {
		t.Fatalf("unexpected normalized name: %q", normalized)
	}
}

func TestNormalizeWorkspaceUpdateRequestRequiresAtLeastOneField(t *testing.T) {
	_, err := normalizeWorkspaceUpdateRequest(domain.UpdateWorkspaceRequest{})
	if err == nil {
		t.Fatal("expected error when update request is empty")
	}
}

func TestNormalizeProjectUpdateRequestRejectsEmptyName(t *testing.T) {
	empty := "   "
	_, err := normalizeProjectUpdateRequest(domain.UpdateProjectRequest{Name: &empty})
	if err == nil {
		t.Fatal("expected error for empty project name")
	}
}
