package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// LinearIssue represents a Linear issue payload
type LinearIssue struct {
	ID        string `json:"id"`
	Identifier string `json:"identifier"`
	Title     string `json:"title"`
	State     *LinearState `json:"state"`
	Priority  *LinearPriority `json:"priority"`
	Labels    []LinearLabel `json:"labels"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Archived  bool   `json:"archived"`
	ClosedAt  *string `json:"closedAt"`
}

// LinearState represents the state of a Linear issue
type LinearState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Color string `json:"color"`
}

// LinearPriority represents the priority of a Linear issue
type LinearPriority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Color string `json:"color"`
}

// LinearLabel represents a label on a Linear issue
type LinearLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Color string `json:"color"`
}

// LinearComment represents a comment on a Linear issue
type LinearComment struct {
	ID        string `json:"id"`
	Body    string `json:"body"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	User      *LinearUser `json:"user"`
}

// LinearUser represents a Linear user
type LinearUser struct {
	ID        string `json:"id"`
	Identifier string `json:"identifier"`
	Name     string `json:"name"`
}

// LinearMention represents a mention in a Linear comment
type LinearMention struct {
	User *LinearUser `json:"user"`
	Text string      `json:"text"`
}

// LinearWebhookEvent represents a Linear webhook event
type LinearWebhookEvent struct {
	Type       string          `json:"type"`
	Issue      *LinearIssue    `json:"issue"`
	Comment    *LinearComment  `json:"comment"`
	Mention    *LinearMention  `json:"mention"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Actor      *LinearUser     `json:"actor"`
	Sequence   int             `json:"sequence"`
	Resource   *LinearResource `json:"resource"`
}

// LinearResource represents the resource in a Linear webhook event
type LinearResource struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// linearAdapter implements integration.Adapter for Linear webhooks
type linearAdapter struct {
	secret     string
	platform   Platform
}

// NewLinearAdapter creates a new Linear webhook adapter
func NewLinearAdapter(secret string, platform Platform) *linearAdapter {
	return &linearAdapter{
		secret: secret,
		platform: platform,
	}
}

// Platform returns the platform name
func (a *linearAdapter) Platform() Platform {
	return a.platform
}

// VerifyWebhook verifies the webhook signature using HMAC SHA256
func (a *linearAdapter) VerifyWebhook(r *http.Request) bool {
	signature := r.Header.Get("Linear-Signature")
	if signature == "" {
		return false
	}

	// Parse the signature format: "sha256=hexhash"
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signatureHash := signature[7:]

	// Read the raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		return false
	}

	// Compute HMAC SHA256
	mac := hmac.New(sha256.New, []byte(a.secret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant-time comparison
	return hmac.Equal([]byte(expectedSig), []byte(signatureHash))
}

// ParseWebhook parses the webhook payload and returns the event type and data
func (a *linearAdapter) ParseWebhook(r *http.Request) (string, json.RawMessage, error) {
	// Read the raw body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Try to unmarshal into a generic map to get the event type
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Get the Linear-Event-Type header
	eventType := r.Header.Get("Linear-Event-Type")
	if eventType == "" {
		return "", nil, fmt.Errorf("missing Linear-Event-Type header")
	}

	return eventType, body, nil
}

// HandleMention handles mention webhook events
func (a *linearAdapter) HandleMention(payload []byte) error {
	var mentionEvent struct {
		Action      string      `json:"action"`
		Mention     *LinearMention `json:"mention"`
		Issue       *LinearIssue  `json:"issue"`
		CreatedAt   string      `json:"createdAt"`
		UpdatedAt   string      `json:"updatedAt"`
		Actor       *LinearUser   `json:"actor"`
	}

	if err := json.Unmarshal(payload, &mentionEvent); err != nil {
		return fmt.Errorf("failed to parse mention event: %w", err)
	}

	log.Printf("Linear mention event: action=%s, mention text=%s, issue=%s", mentionEvent.Action, mentionEvent.Mention.Text, mentionEvent.Issue.Identifier)

	switch mentionEvent.Action {
	case "created":
		log.Printf("Mention created on issue %s by %s", mentionEvent.Issue.Identifier, mentionEvent.Actor.Identifier)
	case "deleted":
		log.Printf("Mention deleted from issue %s", mentionEvent.Issue.Identifier)
	default:
		log.Printf("Unhandled mention action: %s", mentionEvent.Action)
	}

	return nil
}

// HandleCommand handles command webhook events
func (a *linearAdapter) HandleCommand(payload []byte) error {
	var commandEvent struct {
		Action      string       `json:"action"`
		Issue       *LinearIssue `json:"issue"`
		Command     string       `json:"command"`
		Parameters  string       `json:"parameters"`
		CreatedAt   string       `json:"createdAt"`
		UpdatedAt   string       `json:"updatedAt"`
		Actor       *LinearUser  `json:"actor"`
	}

	if err := json.Unmarshal(payload, &commandEvent); err != nil {
		return fmt.Errorf("failed to parse command event: %w", err)
	}

	log.Printf("Linear command event: action=%s, command=%s, issue=%s", commandEvent.Action, commandEvent.Command, commandEvent.Issue.Identifier)

	switch commandEvent.Action {
	case "executed":
		log.Printf("Command %s executed on issue %s by %s", commandEvent.Command, commandEvent.Issue.Identifier, commandEvent.Actor.Identifier)
	case "cancelled":
		log.Printf("Command %s cancelled on issue %s", commandEvent.Command, commandEvent.Issue.Identifier)
	default:
		log.Printf("Unhandled command action: %s", commandEvent.Action)
	}

	return nil
}

// Start starts the webhook listener
func (a *linearAdapter) Start() error {
	log.Println("Linear webhook adapter started")
	return nil
}

// Stop stops the webhook listener
func (a *linearAdapter) Stop() error {
	log.Println("Linear webhook adapter stopped")
	return nil
}

// HandleHTTP handles incoming HTTP webhook requests
func (a *linearAdapter) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify webhook signature
	if !a.VerifyWebhook(r) {
		log.Printf("Webhook signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Parse the webhook
	eventType, payload, err := a.ParseWebhook(r)
	if err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("Received Linear webhook: event=%s", eventType)

	// Handle based on event type
	switch eventType {
	case "issue":
		if err := a.HandleIssueEvent(payload); err != nil {
			log.Printf("Failed to handle issue event: %v", err)
		}
	case "comment":
		if err := a.HandleCommentEvent(payload); err != nil {
			log.Printf("Failed to handle comment event: %v", err)
		}
	case "mention":
		if err := a.HandleMention(payload); err != nil {
			log.Printf("Failed to handle mention event: %v", err)
		}
	default:
		log.Printf("Unhandled Linear event type: %s", eventType)
	}

	// Always respond with 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

// HandleIssueEvent handles issue webhook events
func (a *linearAdapter) HandleIssueEvent(payload []byte) error {
	var issueEvent struct {
		Action      string      `json:"action"`
		Issue       *LinearIssue `json:"issue"`
		CreatedAt   string      `json:"createdAt"`
		UpdatedAt   string      `json:"updatedAt"`
		Actor       *LinearUser   `json:"actor"`
	}

	if err := json.Unmarshal(payload, &issueEvent); err != nil {
		return fmt.Errorf("failed to parse issue event: %w", err)
	}

	log.Printf("Linear Issue event: action=%s, issue #%s, repo=%s", issueEvent.Action, issueEvent.Issue.Identifier, issueEvent.Actor.Identifier)

	switch issueEvent.Action {
	case "created":
		log.Printf("Issue %s created by %s", issueEvent.Issue.Identifier, issueEvent.Actor.Identifier)
	case "updated":
		log.Printf("Issue %s updated by %s", issueEvent.Issue.Identifier, issueEvent.Actor.Identifier)
	case "transited":
		log.Printf("Issue %s transited to new state", issueEvent.Issue.Identifier)
	case "closed":
		log.Printf("Issue %s closed", issueEvent.Issue.Identifier)
	case "reopened":
		log.Printf("Issue %s reopened", issueEvent.Issue.Identifier)
	default:
		log.Printf("Unhandled issue action: %s", issueEvent.Action)
	}

	return nil
}

// HandleCommentEvent handles comment webhook events
func (a *linearAdapter) HandleCommentEvent(payload []byte) error {
	var commentEvent struct {
		Action      string        `json:"action"`
		Comment     *LinearComment `json:"comment"`
		Issue       *LinearIssue   `json:"issue"`
		CreatedAt   string        `json:"createdAt"`
		UpdatedAt   string        `json:"updatedAt"`
		Actor       *LinearUser    `json:"actor"`
	}

	if err := json.Unmarshal(payload, &commentEvent); err != nil {
		return fmt.Errorf("failed to parse comment event: %w", err)
	}

	log.Printf("Linear Comment event: action=%s, comment on issue #%s", commentEvent.Action, commentEvent.Issue.Identifier)

	switch commentEvent.Action {
	case "created":
		log.Printf("Comment created on issue %s by %s", commentEvent.Issue.Identifier, commentEvent.Actor.Identifier)
	case "edited":
		log.Printf("Comment edited on issue %s by %s", commentEvent.Issue.Identifier, commentEvent.Actor.Identifier)
	case "deleted":
		log.Printf("Comment deleted from issue %s", commentEvent.Issue.Identifier)
	default:
		log.Printf("Unhandled comment action: %s", commentEvent.Action)
	}

	return nil
}

// SendMessage sends a message through the platform
func (a *linearAdapter) SendMessage(message string, channelID string) error {
	log.Printf("Linear: sending message to channel %s: %s", channelID, message)
	// In a real implementation, this would send to Linear's API or associated channel
	return nil
}

// UpdateMessage updates an existing message
func (a *linearAdapter) UpdateMessage(messageID string, newContent string) error {
	log.Printf("Linear: updating message %s: %s", messageID, newContent)
	// In a real implementation, this would update a message in Linear
	return nil
}

// DeleteMessage deletes a message
func (a *linearAdapter) DeleteMessage(messageID string) error {
	log.Printf("Linear: deleting message %s", messageID)
	// In a real implementation, this would delete a message from Linear
	return nil
}

// AddReaction adds a reaction to a message or comment
func (a *linearAdapter) AddReaction(itemID string, reactionType string) error {
	log.Printf("Linear: adding reaction %s to item %s", reactionType, itemID)
	// In a real implementation, this would add a reaction in Linear
	return nil
}

// GetUser retrieves a user by ID
func (a *linearAdapter) GetUser(userID string) (string, error) {
	log.Printf("Linear: getting user %s", userID)
	// In a real implementation, this would fetch user from Linear API
	return "", nil
}

// GetChannel retrieves a channel by ID
func (a *linearAdapter) GetChannel(channelID string) (string, error) {
	log.Printf("Linear: getting channel %s", channelID)
	// In a real implementation, this would fetch channel from Linear API
	return "", nil
}

// ListChannels lists all available channels
func (a *linearAdapter) ListChannels() ([]string, error) {
	log.Printf("Linear: listing channels")
	// In a real implementation, this would list channels from Linear API
	return nil, nil
}

// RegisterRoutes registers the Linear webhook endpoint on an http.ServeMux
func RegisterLinearRoutes(mux *http.ServeMux, adapter *linearAdapter) {
	mux.HandleFunc("/linear/webhook", func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify webhook signature
		if !a.VerifyWebhook(r) {
			log.Printf("Webhook signature verification failed")
			http.Error(w, "Invalid signature", http.StatusForbidden)
			return
		}

		// Parse the webhook
		eventType, payload, err := a.ParseWebhook(r)
		if err != nil {
			log.Printf("Failed to parse webhook: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		log.Printf("Received Linear webhook: event=%s", eventType)

		// Handle based on event type
		switch eventType {
		case "issue":
			if err := a.HandleIssueEvent(payload); err != nil {
				log.Printf("Failed to handle issue event: %v", err)
			}
		case "comment":
			if err := a.HandleCommentEvent(payload); err != nil {
				log.Printf("Failed to handle comment event: %v", err)
			}
		case "mention":
			if err := a.HandleMention(payload); err != nil {
				log.Printf("Failed to handle mention event: %v", err)
			}
		default:
			log.Printf("Unhandled Linear event type: %s", eventType)
		}

		// Always respond with 200 OK to acknowledge receipt
		w.WriteHeader(http.StatusOK)
	})
}