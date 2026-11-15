package integration

import (
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/state"
)

// Integration connection states. An integration's lifecycle was previously
// encoded implicitly across the Enabled/Connected/Error fields; ConnectionState
// makes it a single explicit value.
const (
	ConnDisabled  state.State = "disabled"
	ConnConnected state.State = "connected"
	ConnDegraded  state.State = "degraded" // enabled but last contact failed
	ConnOffline   state.State = "offline"  // enabled but never/no longer connected
)

// ConnectionMachine is the explicit integration connection lifecycle.
var ConnectionMachine = state.NewMachine("integration-connection", ConnDisabled, map[state.State][]state.State{
	ConnDisabled:  {ConnOffline},
	ConnOffline:   {ConnConnected, ConnDisabled},
	ConnConnected: {ConnDegraded, ConnOffline, ConnDisabled},
	ConnDegraded:  {ConnConnected, ConnOffline, ConnDisabled},
})

// Platform represents a constrained string type for known platforms.
type Platform string

const (
	// PlatformGitHub represents the GitHub platform.
	PlatformGitHub Platform = "github"
	// PlatformGitLab represents the GitLab platform.
	PlatformGitLab Platform = "gitlab"
	// PlatformBitbucket represents the Bitbucket platform.
	PlatformBitbucket Platform = "bitbucket"
	// PlatformAzureDevOps represents the Azure DevOps platform.
	PlatformAzureDevOps Platform = "azuredevops"
	// PlatformSlack represents the Slack platform.
	PlatformSlack Platform = "slack"
	// PlatformTeams represents the Microsoft Teams platform.
	PlatformTeams Platform = "teams"
	// PlatformLinear represents the Linear platform.
	PlatformLinear Platform = "linear"
	// PlatformJira represents the Jira platform.
	PlatformJira Platform = "jira"
)

// Message represents a message in a conversation.
type Message struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Role      string    `json:"role"` // user, assistant, system
	CreatedAt time.Time `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Reaction represents a reaction to a message.
type Reaction struct {
	ID        string `json:"id"`
	UserID    int    `json:"user_id"`
	MessageID string `json:"message_id"`
	Type      string `json:"type"` // e.g., "thumbup", "heart"
	CreatedAt time.Time `json:"created_at"`
}

// BotCommand represents a bot command.
type BotCommand struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Platform  Platform  `json:"platform"`
	Handler   string    `json:"handler"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommandParameter represents a parameter for a bot command.
type CommandParameter struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"` // e.g., "string", "integer", "boolean"
	Required    bool          `json:"required"`
	Default     interface{}   `json:"default,omitempty"`
	Description string        `json:"description"`
}

