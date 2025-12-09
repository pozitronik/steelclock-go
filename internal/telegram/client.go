// Package telegram provides a Telegram client wrapper for receiving notifications
package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	ID            int
	ChatID        int64
	ChatType      ChatType
	ChatTitle     string
	SenderName    string
	Text          string
	MediaType     string // "photo", "video", "audio", "voice", "sticker", "document", "contact", "location", "poll", etc.
	Time          time.Time
	IsPinned      bool
	IsOutgoing    bool
	UnreadCount   int
	IsForwarded   bool   // True if this is a forwarded message
	ForwardedFrom string // Original sender name for forwarded messages
}

// ChatType represents the type of chat
type ChatType int

const (
	ChatTypePrivate ChatType = iota
	ChatTypeGroup
	ChatTypeChannel
)

// ClientConfig holds configuration for the Telegram client
type ClientConfig struct {
	Auth    *config.TelegramAuthConfig
	Filters *config.TelegramFiltersConfig
}

// Client wraps the Telegram client with notification handling
type Client struct {
	mu sync.RWMutex

	auth        *config.TelegramAuthConfig
	filters     *config.TelegramFiltersConfig
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
	unreadCount   int // Total unread messages count

	// Detailed unread stats
	unreadStats UnreadStats

	// Callbacks (support multiple listeners for shared client)
	onMessageCallbacks []func(MessageInfo)
	onErrorCallbacks   []func(error)
	nextCallbackID     int
	callbackIDs        map[int]bool // tracks valid callback IDs

	// Chat cache for resolving names
	chatCache map[int64]string
	userCache map[int64]string
}

// UnreadStats contains detailed unread message statistics
type UnreadStats struct {
	Total         int // Total unread messages
	Mentions      int // Unread @mentions
	Reactions     int // Unread reactions
	Private       int // Unread in private chats
	Groups        int // Unread in groups
	Channels      int // Unread in channels
	Muted         int // Unread in muted chats
	PrivateMuted  int // Unread in muted private chats
	GroupsMuted   int // Unread in muted groups
	ChannelsMuted int // Unread in muted channels
}

// NewClient creates a new Telegram client
func NewClient(cfg *ClientConfig) (*Client, error) {
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
	sessionPath := cfg.Auth.SessionPath
	if sessionPath == "" {
		// Default: telegram/{api_id}_{phone}.session in executable directory
		exePath, err := os.Executable()
		if err != nil {
			exePath = "."
		}
		exeDir := filepath.Dir(exePath)
		// Sanitize phone number for filename
		phone := strings.ReplaceAll(cfg.Auth.PhoneNumber, "+", "")
		phone = strings.ReplaceAll(phone, " ", "")
		// Include api_id to avoid conflicts between different accounts
		sessionPath = filepath.Join(exeDir, "telegram", fmt.Sprintf("%d_%s.session", cfg.Auth.APIID, phone))
	}

	// Ensure session directory exists
	sessionDir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	maxMessages := 10

	c := &Client{
		auth:        cfg.Auth,
		filters:     cfg.Filters,
		sessionPath: sessionPath,
		maxMessages: maxMessages,
		messages:    make([]MessageInfo, 0, maxMessages),
		chatCache:   make(map[int64]string),
		userCache:   make(map[int64]string),
		callbackIDs: make(map[int]bool),
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
	c.client = telegram.NewClient(c.auth.APIID, c.auth.APIHash, telegram.Options{
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
					return err
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

		if err != nil && !errors.Is(err, context.Canceled) {
			c.notifyError(err)
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
	log.Printf("Telegram: Starting authentication for %s", c.auth.PhoneNumber)

	// Create GUI auth flow
	flow := auth.NewFlow(
		guiAuth{phone: c.auth.PhoneNumber},
		auth.SendCodeOptions{},
	)

	if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
		log.Printf("Telegram: Authentication error: %v", err)
		return err
	}

	log.Printf("Telegram: Authentication successful")
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
	return c.unreadCount
}

// GetUnreadStats returns detailed unread statistics
func (c *Client) GetUnreadStats() UnreadStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.unreadStats
}

// FetchUnreadCount fetches the current unread count from Telegram dialogs
func (c *Client) FetchUnreadCount() error {
	c.mu.RLock()
	api := c.api
	ctx := c.ctx
	c.mu.RUnlock()

	if api == nil || ctx == nil {
		return fmt.Errorf("client not connected")
	}

	// Fetch dialogs to get unread counts
	result, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetDate: 0,
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100, // Get up to 100 dialogs
		Hash:       0,
	})
	if err != nil {
		return fmt.Errorf("failed to get dialogs: %w", err)
	}

	totalUnread := 0
	stats := UnreadStats{}

	switch dialogs := result.(type) {
	case *tg.MessagesDialogs:
		for _, d := range dialogs.Dialogs {
			unread, dialogStats := c.countUnreadFromDialog(d)
			totalUnread += unread
			c.mergeStats(&stats, dialogStats)
		}
	case *tg.MessagesDialogsSlice:
		for _, d := range dialogs.Dialogs {
			unread, dialogStats := c.countUnreadFromDialog(d)
			totalUnread += unread
			c.mergeStats(&stats, dialogStats)
		}
	}

	stats.Total = totalUnread

	c.mu.Lock()
	c.unreadCount = totalUnread
	c.unreadStats = stats
	c.mu.Unlock()

	return nil
}

