package domain

import "testing"

func TestChannel_Valid(t *testing.T) {
	tests := []struct {
		channel Channel
		valid   bool
	}{
		{ChannelSMS, true},
		{ChannelEmail, true},
		{ChannelPush, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.channel.Valid(); got != tt.valid {
			t.Errorf("Channel(%q).Valid() = %v, want %v", tt.channel, got, tt.valid)
		}
	}
}

func TestPriority_Valid(t *testing.T) {
	tests := []struct {
		priority Priority
		valid    bool
	}{
		{PriorityHigh, true},
		{PriorityNormal, true},
		{PriorityLow, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := tt.priority.Valid(); got != tt.valid {
			t.Errorf("Priority(%q).Valid() = %v, want %v", tt.priority, got, tt.valid)
		}
	}
}

func TestPriority_StreamKey(t *testing.T) {
	tests := []struct {
		priority Priority
		want     string
	}{
		{PriorityHigh, "notifications:high"},
		{PriorityNormal, "notifications:normal"},
		{PriorityLow, "notifications:low"},
	}
	for _, tt := range tests {
		if got := tt.priority.StreamKey(); got != tt.want {
			t.Errorf("Priority(%q).StreamKey() = %q, want %q", tt.priority, got, tt.want)
		}
	}
}
