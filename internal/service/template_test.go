package service

import (
	"testing"

	"github.com/insiderone/notifier/internal/domain"
)

func TestTemplateService_Render(t *testing.T) {
	svc := &TemplateService{}

	tests := []struct {
		name        string
		tmpl        *domain.Template
		vars        map[string]string
		wantSubject string
		wantBody    string
		wantErr     bool
	}{
		{
			name: "simple body template",
			tmpl: &domain.Template{
				Body: "Hello {{.Name}}, your code is {{.Code}}",
			},
			vars:     map[string]string{"Name": "Alice", "Code": "1234"},
			wantBody: "Hello Alice, your code is 1234",
		},
		{
			name: "with subject template",
			tmpl: &domain.Template{
				Subject: "Welcome {{.Name}}",
				Body:    "Hello {{.Name}}!",
			},
			vars:        map[string]string{"Name": "Bob"},
			wantSubject: "Welcome Bob",
			wantBody:    "Hello Bob!",
		},
		{
			name: "no variables",
			tmpl: &domain.Template{
				Body: "Static content",
			},
			vars:     nil,
			wantBody: "Static content",
		},
		{
			name: "invalid template syntax",
			tmpl: &domain.Template{
				Body: "Hello {{.Name",
			},
			vars:    map[string]string{"Name": "Alice"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, body, err := svc.Render(tt.tmpl, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if subject != tt.wantSubject {
					t.Errorf("subject = %q, want %q", subject, tt.wantSubject)
				}
				if body != tt.wantBody {
					t.Errorf("body = %q, want %q", body, tt.wantBody)
				}
			}
		})
	}
}
