// Package telegram provides a Telegram client wrapper for receiving notifications
package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/dialog"
)

// MessageInfo contains parsed message information for display
type MessageInfo struct {
	ID          int
	ChatID      int64
	ChatType    ChatType
	ChatTitle   string
	SenderName  string
	Text        string
	Time        time.Time
	IsPinned    bool
	IsOutgoing  bool
	UnreadCount int
}

// ChatType represents the type of chat
type ChatType int

const (
	ChatTypePrivate ChatType = iota
	ChatTypeGroup
	ChatTypeChannel
)

// Client wraps the Telegram client with notification handling
type Client struct {
	mu sync.RWMutex

	cfg         *config.TelegramConfig
	client      *telegram.Client
	api         *tg.Client
	sessionPath string
	ctx         context.Context
	cancel      context.CancelFunc

	// State
	connected     bool
	authenticated bool
	selfID        int64
	messages      []MessageInfo
	maxMessages   int

	// Callbacks
	onMessage func(MessageInfo)
	onError   func(error)

	// Chat cache for resolving names
	chatCache map[int64]string
	userCache map[int64]string
}

// NewClient creates a new Telegram client
func NewClient(cfg *config.TelegramConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("telegram configuration is required")
	}
	if cfg.Auth == nil {
		return nil, fmt.Errorf("telegram auth configuration is required")
	}
	if cfg.Auth.APIID == 0 {
		return nil, fmt.Errorf("telegram api_id is required")
	}
	if cfg.Auth.APIHash == "" {
		return nil, fmt.Errorf("telegram api_hash is required")
	}
	if cfg.Auth.PhoneNumber == "" {
		return nil, fmt.Errorf("telegram phone_number is required")
	}

	// Determine session path
	sessionPath := cfg.SessionPath
	if sessionPath == "" {
		// Default: telegram/{phone}.session in executable directory
		exePath, err := os.Executable()
		if err != nil {
			exePath = "."
		}
		exeDir := filepath.Dir(exePath)
		// Sanitize phone number for filename
		phone := strings.ReplaceAll(cfg.Auth.PhoneNumber, "+", "")
		phone = strings.ReplaceAll(phone, " ", "")
		sessionPath = filepath.Join(exeDir, "telegram", phone+".session")
	}

	// Ensure session directory exists
	sessionDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	maxMessages := 10
	if cfg.Display != nil && cfg.Display.MaxMessages > 0 {
		maxMessages = cfg.Display.MaxMessages
	}

	c := &Client{
		cfg:         cfg,
		sessionPath: sessionPath,
		maxMessages: maxMessages,
		messages:    make([]MessageInfo, 0, maxMessages),
		chatCache:   make(map[int64]string),
		userCache:   make(map[int64]string),
	}

	return c, nil
}

// Connect establishes connection to Telegram
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return nil
	}

	// Use background context for the client - it should run independently
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Create session storage
	sessionStorage := &session.FileStorage{
		Path: c.sessionPath,
	}

	// Create client
	c.client = telegram.NewClient(c.cfg.Auth.APIID, c.cfg.Auth.APIHash, telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  c,
	})

	// Release lock before starting goroutine to avoid deadlock
	c.mu.Unlock()

	// Start client in background
	go func() {
		err := c.client.Run(c.ctx, func(ctx context.Context) error {
			c.mu.Lock()
			c.api = c.client.API()
			c.connected = true
			c.mu.Unlock()

			// Authenticate if needed
			status, err := c.client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get auth status: %w", err)
			}

			if !status.Authorized {
				if err := c.authenticate(ctx); err != nil {
					return fmt.Errorf("authentication failed: %w", err)
				}
			}

			c.mu.Lock()
			c.authenticated = true
			// Get self ID
			self, err := c.api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
			if err == nil && len(self) > 0 {
				if user, ok := self[0].(*tg.User); ok {
					c.selfID = user.ID
				}
			}
			c.mu.Unlock()

			// Keep running until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		})

		if err != nil && c.onError != nil && err != context.Canceled {
			c.onError(err)
		}

		c.mu.Lock()
		c.connected = false
		c.authenticated = false
		c.mu.Unlock()
	}()

	// Wait for connection (not authentication) with timeout
	// Authentication may take longer due to user interaction
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("connection timeout")
		case <-ticker.C:
			c.mu.RLock()
			connected := c.connected
			c.mu.RUnlock()
			if connected {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// authenticate performs interactive authentication
func (c *Client) authenticate(ctx context.Context) error {
	// Create GUI auth flow
	flow := auth.NewFlow(
		guiAuth{phone: c.cfg.Auth.PhoneNumber},
		auth.SendCodeOptions{},
	)

	if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
		return err
	}

	return nil
}

// Disconnect closes the Telegram connection
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.connected = false
	c.authenticated = false
}

