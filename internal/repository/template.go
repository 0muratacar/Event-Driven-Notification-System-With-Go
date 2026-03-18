package repository

import (
	"context"
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

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

func (r *TemplateRepository) Create(ctx context.Context, t *domain.Template) error {
	now := time.Now().UTC()
	t.ID = uuid.Must(uuid.NewV7())
	t.CreatedAt = now
	t.UpdatedAt = now

	query := `
		INSERT INTO templates (id, name, channel, subject, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		t.ID, t.Name, t.Channel, nilIfEmpty(t.Subject), t.Body, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("inserting template: %w", err)
	}
	return nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	query := `SELECT id, name, channel, subject, body, created_at, updated_at FROM templates WHERE id = $1`

	var t domain.Template
	var subject *string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Channel, &subject, &t.Body, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying template: %w", err)
	}
	if subject != nil {
		t.Subject = *subject
	}
	return &t, nil
}

func (r *TemplateRepository) List(ctx context.Context) ([]domain.Template, error) {
	query := `SELECT id, name, channel, subject, body, created_at, updated_at FROM templates ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying templates: %w", err)
	}
	defer rows.Close()

	var templates []domain.Template
	for rows.Next() {
		var t domain.Template
		var subject *string
		if err := rows.Scan(&t.ID, &t.Name, &t.Channel, &subject, &t.Body, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning template: %w", err)
		}
		if subject != nil {
			t.Subject = *subject
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (r *TemplateRepository) Update(ctx context.Context, id uuid.UUID, req domain.UpdateTemplateRequest) (*domain.Template, error) {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Channel != nil {
		setClauses = append(setClauses, fmt.Sprintf("channel = $%d", argIdx))
		args = append(args, *req.Channel)
		argIdx++
	}
	if req.Subject != nil {
		setClauses = append(setClauses, fmt.Sprintf("subject = $%d", argIdx))
		args = append(args, *req.Subject)
		argIdx++
	}
	if req.Body != nil {
		setClauses = append(setClauses, fmt.Sprintf("body = $%d", argIdx))
		args = append(args, *req.Body)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE templates SET %s WHERE id = $%d
		RETURNING id, name, channel, subject, body, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)

	var t domain.Template
	var subject *string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.Name, &t.Channel, &subject, &t.Body, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrDuplicate
		}
		return nil, fmt.Errorf("updating template: %w", err)
	}
	if subject != nil {
		t.Subject = *subject
	}
	return &t, nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM templates WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