// mergeStats merges dialog stats into the total stats
func (c *Client) mergeStats(total *UnreadStats, dialog UnreadStats) {
	total.Mentions += dialog.Mentions
	total.Reactions += dialog.Reactions
	total.Private += dialog.Private
	total.Groups += dialog.Groups
	total.Channels += dialog.Channels
	total.Muted += dialog.Muted
	total.PrivateMuted += dialog.PrivateMuted
	total.GroupsMuted += dialog.GroupsMuted
	total.ChannelsMuted += dialog.ChannelsMuted
}

// countUnreadFromDialog extracts unread count from a dialog based on filters
// Returns total unread count (filtered) and detailed stats (unfiltered for stats)
func (c *Client) countUnreadFromDialog(d tg.DialogClass) (int, UnreadStats) {
	dialogInstance, ok := d.(*tg.Dialog)
	if !ok {
		return 0, UnreadStats{}
	}

	stats := UnreadStats{}
	unread := dialogInstance.UnreadCount
	mentions := dialogInstance.UnreadMentionsCount
	reactions := dialogInstance.UnreadReactionsCount

	// Check mute status from notify settings
	isMuted := dialogInstance.NotifySettings.MuteUntil > 0

	// Categorize by chat type and collect stats
	var filtered bool
	switch peer := dialogInstance.Peer.(type) {
	case *tg.PeerUser:
		stats.Private = unread
		stats.Mentions += mentions
		stats.Reactions += reactions
		if isMuted {
			stats.Muted += unread
			stats.PrivateMuted = unread
		}
		filtered = !c.shouldShowPrivate(peer.UserID)
	case *tg.PeerChat:
		stats.Groups = unread
		stats.Mentions += mentions
		stats.Reactions += reactions
		if isMuted {
			stats.Muted += unread
			stats.GroupsMuted = unread
		}
		filtered = !c.shouldShowGroup(peer.ChatID)
	case *tg.PeerChannel:
		stats.Channels = unread
		stats.Mentions += mentions
		stats.Reactions += reactions
		if isMuted {
			stats.Muted += unread
			stats.ChannelsMuted = unread
		}
		filtered = !c.shouldShowChannel(peer.ChannelID)
	default:
		return 0, stats
	}

	if filtered {
		return 0, stats
	}

	return unread, stats
}