// IsConnected returns true if connected and authenticated
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.authenticated
}

// GetMessages returns the latest messages
func (c *Client) GetMessages() []MessageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]MessageInfo, len(c.messages))
	copy(result, c.messages)
	return result
}

// GetUnreadCount returns total unread count across filtered chats
func (c *Client) GetUnreadCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, msg := range c.messages {
		count += msg.UnreadCount
	}
	return count
}

// SetMessageCallback sets the callback for new messages
func (c *Client) SetMessageCallback(fn func(MessageInfo)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = fn
}

// SetErrorCallback sets the callback for errors
func (c *Client) SetErrorCallback(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = fn
}

// Handle implements telegram.UpdateHandler
func (c *Client) Handle(ctx context.Context, u tg.UpdatesClass) error {
	switch updates := u.(type) {
	case *tg.Updates:
		for _, update := range updates.Updates {
			c.handleUpdate(ctx, update, updates.Users, updates.Chats)
		}
	case *tg.UpdatesCombined:
		for _, update := range updates.Updates {
			c.handleUpdate(ctx, update, updates.Users, updates.Chats)
		}
	case *tg.UpdateShort:
		c.handleUpdate(ctx, updates.Update, nil, nil)
	case *tg.UpdateShortMessage:
		c.handleShortMessage(ctx, updates)
	case *tg.UpdateShortChatMessage:
		c.handleShortChatMessage(ctx, updates)
	}
	return nil
}

// handleUpdate processes a single update
func (c *Client) handleUpdate(ctx context.Context, update tg.UpdateClass, users []tg.UserClass, chats []tg.ChatClass) {
	// Cache users and chats
	c.cacheEntities(users, chats)

	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		c.processMessage(ctx, u.Message)
	case *tg.UpdateNewChannelMessage:
		c.processMessage(ctx, u.Message)
	case *tg.UpdatePinnedMessages:
		if c.shouldShowPinned() {
			// Handle pinned message notification
			c.handlePinnedUpdate(ctx, u.Peer, u.Messages)
		}
	case *tg.UpdatePinnedChannelMessages:
		if c.shouldShowPinned() {
			c.handlePinnedChannelUpdate(ctx, u.ChannelID, u.Messages)
		}
	}
}

// handleShortMessage handles UpdateShortMessage (private message optimization)
func (c *Client) handleShortMessage(ctx context.Context, u *tg.UpdateShortMessage) {
	if !c.shouldShowPrivate(u.UserID) {
		return
	}

	senderID := u.UserID
	if u.Out {
		senderID = c.selfID
	}

	msg := MessageInfo{
		ID:         u.ID,
		ChatID:     u.UserID,
		ChatType:   ChatTypePrivate,
		ChatTitle:  c.getUserName(u.UserID),
		SenderName: c.getUserName(senderID),
		Text:       u.Message,
		Time:       time.Unix(int64(u.Date), 0),
		IsOutgoing: u.Out,
	}

	c.addMessage(msg)
}

// handleShortChatMessage handles UpdateShortChatMessage (group message optimization)
func (c *Client) handleShortChatMessage(ctx context.Context, u *tg.UpdateShortChatMessage) {
	if !c.shouldShowGroup(u.ChatID) {
		return
	}

	msg := MessageInfo{
		ID:         u.ID,
		ChatID:     u.ChatID,
		ChatType:   ChatTypeGroup,
		ChatTitle:  c.getChatName(u.ChatID),
		SenderName: c.getUserName(u.FromID),
		Text:       u.Message,
		Time:       time.Unix(int64(u.Date), 0),
		IsOutgoing: u.Out,
	}

	c.addMessage(msg)
}

