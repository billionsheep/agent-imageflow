package storage

import (
	"context"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestAppendAndListHTTPAuditEvents(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	ctx := context.Background()

	first := domain.HTTPAuditEvent{
		EventID:    "audit_1",
		Timestamp:  time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC),
		Source:     domain.HTTPAuditSourceAPI,
		Action:     "create_task",
		Actor:      "admin",
		ProjectID:  "prj_a",
		TaskID:     "task_1",
		StatusCode: 201,
	}
	second := domain.HTTPAuditEvent{
		EventID:    "audit_2",
		Timestamp:  time.Date(2026, 6, 18, 11, 0, 0, 0, time.UTC),
		Source:     domain.HTTPAuditSourceAPI,
		Action:     "get_asset",
		Actor:      "admin",
		ProjectID:  "prj_b",
		AssetID:    "asset_1",
		StatusCode: 200,
	}

	if err := fs.AppendHTTPAuditEvent(ctx, first); err != nil {
		t.Fatalf("append first event: %v", err)
	}
	if err := fs.AppendHTTPAuditEvent(ctx, second); err != nil {
		t.Fatalf("append second event: %v", err)
	}

	events, err := fs.ListHTTPAuditEvents(ctx, domain.HTTPAuditQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventID != "audit_2" || events[1].EventID != "audit_1" {
		t.Fatalf("expected events to be sorted newest first, got %#v", events)
	}

	filtered, err := fs.ListHTTPAuditEvents(ctx, domain.HTTPAuditQuery{
		Limit:      10,
		ProjectID:  "prj_a",
		Action:     "create_task",
		StatusCode: 201,
	})
	if err != nil {
		t.Fatalf("list filtered events: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(filtered))
	}
	if filtered[0].TaskID != "task_1" {
		t.Fatalf("expected filtered task_id=task_1, got %q", filtered[0].TaskID)
	}
}