// AddMessageCallback adds a callback for new messages and returns an ID for removal.
// Multiple callbacks can be registered; all will be called when a message arrives.
func (c *Client) AddMessageCallback(fn func(MessageInfo)) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextCallbackID
	c.nextCallbackID++
	c.callbackIDs[id] = true
	c.onMessageCallbacks = append(c.onMessageCallbacks, fn)

	return id
}

// AddErrorCallback adds a callback for errors and returns an ID for removal.
// Multiple callbacks can be registered; all will be called when an error occurs.
func (c *Client) AddErrorCallback(fn func(error)) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextCallbackID
	c.nextCallbackID++
	c.callbackIDs[id] = true
	c.onErrorCallbacks = append(c.onErrorCallbacks, fn)

	return id
}

// SetMessageCallback sets the callback for new messages (for backward compatibility).
// Deprecated: Use AddMessageCallback instead for proper multi-widget support.
func (c *Client) SetMessageCallback(fn func(MessageInfo)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Clear existing callbacks and add the new one
	c.onMessageCallbacks = []func(MessageInfo){fn}
}

// SetErrorCallback sets the callback for errors (for backward compatibility).
// Deprecated: Use AddErrorCallback instead for proper multi-widget support.
func (c *Client) SetErrorCallback(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Clear existing callbacks and add the new one
	c.onErrorCallbacks = []func(error){fn}
}

// Handle implements telegram.UpdateHandler
func (c *Client) Handle(_ context.Context, u tg.UpdatesClass) error {
	switch updates := u.(type) {
	case *tg.Updates:
		for _, update := range updates.Updates {
			c.handleUpdate(update, updates.Users, updates.Chats)
		}
	case *tg.UpdatesCombined:
		for _, update := range updates.Updates {
			c.handleUpdate(update, updates.Users, updates.Chats)
		}
	case *tg.UpdateShort:
		c.handleUpdate(updates.Update, nil, nil)
	case *tg.UpdateShortMessage:
		c.handleShortMessage(updates)
	case *tg.UpdateShortChatMessage:
		c.handleShortChatMessage(updates)
	}
	return nil
}

// handleUpdate processes a single update
func (c *Client) handleUpdate(update tg.UpdateClass, users []tg.UserClass, chats []tg.ChatClass) {
	// Cache users and chats
	c.cacheEntities(users, chats)

	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		c.processMessage(u.Message)
	case *tg.UpdateNewChannelMessage:
		c.processMessage(u.Message)
	case *tg.UpdatePinnedMessages:
		// Handle pinned message notification - check filter in handlePinnedUpdate
		c.handlePinnedUpdate(u.Peer, u.Messages)
	case *tg.UpdatePinnedChannelMessages:
		// Check if channel pinned messages should be shown
		if c.shouldShowPinned(ChatTypeChannel) {
			c.handlePinnedChannelUpdate(u.ChannelID, u.Messages)
		}
	}
}

// extractForwardInfo extracts forwarded message info from a GetFwdFrom function
func (c *Client) extractForwardInfo(getFwdFrom func() (tg.MessageFwdHeader, bool)) (bool, string) {
	fwdFrom, ok := getFwdFrom()
	if !ok {
		return false, ""
	}

	forwardedFrom := ""
	// Try to get original sender name
	if fromName, nameOk := fwdFrom.GetFromName(); nameOk && fromName != "" {
		// Privacy-hidden forward - use provided name
		forwardedFrom = fromName
	} else if fromID, idOk := fwdFrom.GetFromID(); idOk {
		// Have original sender ID
		switch from := fromID.(type) {
		case *tg.PeerUser:
			forwardedFrom = c.getUserName(from.UserID)
		case *tg.PeerChannel:
			forwardedFrom = c.getChannelName(from.ChannelID)
		}
	}
	// Try post author for channels
	if forwardedFrom == "" {
		if postAuthor, authorOk := fwdFrom.GetPostAuthor(); authorOk && postAuthor != "" {
			forwardedFrom = postAuthor
		}
	}

	return true, forwardedFrom
}

