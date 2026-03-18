package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/insiderone/notifier/internal/domain"
)

var (
	phoneRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	validate   *validator.Validate
)

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

func ValidateStruct(s any) error {
	return validate.Struct(s)
}

func ValidateNotificationContent(channel domain.Channel, recipient, subject, body string) error {
	switch channel {
	case domain.ChannelEmail:
		if _, err := mail.ParseAddress(recipient); err != nil {
			return fmt.Errorf("%w: invalid email address: %s", domain.ErrValidation, recipient)
		}
		if subject == "" {
			return fmt.Errorf("%w: email notifications require a subject", domain.ErrValidation)
		}
	case domain.ChannelSMS:
		if !phoneRegex.MatchString(recipient) {
			return fmt.Errorf("%w: invalid phone number (must be E.164 format): %s", domain.ErrValidation, recipient)
		}
		if len(body) > 1600 {
			return fmt.Errorf("%w: SMS body exceeds 1600 characters", domain.ErrValidation)
		}
	case domain.ChannelPush:
		if strings.TrimSpace(recipient) == "" {
			return fmt.Errorf("%w: push notifications require a device token", domain.ErrValidation)
		}
		if len(body) > 4096 {
			return fmt.Errorf("%w: push body exceeds 4096 characters", domain.ErrValidation)
		}
	default:
		return fmt.Errorf("%w: %s", domain.ErrInvalidChannel, channel)
	}

	if body == "" {
		return fmt.Errorf("%w: body is required", domain.ErrValidation)
	}
	return nil
}
