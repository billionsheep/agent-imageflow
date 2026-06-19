package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const httpAuditDirectoryName = "http-api"

func (s LocalStorage) AppendHTTPAuditEvent(ctx context.Context, event domain.HTTPAuditEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}
	if strings.TrimSpace(event.EventID) == "" {
		event.EventID = domain.NewID("audit")
	}
	if strings.TrimSpace(event.Source) == "" {
		event.Source = domain.HTTPAuditSourceAPI
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	dir := filepath.Join(s.root, "audit", httpAuditDirectoryName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	filePath := filepath.Join(dir, event.Timestamp.Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func (s LocalStorage) ListHTTPAuditEvents(ctx context.Context, query domain.HTTPAuditQuery) ([]domain.HTTPAuditEvent, error) {
	limit := query.Limit
	if limit < 1 {
		limit = 50
	}

	pattern := filepath.Join(s.root, "audit", httpAuditDirectoryName, "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	events := make([]domain.HTTPAuditEvent, 0, limit)
	for _, filePath := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var event domain.HTTPAuditEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				file.Close()
				return nil, err
			}
			if !httpAuditEventMatchesQuery(event, query) {
				continue
			}
			events = append(events, event)
		}
		if err := scanner.Err(); err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].EventID > events[j].EventID
		}
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

func httpAuditEventMatchesQuery(event domain.HTTPAuditEvent, query domain.HTTPAuditQuery) bool {
	if query.WorkspaceID != "" && event.WorkspaceID != query.WorkspaceID {
		return false
	}
	if query.ProjectID != "" && event.ProjectID != query.ProjectID {
		return false
	}
	if query.CampaignID != "" && event.CampaignID != query.CampaignID {
		return false
	}
	if query.TaskID != "" && event.TaskID != query.TaskID {
		return false
	}
	if query.AssetID != "" && event.AssetID != query.AssetID {
		return false
	}
	if query.InputFileID != "" && event.InputFileID != query.InputFileID {
		return false
	}
	if query.Action != "" && event.Action != query.Action {
		return false
	}
	if query.Actor != "" && event.Actor != query.Actor {
		return false
	}
	if query.StatusCode > 0 && event.StatusCode != query.StatusCode {
		return false
	}
	return true
}
