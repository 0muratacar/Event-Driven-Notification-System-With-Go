package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"text/template"

	"github.com/google/uuid"

	"github.com/insiderone/notifier/internal/domain"
	"github.com/insiderone/notifier/internal/repository"
	"github.com/insiderone/notifier/internal/validator"
)

type TemplateService struct {
	repo   *repository.TemplateRepository
	logger *slog.Logger
}

func NewTemplateService(repo *repository.TemplateRepository, logger *slog.Logger) *TemplateService {
	return &TemplateService{repo: repo, logger: logger}
}

func (s *TemplateService) Create(ctx context.Context, req domain.CreateTemplateRequest) (*domain.Template, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	if !req.Channel.Valid() {
		return nil, domain.ErrInvalidChannel
	}

	t := &domain.Template{
		Name:    req.Name,
		Channel: req.Channel,
		Subject: req.Subject,
		Body:    req.Body,
	}

	if _, err := template.New("test").Parse(t.Body); err != nil {
		return nil, fmt.Errorf("%w: invalid template syntax: %v", domain.ErrTemplateRender, err)
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}

	s.logger.Info("template created", "id", t.ID, "name", t.Name)
	return t, nil
}

func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TemplateService) List(ctx context.Context) ([]domain.Template, error) {
	return s.repo.List(ctx)
}

func (s *TemplateService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateTemplateRequest) (*domain.Template, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	if req.Body != nil {
		if _, err := template.New("test").Parse(*req.Body); err != nil {
			return nil, fmt.Errorf("%w: invalid template syntax: %v", domain.ErrTemplateRender, err)
		}
	}
	return s.repo.Update(ctx, id, req)
}

func (s *TemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *TemplateService) Render(tmpl *domain.Template, vars map[string]string) (subject, body string, err error) {
	bodyTmpl, err := template.New("body").Parse(tmpl.Body)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", domain.ErrTemplateRender, err)
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, vars); err != nil {
		return "", "", fmt.Errorf("%w: %v", domain.ErrTemplateRender, err)
	}

	if tmpl.Subject != "" {
		subjectTmpl, err := template.New("subject").Parse(tmpl.Subject)
		if err != nil {
			return "", "", fmt.Errorf("%w: %v", domain.ErrTemplateRender, err)
		}
		var subjectBuf bytes.Buffer
		if err := subjectTmpl.Execute(&subjectBuf, vars); err != nil {
			return "", "", fmt.Errorf("%w: %v", domain.ErrTemplateRender, err)
		}
		subject = subjectBuf.String()
	}

	return subject, bodyBuf.String(), nil
}
