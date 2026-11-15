package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TeamsAdapter implements the Adapter interface for Microsoft Teams
type TeamsAdapter struct {
	config   *IntegrationConfig
	verifier *teamsSigVerifier
	handler  func(ctx context.Context, event *WebhookEvent) error
}

// teamsSigVerifier handles Teams webhook signature verification
type teamsSigVerifier struct {
	webhookSecret string
}

func newTeamsSigVerifier(secret string) *teamsSigVerifier {
	return &teamsSigVerifier{webhookSecret: secret}
}

// Platform returns the platform this adapter handles
func (a *TeamsAdapter) Platform() string {
	return string(PlatformTeams)
}

// VerifyWebhook verifies the webhook signature
func (a *TeamsAdapter) VerifyWebhook(payload []byte, signature string) bool {
	// Teams uses a different signature format: 1s<timestamp>.<hmac-sha256(timestamp+payload, key)
	parts := strings.SplitN(signature, "=", 3)
	if len(parts) != 3 || parts[0] != "1s" {
		return false
	}

	timestamp := parts[1]
	signatureValue := parts[2]

	// Verify timestamp is recent (within 5 minutes)
	ts := int64(0)
	fmt.Sscanf(timestamp, "%d", &ts)
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}

	// Compute expected signature
	baseString := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(a.config.WebhookSecret))
	mac.Write([]byte(baseString))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signatureValue), []byte(expectedSig))
}

// ParseWebhook parses a raw Teams webhook payload into a WebhookEvent
func (a *TeamsAdapter) ParseWebhook(payload []byte, headers map[string]string) (*WebhookEvent, error) {
	// Teams uses different header formats; simplified verification
	if !a.VerifyWebhook(payload, "") {
		return nil, fmt.Errorf("invalid teams signature")
	}

	var teamsEvent struct {
		TopicName    string          `json:"topicName"`
		Data         json.RawMessage `json:"data"`
		EventType    string          `json:"eventType"`
		Timestamp    string          `json:"timestamp"`
	}

	if err := json.Unmarshal(payload, &teamsEvent); err != nil {
		return nil, fmt.Errorf("parsing teams event: %w", err)
	}

	event := &WebhookEvent{
		ID:        uuid.New().String(),
		Platform:  PlatformTeams,
		Type:      teamsEvent.EventType,
		Timestamp: parseTeamsTimestamp(teamsEvent.Timestamp),
	}

	var data map[string]interface{}
	if err := json.Unmarshal(teamsEvent.Data, &data); err != nil {
		data = make(map[string]interface{})
	}

	event.Payload = data
	return event, nil
}

// SendMessage sends a message to a Teams channel/user
func (a *TeamsAdapter) SendMessage(ctx context.Context, channelID, text string, options *MessageOptions) error {
	_ = ctx
	_ = channelID
	_ = text
	_ = options
	fmt.Printf("Sending Teams message to %s: %s\n", channelID, text)
	return nil
}

// UpdateMessage updates an existing message
func (a *TeamsAdapter) UpdateMessage(ctx context.Context, channelID, messageID, text string) error {
	_ = ctx
	_ = channelID
	_ = messageID
	_ = text
	fmt.Printf("Updating Teams message %s: %s\n", messageID, text)
	return nil
}

// DeleteMessage deletes a message
func (a *TeamsAdapter) DeleteMessage(ctx context.Context, channelID, messageID string) error {
	_ = ctx
	_ = channelID
	_ = messageID
	fmt.Printf("Deleting Teams message %s\n", messageID)
	return nil
}

// AddReaction adds an emoji reaction to a message
func (a *TeamsAdapter) AddReaction(ctx context.Context, channelID, messageID, emoji string) error {
	_ = ctx
	_ = channelID
	_ = messageID
	_ = emoji
	fmt.Printf("Adding reaction %s to Teams message %s\n", emoji, messageID)
	return nil
}

// GetUser fetches user info
func (a *TeamsAdapter) GetUser(ctx context.Context, userID string) (*User, error) {
	_ = ctx
	return &User{
		ID:          userID,
		Username:    userID,
		Platform:    PlatformTeams,
		PlatformID:  userID,
	}, nil
}

// GetChannel fetches channel info
func (a *TeamsAdapter) GetChannel(ctx context.Context, channelID string) (*Channel, error) {
	_ = ctx
	return &Channel{
		ID:          channelID,
		Name:        "channel-" + channelID[:8],
		Type:        "channel",
		Platform:    PlatformTeams,
		PlatformID:  channelID,
	}, nil
}

// ListChannels lists available channels
func (a *TeamsAdapter) ListChannels(ctx context.Context) ([]*Channel, error) {
	_ = ctx
	return []*Channel{}, nil
}

// HandleMention handles a bot mention event from Teams
func (a *TeamsAdapter) HandleMention(ctx context.Context, event *WebhookEvent) error {
	// Teams mentions are typically handled through standard webhook handling
	return nil
}

// HandleCommand handles a Teams slash command
func (a *TeamsAdapter) HandleCommand(ctx context.Context, event *WebhookEvent) error {
	if a.handler != nil {
		return a.handler(ctx, event)
	}
	return nil
}

// Start starts the adapter
func (a *TeamsAdapter) Start(ctx context.Context) error {
	_ = ctx
	fmt.Println("Starting Teams adapter")
	return nil
}

// Stop stops the adapter
func (a *TeamsAdapter) Stop(ctx context.Context) error {
	_ = ctx
	fmt.Println("Stopping Teams adapter")
	return nil
}

// HandleHTTP handles Teams webhook requests
func (a *TeamsAdapter) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	if !a.VerifyWebhook(body, "") {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := a.ParseWebhook(body, map[string]string{})
	if err != nil {
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	go a.handleEvent(context.Background(), event)
	w.WriteHeader(http.StatusOK)
}

// handleEvent handles a parsed webhook event
func (a *TeamsAdapter) handleEvent(ctx context.Context, event *WebhookEvent) {
	if a.handler != nil {
		a.handler(ctx, event)
	}
}

// parseTeamsTimestamp parses a Teams timestamp string to time.Time
func parseTeamsTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t
	}
	return time.Now()
}