// handleShortMessage handles UpdateShortMessage (private message optimization)
func (c *Client) handleShortMessage(u *tg.UpdateShortMessage) {
	if !c.shouldShowPrivate(u.UserID) {
		return
	}

	senderID := u.UserID
	if u.Out {
		senderID = c.selfID
	}

	// Check for forwarded message
	isForwarded, forwardedFrom := c.extractForwardInfo(u.GetFwdFrom)

	msg := MessageInfo{
		ID:            u.ID,
		ChatID:        u.UserID,
		ChatType:      ChatTypePrivate,
		ChatTitle:     c.getUserName(u.UserID),
		SenderName:    c.getUserName(senderID),
		Text:          u.Message,
		Time:          time.Unix(int64(u.Date), 0),
		IsOutgoing:    u.Out,
		IsForwarded:   isForwarded,
		ForwardedFrom: forwardedFrom,
	}

	c.addMessage(msg)
}

// handleShortChatMessage handles UpdateShortChatMessage (group message optimization)
func (c *Client) handleShortChatMessage(u *tg.UpdateShortChatMessage) {
	if !c.shouldShowGroup(u.ChatID) {
		return
	}

	// Check for forwarded message
	isForwarded, forwardedFrom := c.extractForwardInfo(u.GetFwdFrom)

	msg := MessageInfo{
		ID:            u.ID,
		ChatID:        u.ChatID,
		ChatType:      ChatTypeGroup,
		ChatTitle:     c.getChatName(u.ChatID),
		SenderName:    c.getUserName(u.FromID),
		Text:          u.Message,
		Time:          time.Unix(int64(u.Date), 0),
		IsOutgoing:    u.Out,
		IsForwarded:   isForwarded,
		ForwardedFrom: forwardedFrom,
	}

	c.addMessage(msg)
}

// processMessage processes a full message object
func (c *Client) processMessage(msg tg.MessageClass) {
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

	// Check for forwarded message
	isForwarded, forwardedFrom := c.extractForwardInfo(message.GetFwdFrom)

	info := MessageInfo{
		ID:            message.ID,
		ChatID:        chatID,
		ChatType:      chatType,
		ChatTitle:     chatTitle,
		SenderName:    senderName,
		Text:          message.Message,
		MediaType:     getMediaType(message.Media),
		Time:          time.Unix(int64(message.Date), 0),
		IsPinned:      message.Pinned,
		IsOutgoing:    message.Out,
		IsForwarded:   isForwarded,
		ForwardedFrom: forwardedFrom,
	}

	c.addMessage(info)
}

// handlePinnedUpdate handles pinned message updates
func (c *Client) handlePinnedUpdate(peer tg.PeerClass, _ []int) {
	var chatID int64
	var chatType ChatType
	var chatTitle string

	switch p := peer.(type) {
	case *tg.PeerUser:
		if !c.shouldShowPrivate(p.UserID) || !c.shouldShowPinned(ChatTypePrivate) {
			return
		}
		chatID = p.UserID
		chatType = ChatTypePrivate
		chatTitle = c.getUserName(p.UserID)
	case *tg.PeerChat:
		if !c.shouldShowGroup(p.ChatID) || !c.shouldShowPinned(ChatTypeGroup) {
			return
		}
		chatID = p.ChatID
		chatType = ChatTypeGroup
		chatTitle = c.getChatName(p.ChatID)
	default:
		return
	}

	msg := MessageInfo{
		ChatID:    chatID,
		ChatType:  chatType,
		ChatTitle: chatTitle,
		Text:      "[Message pinned]",
		Time:      time.Now(),
		IsPinned:  true,
	}

	c.addMessage(msg)
}