// processMessage processes a full message object
func (c *Client) processMessage(ctx context.Context, msg tg.MessageClass) {
	message, ok := msg.(*tg.Message)
	if !ok {
		return
	}

	// Skip outgoing messages unless we want them
	if message.Out {
		return
	}

	// Determine chat type and ID
	var chatID int64
	var chatType ChatType
	var chatTitle string

	switch peer := message.PeerID.(type) {
	case *tg.PeerUser:
		if !c.shouldShowPrivate(peer.UserID) {
			return
		}
		chatID = peer.UserID
		chatType = ChatTypePrivate
		chatTitle = c.getUserName(peer.UserID)

	case *tg.PeerChat:
		if !c.shouldShowGroup(peer.ChatID) {
			return
		}
		chatID = peer.ChatID
		chatType = ChatTypeGroup
		chatTitle = c.getChatName(peer.ChatID)

	case *tg.PeerChannel:
		if !c.shouldShowChannel(peer.ChannelID) {
			return
		}
		chatID = peer.ChannelID
		chatType = ChatTypeChannel
		chatTitle = c.getChannelName(peer.ChannelID)

	default:
		return
	}

	// Get sender name
	senderName := ""
	if message.FromID != nil {
		switch from := message.FromID.(type) {
		case *tg.PeerUser:
			senderName = c.getUserName(from.UserID)
		case *tg.PeerChannel:
			senderName = c.getChannelName(from.ChannelID)
		}
	}

	info := MessageInfo{
		ID:         message.ID,
		ChatID:     chatID,
		ChatType:   chatType,
		ChatTitle:  chatTitle,
		SenderName: senderName,
		Text:       message.Message,
		Time:       time.Unix(int64(message.Date), 0),
		IsPinned:   message.Pinned,
		IsOutgoing: message.Out,
	}

	c.addMessage(info)
}

// handlePinnedUpdate handles pinned message updates
func (c *Client) handlePinnedUpdate(ctx context.Context, peer tg.PeerClass, messageIDs []int) {
	var chatID int64
	var chatType ChatType

	switch p := peer.(type) {
	case *tg.PeerUser:
		if !c.shouldShowPrivate(p.UserID) {
			return
		}
		chatID = p.UserID
		chatType = ChatTypePrivate
	case *tg.PeerChat:
		if !c.shouldShowGroup(p.ChatID) {
			return
		}
		chatID = p.ChatID
		chatType = ChatTypeGroup
	default:
		return
	}

	msg := MessageInfo{
		ChatID:    chatID,
		ChatType:  chatType,
		ChatTitle: c.getChatName(chatID),
		Text:      "[Message pinned]",
		Time:      time.Now(),
		IsPinned:  true,
	}

	c.addMessage(msg)
}

// handlePinnedChannelUpdate handles channel pinned message updates
func (c *Client) handlePinnedChannelUpdate(ctx context.Context, channelID int64, messageIDs []int) {
	if !c.shouldShowChannel(channelID) {
		return
	}

	msg := MessageInfo{
		ChatID:    channelID,
		ChatType:  ChatTypeChannel,
		ChatTitle: c.getChannelName(channelID),
		Text:      "[Message pinned]",
		Time:      time.Now(),
		IsPinned:  true,
	}

	c.addMessage(msg)
}

// addMessage adds a message to the list and triggers callback
func (c *Client) addMessage(msg MessageInfo) {
	c.mu.Lock()

	// Add to front of list
	c.messages = append([]MessageInfo{msg}, c.messages...)

	// Trim to max size
	if len(c.messages) > c.maxMessages {
		c.messages = c.messages[:c.maxMessages]
	}

	callback := c.onMessage
	c.mu.Unlock()

	if callback != nil {
		callback(msg)
	}
}

// Filter methods

func (c *Client) shouldShowPrivate(userID int64) bool {
	defaultEnabled := true // Default: show private chats
	if c.cfg.Filters == nil {
		return defaultEnabled
	}
	return c.shouldShow(c.cfg.Filters.PrivateChats, userID, defaultEnabled)
}

