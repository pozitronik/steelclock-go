package telegram

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.TelegramConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "missing auth",
			cfg: &config.TelegramConfig{
				Auth: nil,
			},
			wantErr: true,
		},
		{
			name: "missing api_id",
			cfg: &config.TelegramConfig{
				Auth: &config.TelegramAuthConfig{
					APIID:       0,
					APIHash:     "hash",
					PhoneNumber: "+1234567890",
				},
			},
			wantErr: true,
		},
		{
			name: "missing api_hash",
			cfg: &config.TelegramConfig{
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "",
					PhoneNumber: "+1234567890",
				},
			},
			wantErr: true,
		},
		{
			name: "missing phone_number",
			cfg: &config.TelegramConfig{
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "hash",
					PhoneNumber: "",
				},
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &config.TelegramConfig{
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "testhash",
					PhoneNumber: "+1234567890",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with session path",
			cfg: &config.TelegramConfig{
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "testhash",
					PhoneNumber: "+1234567890",
				},
				SessionPath: "/tmp/test.session",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestClient_IsConnected(t *testing.T) {
	cfg := &config.TelegramConfig{
		Auth: &config.TelegramAuthConfig{
			APIID:       12345,
			APIHash:     "testhash",
			PhoneNumber: "+1234567890",
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should not be connected initially
	if client.IsConnected() {
		t.Error("IsConnected() = true, want false (not connected yet)")
	}
}

func TestClient_GetMessages(t *testing.T) {
	cfg := &config.TelegramConfig{
		Auth: &config.TelegramAuthConfig{
			APIID:       12345,
			APIHash:     "testhash",
			PhoneNumber: "+1234567890",
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should return empty slice initially
	messages := client.GetMessages()
	if messages == nil {
		t.Error("GetMessages() returned nil, want empty slice")
	}
	if len(messages) != 0 {
		t.Errorf("GetMessages() returned %d messages, want 0", len(messages))
	}
}

func TestClient_Filters(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name         string
		filters      *config.TelegramFiltersConfig
		checkPrivate func(*Client) bool
		checkGroup   func(*Client) bool
		checkChannel func(*Client) bool
	}{
		{
			name:    "nil filters - defaults",
			filters: nil,
			checkPrivate: func(c *Client) bool {
				return c.shouldShowPrivate(123)
			},
			checkGroup: func(c *Client) bool {
				return !c.shouldShowGroup(123) // Default: false for groups
			},
			checkChannel: func(c *Client) bool {
				return !c.shouldShowChannel(123) // Default: false for channels
			},
		},
		{
			name: "private disabled",
			filters: &config.TelegramFiltersConfig{
				PrivateChats: &config.TelegramChatFilterConfig{
					Enabled: &falseVal,
				},
			},
			checkPrivate: func(c *Client) bool {
				return !c.shouldShowPrivate(123)
			},
			checkGroup: func(c *Client) bool {
				return true // Not checking groups
			},
			checkChannel: func(c *Client) bool {
				return true // Not checking channels
			},
		},
		{
			name: "groups enabled",
			filters: &config.TelegramFiltersConfig{
				Groups: &config.TelegramChatFilterConfig{
					Enabled: &trueVal,
				},
			},
			checkPrivate: func(c *Client) bool {
				return true // Not checking private
			},
			checkGroup: func(c *Client) bool {
				return c.shouldShowGroup(123)
			},
			checkChannel: func(c *Client) bool {
				return true // Not checking channels
			},
		},
		{
			name: "whitelist overrides disabled",
			filters: &config.TelegramFiltersConfig{
				PrivateChats: &config.TelegramChatFilterConfig{
					Enabled:   &falseVal, // disabled
					Whitelist: []string{"123"},
				},
			},
			checkPrivate: func(c *Client) bool {
				// 123 is whitelisted - shown even though private chats are disabled
				// 456 is not whitelisted - not shown because disabled
				return c.shouldShowPrivate(123) && !c.shouldShowPrivate(456)
			},
			checkGroup: func(c *Client) bool {
				return true
			},
			checkChannel: func(c *Client) bool {
				return true
			},
		},
		{
			name: "blacklist overrides enabled",
			filters: &config.TelegramFiltersConfig{
				PrivateChats: &config.TelegramChatFilterConfig{
					Enabled:   &trueVal, // enabled
					Blacklist: []string{"123"},
				},
			},
			checkPrivate: func(c *Client) bool {
				// 123 is blacklisted - not shown even though private chats are enabled
				// 456 is not blacklisted - shown because enabled
				return !c.shouldShowPrivate(123) && c.shouldShowPrivate(456)
			},
			checkGroup: func(c *Client) bool {
				return true
			},
			checkChannel: func(c *Client) bool {
				return true
			},
		},
		{
			name: "blacklist has highest priority",
			filters: &config.TelegramFiltersConfig{
				PrivateChats: &config.TelegramChatFilterConfig{
					Enabled:   &trueVal,
					Whitelist: []string{"123"},
					Blacklist: []string{"123", "456"}, // 123 is in both lists
				},
			},
			checkPrivate: func(c *Client) bool {
				// 123 is in both - blacklist wins, not shown
				// 456 is only in blacklist - not shown
				// 789 is in neither - shown (enabled)
				return !c.shouldShowPrivate(123) && !c.shouldShowPrivate(456) && c.shouldShowPrivate(789)
			},
			checkGroup: func(c *Client) bool {
				return true
			},
			checkChannel: func(c *Client) bool {
				return true
			},
		},
		{
			name: "whitelist for disabled groups",
			filters: &config.TelegramFiltersConfig{
				Groups: &config.TelegramChatFilterConfig{
					Enabled:   &falseVal, // groups disabled by default
					Whitelist: []string{"999"},
				},
			},
			checkPrivate: func(c *Client) bool {
				return true
			},
			checkGroup: func(c *Client) bool {
				// 999 is whitelisted - shown even though groups are disabled
				// 888 is not whitelisted - not shown
				return c.shouldShowGroup(999) && !c.shouldShowGroup(888)
			},
			checkChannel: func(c *Client) bool {
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.TelegramConfig{
				Auth: &config.TelegramAuthConfig{
					APIID:       12345,
					APIHash:     "testhash",
					PhoneNumber: "+1234567890",
				},
				Filters: tt.filters,
			}

			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if !tt.checkPrivate(client) {
				t.Error("Private chat filter check failed")
			}
			if !tt.checkGroup(client) {
				t.Error("Group filter check failed")
			}
			if !tt.checkChannel(client) {
				t.Error("Channel filter check failed")
			}
		})
	}
}

func TestChatType(t *testing.T) {
	if ChatTypePrivate != 0 {
		t.Errorf("ChatTypePrivate = %d, want 0", ChatTypePrivate)
	}
	if ChatTypeGroup != 1 {
		t.Errorf("ChatTypeGroup = %d, want 1", ChatTypeGroup)
	}
	if ChatTypeChannel != 2 {
		t.Errorf("ChatTypeChannel = %d, want 2", ChatTypeChannel)
	}
}