// handlePinnedChannelUpdate handles channel pinned message updates
func (c *Client) handlePinnedChannelUpdate(channelID int64, _ []int) {
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

	// Copy callbacks to avoid holding lock during callback execution
	callbacks := make([]func(MessageInfo), len(c.onMessageCallbacks))
	copy(callbacks, c.onMessageCallbacks)
	c.mu.Unlock()

	// Call all registered callbacks
	for _, callback := range callbacks {
		if callback != nil {
			callback(msg)
		}
	}
}

// notifyError calls all registered error callbacks
func (c *Client) notifyError(err error) {
	c.mu.RLock()
	callbacks := make([]func(error), len(c.onErrorCallbacks))
	copy(callbacks, c.onErrorCallbacks)
	c.mu.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(err)
		}
	}
}

// Filter methods

func (c *Client) shouldShowPrivate(userID int64) bool {
	if c.filters == nil {
		return true // Default: show private chats
	}
	return c.shouldShow(c.filters.PrivateChats, userID, true)
}

func (c *Client) shouldShowGroup(chatID int64) bool {
	if c.filters == nil {
		return false // Default: don't show groups
	}
	return c.shouldShow(c.filters.Groups, chatID, false)
}

func (c *Client) shouldShowChannel(channelID int64) bool {
	if c.filters == nil {
		return false // Default: don't show channels
	}
	return c.shouldShow(c.filters.Channels, channelID, false)
}

func (c *Client) shouldShow(chatCfg *config.TelegramChatFilterConfig, id int64, defaultEnabled bool) bool {
	if chatCfg == nil {
		return defaultEnabled
	}

	idStr := fmt.Sprintf("%d", id)

	// Blacklist has the highest priority - never show these
	for _, item := range chatCfg.Blacklist {
		if item == idStr {
			return false
		}
	}

	// Whitelist overrides enabled setting - always show these
	for _, item := range chatCfg.Whitelist {
		if item == idStr {
			return true
		}
	}

	// Check if this chat type is enabled
	if chatCfg.Enabled != nil {
		return *chatCfg.Enabled
	}

	return defaultEnabled
}

func (c *Client) shouldShowPinned(chatType ChatType) bool {
	if c.filters == nil {
		// No filters configured - use defaults
		switch chatType {
		case ChatTypeChannel:
			return false
		default:
			return true
		}
	}

	var chatCfg *config.TelegramChatFilterConfig
	var defaultVal bool

	switch chatType {
	case ChatTypePrivate:
		chatCfg = c.filters.PrivateChats
		defaultVal = true
	case ChatTypeGroup:
		chatCfg = c.filters.Groups
		defaultVal = true
	case ChatTypeChannel:
		chatCfg = c.filters.Channels
		defaultVal = false
	default:
		return true
	}

	if chatCfg == nil || chatCfg.PinnedMessages == nil {
		return defaultVal
	}
	return *chatCfg.PinnedMessages
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
	name, ok := c.userCache[userID]
	c.mu.RUnlock()

	if ok {
		return name
	}

	// Try to fetch user info from API
	if c.api != nil && c.ctx != nil {
		name = c.fetchUserName(userID)
		if name != "" {
			c.mu.Lock()
			c.userCache[userID] = name
			c.mu.Unlock()
			return name
		}
	}

	return fmt.Sprintf("User %d", userID)
}

