package db

import (
	"context"
	"database/sql"
)

func Migrate(ctx context.Context, conn *sql.DB) error {
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(420061801)`); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(420061801)`)
	}()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS workspace (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS project (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspace(id),
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			style_preset TEXT NOT NULL DEFAULT '',
			metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS campaign (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspace(id),
			project_id TEXT NOT NULL REFERENCES project(id),
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS generation_task (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspace(id),
			project_id TEXT NOT NULL REFERENCES project(id),
			campaign_id TEXT NOT NULL REFERENCES campaign(id),
			idempotency_key TEXT NOT NULL DEFAULT '',
			input_hash TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL,
			purpose TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL,
			negative_prompt TEXT NOT NULL DEFAULT '',
			style_preset TEXT NOT NULL DEFAULT '',
			aspect_ratio TEXT NOT NULL DEFAULT '1:1',
			output_format TEXT NOT NULL DEFAULT 'png',
			structured_input_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			provider TEXT NOT NULL DEFAULT 'mock',
			status TEXT NOT NULL,
			requested_count INTEGER NOT NULL DEFAULT 1,
			review_required BOOLEAN NOT NULL DEFAULT true,
			created_by TEXT NOT NULL DEFAULT 'local-user',
			trace_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			error_code TEXT,
			error_message TEXT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS generation_task_idempotency_idx
			ON generation_task(workspace_id, project_id, idempotency_key)
			WHERE idempotency_key <> ''`,
		`CREATE TABLE IF NOT EXISTS task_attempt (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES generation_task(id),
			attempt_no INTEGER NOT NULL,
			status TEXT NOT NULL,
			provider TEXT NOT NULL,
			provider_request_id TEXT NOT NULL DEFAULT '',
			started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			finished_at TIMESTAMPTZ,
			latency_ms INTEGER,
			retry_after TIMESTAMPTZ,
			error_code TEXT,
			error_message TEXT,
			raw_response_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			cost_json JSONB NOT NULL DEFAULT '{}'::jsonb
		)`,
		`CREATE TABLE IF NOT EXISTS asset (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspace(id),
			project_id TEXT NOT NULL REFERENCES project(id),
			campaign_id TEXT NOT NULL REFERENCES campaign(id),
			task_id TEXT NOT NULL REFERENCES generation_task(id),
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'image',
			current_version_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS asset_version (
			id TEXT PRIMARY KEY,
			asset_id TEXT NOT NULL REFERENCES asset(id),
			version INTEGER NOT NULL,
			status TEXT NOT NULL,
			file_path TEXT NOT NULL,
			thumbnail_path TEXT NOT NULL,
			metadata_path TEXT NOT NULL,
			object_key TEXT NOT NULL DEFAULT '',
			public_url TEXT NOT NULL DEFAULT '',
			mime_type TEXT NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			hash TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			prompt TEXT NOT NULL,
			parameters_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			cost_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(asset_id, version)
		)`,
		`CREATE INDEX IF NOT EXISTS asset_version_hash_idx ON asset_version(hash)`,
		`CREATE TABLE IF NOT EXISTS review_event (
			id TEXT PRIMARY KEY,
			asset_id TEXT NOT NULL REFERENCES asset(id),
			version_id TEXT NOT NULL REFERENCES asset_version(id),
			action TEXT NOT NULL,
			reviewer TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS delivery_event (
			id TEXT PRIMARY KEY,
			asset_id TEXT NOT NULL REFERENCES asset(id),
			version_id TEXT NOT NULL REFERENCES asset_version(id),
			target_type TEXT NOT NULL DEFAULT '',
			target_ref TEXT NOT NULL DEFAULT '',
			snippet TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}

	for _, statement := range statements {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func SeedDefaults(ctx context.Context, conn *sql.DB, workspaceID, projectID, campaignID string) error {
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO workspace (id, name, metadata_json)
		VALUES ($1, 'Default Workspace', '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, workspaceID); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO project (id, workspace_id, name, description, style_preset, metadata_json)
		VALUES ($1, $2, '小红书 AI 动漫账号', '默认内容账号 demo project', 'anime-cover', '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, projectID, workspaceID); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `
		INSERT INTO campaign (id, workspace_id, project_id, name, description, metadata_json)
		VALUES ($1, $2, $3, '7 天封面图计划', '默认 campaign demo', '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, campaignID, workspaceID, projectID); err != nil {
		return err
	}
	return nil
}
