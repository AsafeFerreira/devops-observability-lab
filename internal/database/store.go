package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
	"github.com/asafeferreira/devops-observability-lab/migrations"
)

var ErrNotFound = errors.New("import not found")

type Store struct {
	pool    *pgxpool.Pool
	metrics *observability.Metrics
	tracer  trace.Tracer
}

type OutboxEvent struct {
	ID        string
	EventType string
	Payload   []byte
}

func OpenWithRetry(ctx context.Context, databaseURL string, metrics *observability.Metrics) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	config.MaxConns = 15
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 15 * time.Minute

	var lastError error
	for attempt := 1; attempt <= 15; attempt++ {
		pool, openError := pgxpool.NewWithConfig(ctx, config)
		if openError == nil {
			pingContext, cancel := context.WithTimeout(ctx, 3*time.Second)
			openError = pool.Ping(pingContext)
			cancel()
			if openError == nil {
				store := &Store{
					pool:    pool,
					metrics: metrics,
					tracer:  otel.Tracer("lab/database"),
				}
				metrics.RegisterPool(pool)
				return store, nil
			}
			pool.Close()
		}
		lastError = openError
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("connect to database after retries: %w", lastError)
}

func (store *Store) Close() { store.pool.Close() }

func (store *Store) Ping(ctx context.Context) error { return store.pool.Ping(ctx) }

