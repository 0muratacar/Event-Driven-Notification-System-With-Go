package validator

import (
	"strings"
	"testing"

	"github.com/insiderone/notifier/internal/domain"
)

func TestValidateNotificationContent_Email(t *testing.T) {
	tests := []struct {
		name      string
		recipient string
		subject   string
		body      string
		wantErr   bool
	}{
		{"valid email", "user@example.com", "Hello", "Body text", false},
		{"invalid email", "not-an-email", "Hello", "Body text", true},
		{"missing subject", "user@example.com", "", "Body text", true},
		{"missing body", "user@example.com", "Hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotificationContent(domain.ChannelEmail, tt.recipient, tt.subject, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNotificationContent_SMS(t *testing.T) {
	tests := []struct {
		name      string
		recipient string
		body      string
		wantErr   bool
	}{
		{"valid phone", "+12025551234", "Hello", false},
		{"invalid phone - no plus", "12025551234", "Hello", true},
		{"invalid phone - too short", "+123", "Hello", true},
		{"body too long", "+12025551234", strings.Repeat("a", 1601), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotificationContent(domain.ChannelSMS, tt.recipient, "", tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNotificationContent_Push(t *testing.T) {
	tests := []struct {
		name      string
		recipient string
		body      string
		wantErr   bool
	}{
		{"valid push", "device-token-123", "Hello", false},
		{"empty device token", "", "Hello", true},
		{"whitespace device token", "   ", "Hello", true},
		{"body too long", "device-token-123", strings.Repeat("a", 4097), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNotificationContent(domain.ChannelPush, tt.recipient, "", tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNotificationContent_InvalidChannel(t *testing.T) {
	err := ValidateNotificationContent("invalid", "test", "", "body")
	if err == nil {
		t.Error("expected error for invalid channel")
	}
}