// User represents a user in the integration.
type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Platform    Platform  `json:"platform"`
	PlatformID  string    `json:"platform_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MessageOptions provides options for sending messages
type MessageOptions struct {
	ThreadID     string `json:"thread_id,omitempty"`
	Blocks       []map[string]interface{} `json:"blocks,omitempty"`
	Attachments  []map[string]interface{} `json:"attachments,omitempty"`
	Markdown     bool   `json:"markdown,omitempty"`
	Silent       bool   `json:"silent,omitempty"`
	Ephemeral    bool   `json:"ephemeral,omitempty"`
	ReplyTo      string `json:"reply_to,omitempty"`
}

// IntegrationConfig holds configuration for a platform integration
type IntegrationConfig struct {
	Platform       Platform            `json:"platform"`
	Enabled        bool                `json:"enabled"`
	Credentials    map[string]string   `json:"credentials"`
	WebhookSecret  string              `json:"webhook_secret"`
	BotUserID      string              `json:"bot_user_id"`
	AllowedUsers   []string            `json:"allowed_users"`
	AllowedChannels []string           `json:"allowed_channels"`
	Settings       map[string]interface{} `json:"settings"`
}

// IntegrationStatus represents the status of an integration.
// Enabled/Connected/Error are retained for JSON compatibility; State is
// the explicit lifecycle value derived from them (see ConnectionMachine).
type IntegrationStatus struct {
	ID        string      `json:"id"`
	Platform  Platform    `json:"platform"`
	Enabled   bool        `json:"enabled"`
	Connected bool        `json:"connected"`
	State     state.State `json:"state"`
	LastSeen  time.Time   `json:"last_seen"`
	Error     string      `json:"error,omitempty"`
}

// ConnectionState derives the explicit state from the legacy boolean fields.
func (s *IntegrationStatus) ConnectionState() state.State {
	switch {
	case !s.Enabled:
		return ConnDisabled
	case s.Connected && s.Error != "":
		return ConnDegraded
	case s.Connected:
		return ConnConnected
	default:
		return ConnOffline
	}
}

// PlatformConfig holds the runtime configuration registered for a platform.
type PlatformConfig struct {
	Config IntegrationConfig `json:"config"`
	Status IntegrationStatus `json:"status"`
}

// WebhookRegistration describes a registered webhook endpoint.
type WebhookRegistration struct {
	ID       string   `json:"id"`
	Platform Platform `json:"platform"`
	URL      string   `json:"url"`
	Events   []string `json:"events"`
}

// IntegrationEvent is an inbound event from a platform.
type IntegrationEvent struct {
	ID        string                 `json:"id"`
	Platform  Platform               `json:"platform"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}

// WebhookEvent is a normalized inbound webhook event from any platform.
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Platform  Platform               `json:"platform"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// Channel is a messaging channel on a platform.
type Channel struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Platform   Platform `json:"platform"`
	PlatformID string   `json:"platform_id"`
}

// IntegrationManager manages platform integrations.
type IntegrationManager struct {
	platforms map[Platform]PlatformConfig
	webhooks  map[string]*WebhookRegistration
	events    chan IntegrationEvent
}

// NewIntegrationManager creates a new integration manager with default settings.
func NewIntegrationManager() *IntegrationManager {
	return &IntegrationManager{
		platforms: make(map[Platform]PlatformConfig),
		webhooks:  make(map[string]*WebhookRegistration),
		events:    make(chan IntegrationEvent, 100),
	}
}

// RegisterPlatform registers a platform configuration.
func (m *IntegrationManager) RegisterPlatform(platform Platform, config PlatformConfig) {
	m.platforms[platform] = config
}

// GetPlatformConfig returns the configuration for a platform.
func (m *IntegrationManager) GetPlatformConfig(platform Platform) (PlatformConfig, bool) {
	config, ok := m.platforms[platform]
	return config, ok
}

// RegisterWebhook registers a webhook endpoint.
func (m *IntegrationManager) RegisterWebhook(registration *WebhookRegistration) {
	m.webhooks[registration.ID] = registration
}

// GetWebhook returns a webhook registration by ID.
func (m *IntegrationManager) GetWebhook(id string) (*WebhookRegistration, bool) {
	registration, ok := m.webhooks[id]
	return registration, ok
}

// ProcessEvent processes an integration event.
func (m *IntegrationManager) ProcessEvent(event IntegrationEvent) {
	m.events <- event
}

// Utility function to check if a platform is valid.
func IsValidPlatform(p string) bool {
	for _, platform := range []Platform{
		PlatformGitHub, PlatformGitLab, PlatformBitbucket,
		PlatformAzureDevOps, PlatformSlack, PlatformTeams, PlatformLinear, PlatformJira,
	} {
		if string(platform) == p {
			return true
		}
	}
	return false
}

// Utility function to get all known platforms.
func GetAllPlatforms() []Platform {
	return []Platform{
		PlatformGitHub, PlatformGitLab, PlatformBitbucket,
		PlatformAzureDevOps, PlatformSlack, PlatformTeams, PlatformLinear, PlatformJira,
	}
}

// Utility function to platform string to Platform type.
func PlatformFromString(s string) (Platform, bool) {
	platform := Platform(s)
	valid := IsValidPlatform(string(platform))
	return platform, valid
}