func (store *Store) Migrate(ctx context.Context) error {
	connection, err := store.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, `SELECT pg_advisory_lock(84729301)`); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() { _, _ = connection.Exec(context.Background(), `SELECT pg_advisory_unlock(84729301)`) }()

	if _, err := connection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create migration registry: %w", err)
	}

	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || len(entry.Name()) < 4 || entry.Name()[len(entry.Name())-4:] != ".sql" {
			continue
		}
		var exists bool
		if err := connection.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, entry.Name()).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists {
			continue
		}
		contents, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := connection.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.Exec(ctx, string(contents)); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, entry.Name())
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (store *Store) CreateImport(
	ctx context.Context,
	tenantID, idempotencyKey string,
	request model.CreateImportRequest,
	correlationID string,
	traceContext map[string]string,
) (_ *model.Import, err error) {
	ctx, finish := store.operation(ctx, "create_import")
	defer func() { finish(err) }()

	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const query = `
		WITH inserted AS (
			INSERT INTO imports (tenant_id, idempotency_key, source, record_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
			RETURNING id, tenant_id, source, record_count, status, attempts,
			          last_error, created_at, updated_at, processed_at
		)
		SELECT id, tenant_id, source, record_count, status, attempts,
		       last_error, created_at, updated_at, processed_at, false AS replay
		FROM inserted
		UNION ALL
		SELECT id, tenant_id, source, record_count, status, attempts,
		       last_error, created_at, updated_at, processed_at, true AS replay
		FROM imports
		WHERE tenant_id = $1 AND idempotency_key = $2
		  AND NOT EXISTS (SELECT 1 FROM inserted)
		LIMIT 1`

	created := &model.Import{}
	err = tx.QueryRow(ctx, query, tenantID, idempotencyKey, request.Source, request.RecordCount).Scan(
		&created.ID, &created.TenantID, &created.Source, &created.RecordCount,
		&created.Status, &created.Attempts, &created.LastError, &created.CreatedAt,
		&created.UpdatedAt, &created.ProcessedAt, &created.IdempotentReplay,
	)
	if err != nil {
		return nil, err
	}

	if !created.IdempotentReplay {
		event := model.ImportRequestedEvent{
			ImportID:      created.ID,
			TenantID:      tenantID,
			Source:        request.Source,
			RecordCount:   request.RecordCount,
			CorrelationID: correlationID,
			TraceContext:  traceContext,
		}
		payload, marshalError := json.Marshal(event)
		if marshalError != nil {
			return nil, marshalError
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO outbox_events (event_type, payload) VALUES ('import.requested', $1)`, payload); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (store *Store) GetImport(ctx context.Context, tenantID, importID string) (_ *model.Import, err error) {
	ctx, finish := store.operation(ctx, "get_import")
	defer func() { finish(err) }()

	item := &model.Import{}
	err = store.pool.QueryRow(ctx, `
		SELECT id, tenant_id, source, record_count, status, attempts,
		       last_error, created_at, updated_at, processed_at
		FROM imports WHERE tenant_id = $1 AND id = $2`, tenantID, importID).Scan(
		&item.ID, &item.TenantID, &item.Source, &item.RecordCount, &item.Status,
		&item.Attempts, &item.LastError, &item.CreatedAt, &item.UpdatedAt, &item.ProcessedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (store *Store) ListImports(ctx context.Context, tenantID string, limit int) (_ []model.Import, err error) {
	ctx, finish := store.operation(ctx, "list_imports")
	defer func() { finish(err) }()

	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := store.pool.Query(ctx, `
		SELECT id, tenant_id, source, record_count, status, attempts,
		       last_error, created_at, updated_at, processed_at
		FROM imports WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Import, 0)
	for rows.Next() {
		var item model.Import
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Source, &item.RecordCount,
			&item.Status, &item.Attempts, &item.LastError, &item.CreatedAt,
			&item.UpdatedAt, &item.ProcessedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (store *Store) ClaimOutbox(ctx context.Context, workerID string, limit int) (_ []OutboxEvent, err error) {
	ctx, finish := store.operation(ctx, "claim_outbox")
	defer func() { finish(err) }()

	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM outbox_events
			WHERE published_at IS NULL
			  AND (locked_at IS NULL OR locked_at < now() - interval '1 minute')
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE outbox_events event
		SET locked_at = now(), locked_by = $2
		FROM candidates
		WHERE event.id = candidates.id
		RETURNING event.id, event.event_type, event.payload`, limit, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0)
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(&event.ID, &event.EventType, &event.Payload); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (store *Store) MarkOutboxPublished(ctx context.Context, eventID string) error {
	_, err := store.pool.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = now(), locked_at = NULL, locked_by = NULL
		WHERE id = $1`, eventID)
	return err
}

func (store *Store) ReleaseOutbox(ctx context.Context, eventID string) error {
	_, err := store.pool.Exec(ctx, `
		UPDATE outbox_events SET locked_at = NULL, locked_by = NULL WHERE id = $1`, eventID)
	return err
}

func (store *Store) StartProcessing(ctx context.Context, importID string) (_ *model.Import, err error) {
	ctx, finish := store.operation(ctx, "start_processing")
	defer func() { finish(err) }()

	item := &model.Import{}
	err = store.pool.QueryRow(ctx, `
		UPDATE imports
		SET status = 'PROCESSING', attempts = attempts + 1,
		    last_error = '', updated_at = now()
		WHERE id = $1
		RETURNING id, tenant_id, source, record_count, status, attempts,
		          last_error, created_at, updated_at, processed_at`, importID).Scan(
		&item.ID, &item.TenantID, &item.Source, &item.RecordCount, &item.Status,
		&item.Attempts, &item.LastError, &item.CreatedAt, &item.UpdatedAt, &item.ProcessedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (store *Store) CompleteImport(ctx context.Context, importID string, status model.ImportStatus, safeError string) (err error) {
	ctx, finish := store.operation(ctx, "complete_import")
	defer func() { finish(err) }()

	processedAt := "NULL"
	if status == model.StatusSucceeded || status == model.StatusDeadLetter {
		processedAt = "now()"
	}
	query := fmt.Sprintf(`
		UPDATE imports SET status = $2, last_error = $3, updated_at = now(), processed_at = %s
		WHERE id = $1`, processedAt)
	result, err := store.pool.Exec(ctx, query, importID, status, safeError)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (store *Store) CountImportsByStatus(ctx context.Context) (_ map[model.ImportStatus]int64, err error) {
	ctx, finish := store.operation(ctx, "count_imports_by_status")
	defer func() { finish(err) }()

	rows, err := store.pool.Query(ctx, `SELECT status, count(*) FROM imports GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[model.ImportStatus]int64)
	for rows.Next() {
		var status model.ImportStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (store *Store) operation(ctx context.Context, name string) (context.Context, func(error)) {
	started := time.Now()
	ctx, span := store.tracer.Start(ctx, "postgres."+name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", name),
		),
	)
	return ctx, func(err error) {
		result := "success"
		if err != nil {
			result = "failure"
			span.RecordError(err)
			span.SetStatus(codes.Error, "database operation failed")
		}
		store.metrics.DBQueries.WithLabelValues(name, result).Inc()
		store.metrics.DBQueryDuration.WithLabelValues(name).Observe(time.Since(started).Seconds())
		span.End()
	}
}
