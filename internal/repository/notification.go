package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/insiderone/notifier/internal/domain"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	templateVars, err := json.Marshal(n.TemplateVars)
	if err != nil {
		return fmt.Errorf("marshaling template vars: %w", err)
	}

	now := time.Now().UTC()
	n.ID = uuid.Must(uuid.NewV7())
	n.CreatedAt = now
	n.UpdatedAt = now

	query := `
		INSERT INTO notifications (
			id, idempotency_key, batch_id, channel, priority, recipient, subject, body,
			template_id, template_vars, status, scheduled_at, attempt_count, max_retries,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err = r.pool.Exec(ctx, query,
		n.ID, nilIfEmpty(n.IdempotencyKey), n.BatchID, n.Channel, n.Priority,
		n.Recipient, nilIfEmpty(n.Subject), n.Body, n.TemplateID, templateVars,
		n.Status, n.ScheduledAt, n.AttemptCount, n.MaxRetries,
		n.CreatedAt, n.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "idempotency") {
				return domain.ErrIdempotencyConflict
			}
			return domain.ErrDuplicate
		}
		return fmt.Errorf("inserting notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	for _, n := range notifications {
		n.ID = uuid.Must(uuid.NewV7())
		n.CreatedAt = now
		n.UpdatedAt = now

		templateVars, err := json.Marshal(n.TemplateVars)
		if err != nil {
			return fmt.Errorf("marshaling template vars: %w", err)
		}

		query := `
			INSERT INTO notifications (
				id, idempotency_key, batch_id, channel, priority, recipient, subject, body,
				template_id, template_vars, status, scheduled_at, attempt_count, max_retries,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

		_, err = tx.Exec(ctx, query,
			n.ID, nilIfEmpty(n.IdempotencyKey), n.BatchID, n.Channel, n.Priority,
			n.Recipient, nilIfEmpty(n.Subject), n.Body, n.TemplateID, templateVars,
			n.Status, n.ScheduledAt, n.AttemptCount, n.MaxRetries,
			n.CreatedAt, n.UpdatedAt,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return domain.ErrIdempotencyConflict
			}
			return fmt.Errorf("inserting notification in batch: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing batch: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	query := `
		SELECT id, idempotency_key, batch_id, channel, priority, recipient, subject, body,
			template_id, template_vars, status, scheduled_at, attempt_count, max_retries,
			last_error, delivered_at, created_at, updated_at
		FROM notifications WHERE id = $1`

	return r.scanNotification(r.pool.QueryRow(ctx, query, id))
}

func (r *NotificationRepository) List(ctx context.Context, filter domain.NotificationFilter) (*domain.NotificationList, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	var conditions []string
	var args []any
	argIdx := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Channel != nil {
		conditions = append(conditions, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *filter.Channel)
		argIdx++
	}
	if filter.BatchID != nil {
		conditions = append(conditions, fmt.Sprintf("batch_id = $%d", argIdx))
		args = append(args, *filter.BatchID)
		argIdx++
	}
	if filter.Cursor != nil {
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.ID)
		argIdx += 2
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, idempotency_key, batch_id, channel, priority, recipient, subject, body,
			template_id, template_vars, status, scheduled_at, attempt_count, max_retries,
			last_error, delivered_at, created_at, updated_at
		FROM notifications %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d`, where, argIdx)
	args = append(args, filter.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying notifications: %w", err)
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		n, err := r.scanNotificationFromRows(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *n)
	}

	result := &domain.NotificationList{}
	if len(notifications) > filter.Limit {
		notifications = notifications[:filter.Limit]
		last := notifications[len(notifications)-1]
		result.NextCursor = &domain.Cursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
	}
	result.Notifications = notifications
	return result, nil
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status, lastError string) error {
	now := time.Now().UTC()
	var query string
	var args []any

	if status == domain.StatusDelivered {
		query = `UPDATE notifications SET status = $1, delivered_at = $2, updated_at = $3 WHERE id = $4`
		args = []any{status, now, now, id}
	} else if lastError != "" {
		query = `UPDATE notifications SET status = $1, last_error = $2, updated_at = $3 WHERE id = $4`
		args = []any{status, lastError, now, id}
	} else {
		query = `UPDATE notifications SET status = $1, updated_at = $2 WHERE id = $3`
		args = []any{status, now, id}
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("updating notification status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE notifications SET attempt_count = attempt_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *NotificationRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status IN ('pending', 'scheduled', 'queued')
		RETURNING id`

	var returnedID uuid.UUID
	err := r.pool.QueryRow(ctx, query, id).Scan(&returnedID)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := r.GetByID(ctx, id)
		if getErr != nil {
			return domain.ErrNotFound
		}
		_ = existing
		return domain.ErrCannotCancel
	}
	return err
}

func (r *NotificationRepository) FetchScheduledDue(ctx context.Context, limit int) ([]*domain.Notification, error) {
	query := `
		UPDATE notifications
		SET status = 'pending', updated_at = NOW()
		WHERE id IN (
			SELECT id FROM notifications
			WHERE status = 'scheduled' AND scheduled_at <= NOW()
			ORDER BY scheduled_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, idempotency_key, batch_id, channel, priority, recipient, subject, body,
			template_id, template_vars, status, scheduled_at, attempt_count, max_retries,
			last_error, delivered_at, created_at, updated_at`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetching scheduled notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		n, err := r.scanNotificationFromRows(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (r *NotificationRepository) CreateDeliveryAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	attempt.ID = uuid.Must(uuid.NewV7())
	attempt.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO delivery_attempts (id, notification_id, attempt_number, status_code, response_body, error, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		attempt.ID, attempt.NotificationID, attempt.AttemptNumber,
		attempt.StatusCode, nilIfEmpty(attempt.ResponseBody), nilIfEmpty(attempt.Error),
		attempt.DurationMs, attempt.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) GetDeliveryAttempts(ctx context.Context, notificationID uuid.UUID) ([]domain.DeliveryAttempt, error) {
	query := `
		SELECT id, notification_id, attempt_number, status_code, response_body, error, duration_ms, created_at
		FROM delivery_attempts
		WHERE notification_id = $1
		ORDER BY attempt_number ASC`

	rows, err := r.pool.Query(ctx, query, notificationID)
	if err != nil {
		return nil, fmt.Errorf("querying delivery attempts: %w", err)
	}
	defer rows.Close()

	var attempts []domain.DeliveryAttempt
	for rows.Next() {
		var a domain.DeliveryAttempt
		var responseBody, errStr *string
		if err := rows.Scan(&a.ID, &a.NotificationID, &a.AttemptNumber, &a.StatusCode,
			&responseBody, &errStr, &a.DurationMs, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning delivery attempt: %w", err)
		}
		if responseBody != nil {
			a.ResponseBody = *responseBody
		}
		if errStr != nil {
			a.Error = *errStr
		}
		attempts = append(attempts, a)
	}
	return attempts, nil
}

func (r *NotificationRepository) scanNotification(row pgx.Row) (*domain.Notification, error) {
	var n domain.Notification
	var idempotencyKey, subject, lastError *string
	var templateVarsJSON []byte

	err := row.Scan(
		&n.ID, &idempotencyKey, &n.BatchID, &n.Channel, &n.Priority,
		&n.Recipient, &subject, &n.Body, &n.TemplateID, &templateVarsJSON,
		&n.Status, &n.ScheduledAt, &n.AttemptCount, &n.MaxRetries,
		&lastError, &n.DeliveredAt, &n.CreatedAt, &n.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning notification: %w", err)
	}

	if idempotencyKey != nil {
		n.IdempotencyKey = *idempotencyKey
	}
	if subject != nil {
		n.Subject = *subject
	}
	if lastError != nil {
		n.LastError = *lastError
	}
	if len(templateVarsJSON) > 0 {
		json.Unmarshal(templateVarsJSON, &n.TemplateVars)
	}
	return &n, nil
}

func (r *NotificationRepository) scanNotificationFromRows(rows pgx.Rows) (*domain.Notification, error) {
	var n domain.Notification
	var idempotencyKey, subject, lastError *string
	var templateVarsJSON []byte

	err := rows.Scan(
		&n.ID, &idempotencyKey, &n.BatchID, &n.Channel, &n.Priority,
		&n.Recipient, &subject, &n.Body, &n.TemplateID, &templateVarsJSON,
		&n.Status, &n.ScheduledAt, &n.AttemptCount, &n.MaxRetries,
		&lastError, &n.DeliveredAt, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning notification row: %w", err)
	}

	if idempotencyKey != nil {
		n.IdempotencyKey = *idempotencyKey
	}
	if subject != nil {
		n.Subject = *subject
	}
	if lastError != nil {
		n.LastError = *lastError
	}
	if len(templateVarsJSON) > 0 {
		json.Unmarshal(templateVarsJSON, &n.TemplateVars)
	}
	return &n, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