// fetchUserName fetches user info from Telegram API
func (c *Client) fetchUserName(userID int64) string {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	users, err := c.api.UsersGetUsers(ctx, []tg.InputUserClass{
		&tg.InputUser{UserID: userID},
	})
	if err != nil {
		log.Printf("Telegram: Failed to fetch user %d: %v", userID, err)
		return ""
	}

	for _, u := range users {
		if user, ok := u.(*tg.User); ok && user.ID == userID {
			name := user.FirstName
			if user.LastName != "" {
				name += " " + user.LastName
			}
			if name == "" {
				name = user.Username
			}
			return name
		}
	}

	return ""
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
	log.Printf("Telegram: Password (2FA) callback called for %s", a.phone)
	password, ok := dialog.InputBox(
		fmt.Sprintf("Telegram 2FA (%s)", a.phone),
		fmt.Sprintf("Enter your Two-Factor Authentication password for account %s:", a.phone),
		true, // masked
	)
	log.Printf("Telegram: Password dialog result: ok=%v", ok)
	if !ok {
		return "", fmt.Errorf("2FA entry cancelled")
	}
	if password == "" {
		return "", fmt.Errorf("empty password entered")
	}
	return password, nil
}

func (a guiAuth) Code(_ context.Context, sentCode *tg.AuthSentCode) (string, error) {
	log.Printf("Telegram: Code callback called for %s", a.phone)
	prompt := fmt.Sprintf("Enter the verification code sent to %s:", a.phone)

	// Provide more context if available
	if sentCode != nil && sentCode.Type != nil {
		switch t := sentCode.Type.(type) {
		case *tg.AuthSentCodeTypeApp:
			prompt = fmt.Sprintf("Enter the %d-digit code from your Telegram app (%s):", t.Length, a.phone)
		case *tg.AuthSentCodeTypeSMS:
			prompt = fmt.Sprintf("Enter the %d-digit code sent via SMS to %s:", t.Length, a.phone)
		case *tg.AuthSentCodeTypeCall:
			prompt = fmt.Sprintf("Enter the %d-digit code from the phone call to %s:", t.Length, a.phone)
		}
	}

	log.Printf("Telegram: Showing code dialog for %s", a.phone)
	code, ok := dialog.InputBox(fmt.Sprintf("Telegram Verification (%s)", a.phone), prompt, false)
	log.Printf("Telegram: Code dialog result: ok=%v, code_len=%d", ok, len(code))
	if !ok {
		return "", fmt.Errorf("code entry cancelled")
	}
	if code == "" {
		return "", fmt.Errorf("empty code entered")
	}
	return code, nil
}

func (a guiAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up not supported - please create a Telegram account first")
}

func (a guiAuth) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

// getMediaType returns a human-readable media type from a message's media
func getMediaType(media tg.MessageMediaClass) string {
	if media == nil {
		return ""
	}

	switch media.(type) {
	case *tg.MessageMediaPhoto:
		return "Photo"
	case *tg.MessageMediaGeo:
		return "Location"
	case *tg.MessageMediaContact:
		return "Contact"
	case *tg.MessageMediaDocument:
		// Documents can be various types - check for specific document types
		doc, ok := media.(*tg.MessageMediaDocument)
		if ok {
			if d, ok := doc.Document.(*tg.Document); ok {
				for _, attr := range d.Attributes {
					switch attr := attr.(type) {
					case *tg.DocumentAttributeSticker:
						return "Sticker"
					case *tg.DocumentAttributeVideo:
						return "Video"
					case *tg.DocumentAttributeAudio:
						if attr.Voice {
							return "Voice message"
						}
						return "Audio"
					case *tg.DocumentAttributeAnimated:
						return "GIF"
					}
				}
			}
		}
		return "Document"
	case *tg.MessageMediaVenue:
		return "Venue"
	case *tg.MessageMediaGame:
		return "Game"
	case *tg.MessageMediaInvoice:
		return "Invoice"
	case *tg.MessageMediaGeoLive:
		return "Live location"
	case *tg.MessageMediaPoll:
		return "Poll"
	case *tg.MessageMediaDice:
		return "Dice"
	case *tg.MessageMediaStory:
		return "Story"
	case *tg.MessageMediaGiveaway:
		return "Giveaway"
	case *tg.MessageMediaGiveawayResults:
		return "Giveaway results"
	case *tg.MessageMediaPaidMedia:
		return "Paid media"
	default:
		return "Media"
	}
}