func (c *Client) shouldShowGroup(chatID int64) bool {
	defaultEnabled := false // Default: don't show groups
	if c.cfg.Filters == nil {
		return defaultEnabled
	}
	return c.shouldShow(c.cfg.Filters.Groups, chatID, defaultEnabled)
}

func (c *Client) shouldShowChannel(channelID int64) bool {
	defaultEnabled := false // Default: don't show channels
	if c.cfg.Filters == nil {
		return defaultEnabled
	}
	return c.shouldShow(c.cfg.Filters.Channels, channelID, defaultEnabled)
}

func (c *Client) shouldShow(filter *config.TelegramChatFilterConfig, id int64, defaultEnabled bool) bool {
	if filter == nil {
		return defaultEnabled
	}

	idStr := fmt.Sprintf("%d", id)

	// Blacklist has highest priority - never show these
	for _, item := range filter.Blacklist {
		if item == idStr {
			return false
		}
	}

	// Whitelist overrides enabled setting - always show these
	for _, item := range filter.Whitelist {
		if item == idStr {
			return true
		}
	}

	// Check if this chat type is enabled
	if filter.Enabled != nil {
		return *filter.Enabled
	}

	return defaultEnabled
}

func (c *Client) shouldShowPinned() bool {
	if c.cfg.Filters == nil || c.cfg.Filters.ShowPinnedMessages == nil {
		return true // Default: show pinned
	}
	return *c.cfg.Filters.ShowPinnedMessages
}

// Cache methods

func (c *Client) cacheEntities(users []tg.UserClass, chats []tg.ChatClass) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			name := user.FirstName
			if user.LastName != "" {
				name += " " + user.LastName
			}
			if name == "" {
				name = user.Username
			}
			c.userCache[user.ID] = name
		}
	}

	for _, ch := range chats {
		switch chat := ch.(type) {
		case *tg.Chat:
			c.chatCache[chat.ID] = chat.Title
		case *tg.Channel:
			c.chatCache[chat.ID] = chat.Title
		}
	}
}

func (c *Client) getUserName(userID int64) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if name, ok := c.userCache[userID]; ok {
		return name
	}
	return fmt.Sprintf("User %d", userID)
}

func (c *Client) getChatName(chatID int64) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if name, ok := c.chatCache[chatID]; ok {
		return name
	}
	return fmt.Sprintf("Chat %d", chatID)
}

func (c *Client) getChannelName(channelID int64) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if name, ok := c.chatCache[channelID]; ok {
		return name
	}
	return fmt.Sprintf("Channel %d", channelID)
}

// guiAuth implements auth.UserAuthenticator with GUI dialogs
type guiAuth struct {
	phone string
}

func (a guiAuth) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a guiAuth) Password(_ context.Context) (string, error) {
	password, ok := dialog.InputBox(
		"Telegram 2FA",
		"Enter your Two-Factor Authentication password:",
		true, // masked
	)
	if !ok {
		return "", fmt.Errorf("authentication cancelled by user")
	}
	return password, nil
}

func (a guiAuth) Code(_ context.Context, sentCode *tg.AuthSentCode) (string, error) {
	prompt := "Enter the verification code sent to your Telegram:"

	// Provide more context if available
	switch t := sentCode.Type.(type) {
	case *tg.AuthSentCodeTypeApp:
		prompt = fmt.Sprintf("Enter the %d-digit code from your Telegram app:", t.Length)
	case *tg.AuthSentCodeTypeSMS:
		prompt = fmt.Sprintf("Enter the %d-digit code sent via SMS:", t.Length)
	case *tg.AuthSentCodeTypeCall:
		prompt = fmt.Sprintf("Enter the %d-digit code from the phone call:", t.Length)
	}

	code, ok := dialog.InputBox("Telegram Verification", prompt, false)
	if !ok {
		return "", fmt.Errorf("authentication cancelled by user")
	}
	return code, nil
}

func (a guiAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up not supported - please create a Telegram account first")
}

func (a guiAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return nil
}
