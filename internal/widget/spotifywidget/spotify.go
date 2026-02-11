// Package spotifywidget implements a media player widget for Spotify
// using the Spotify Web API with OAuth PKCE authentication.
package spotifywidget

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/anim"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/spotify"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
)

func init() {
	widget.Register("spotify", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Placeholder mode constants
const (
	placeholderModeText = "text"
	placeholderModeIcon = "icon"
	placeholderModeHide = "hide"
)

// Error backoff constants
const (
	maxConsecutiveErrors = 5
	errorBackoffDuration = 30 * time.Second
)

// Widget displays information from Spotify
type Widget struct {
	*widget.BaseWidget

	// Configuration
	format          string
	fontSize        int
	fontName        string
	horizAlign      config.HAlign
	vertAlign       config.VAlign
	padding         int
	placeholderMode string
	placeholderText string

	// Scroll settings
	scrollEnabled bool
	scrollGap     int

	// Auto-show event flags
	autoShowOnTrackChange bool
	autoShowOnPlay        bool
	autoShowOnPause       bool
	autoShowOnStop        bool
	autoShowDuration      time.Duration

	// Runtime state
	client        spotify.Client
	fontFace      font.Face
	currentText   string
	previousTrack string // track ID for track change detection
	previousState spotify.PlaybackState
	mu            sync.RWMutex

	// Error handling
	consecutiveErrors int
	errorBackoffUntil time.Time
	lastError         error

	// Auth state
	authStarted bool
	authCtx     context.Context
	authCancel  context.CancelFunc

	// Shared scroller
	scroller *anim.TextScroller
}

// New creates a new Spotify widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Validate required auth config
	if cfg.SpotifyAuth == nil {
		return nil, fmt.Errorf("spotify_auth configuration is required")
	}
	if cfg.SpotifyAuth.ClientID == "" {
		return nil, fmt.Errorf("spotify_auth.client_id is required")
	}

	// Extract text settings using helper
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Use larger default font for media player
	fontSize := textSettings.FontSize
	if fontSize == 10 { // default value
		fontSize = 12
	}

	// Extract format from text.format
	format := "{artist} - {title}"
	if cfg.Text != nil && cfg.Text.Format != "" {
		format = cfg.Text.Format
	}

	// Extract spotify-specific settings
	placeholderMode := placeholderModeText
	placeholderText := "[Not playing]"

	if cfg.Spotify != nil && cfg.Spotify.Placeholder != nil {
		if cfg.Spotify.Placeholder.Mode != "" {
			placeholderMode = cfg.Spotify.Placeholder.Mode
		}
		if cfg.Spotify.Placeholder.Text != "" {
			placeholderText = cfg.Spotify.Placeholder.Text
		}
	}

	// Auto-show defaults: only track change is enabled by default
	autoShowOnTrackChange := true
	autoShowOnPlay := false
	autoShowOnPause := false
	autoShowOnStop := false
	autoShowDuration := 5 * time.Second

	// Extract auto-show settings
	if cfg.SpotifyAutoShow != nil {
		if cfg.SpotifyAutoShow.OnTrackChange != nil {
			autoShowOnTrackChange = *cfg.SpotifyAutoShow.OnTrackChange
		}
		autoShowOnPlay = cfg.SpotifyAutoShow.OnPlay
		autoShowOnPause = cfg.SpotifyAutoShow.OnPause
		autoShowOnStop = cfg.SpotifyAutoShow.OnStop
		if cfg.SpotifyAutoShow.DurationSec > 0 {
			autoShowDuration = time.Duration(cfg.SpotifyAutoShow.DurationSec * float64(time.Second))
		}
	}

	// Extract scroll settings
	scrollEnabled := false
	scrollDirection := anim.ScrollLeft
	scrollSpeed := 30.0 // pixels per second
	scrollMode := anim.ScrollContinuous
	scrollPauseMs := 1000
	scrollGap := 20

	if cfg.Scroll != nil {
		scrollEnabled = cfg.Scroll.Enabled
		if cfg.Scroll.Direction != "" {
			scrollDirection = cfg.Scroll.Direction
		}
		if cfg.Scroll.Speed > 0 {
			scrollSpeed = cfg.Scroll.Speed
		}
		if cfg.Scroll.Mode != "" {
			scrollMode = cfg.Scroll.Mode
		}
		if cfg.Scroll.PauseMs > 0 {
			scrollPauseMs = cfg.Scroll.PauseMs
		}
		if cfg.Scroll.Gap > 0 {
			scrollGap = cfg.Scroll.Gap
		}
	}

	// Load font
	fontFace, err := bitmap.LoadFont(textSettings.FontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Create Spotify client
	authMode := spotify.AuthModeOAuth
	if cfg.SpotifyAuth.Mode == "manual" {
		authMode = spotify.AuthModeManual
	}

	clientCfg := &spotify.ClientConfig{
		ClientID:     cfg.SpotifyAuth.ClientID,
		AuthMode:     authMode,
		AccessToken:  cfg.SpotifyAuth.AccessToken,
		RefreshToken: cfg.SpotifyAuth.RefreshToken,
		TokenPath:    cfg.SpotifyAuth.TokenPath,
		CallbackPort: cfg.SpotifyAuth.CallbackPort,
	}

	client, err := spotify.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spotify client: %w", err)
	}

	// Create scroller with configuration
	scroller := anim.NewTextScroller(anim.ScrollerConfig{
		Speed:     scrollSpeed,
		Mode:      scrollMode,
		Direction: scrollDirection,
		Gap:       scrollGap,
		PauseMs:   scrollPauseMs,
	})

	return &Widget{
		BaseWidget:            base,
		format:                format,
		fontSize:              fontSize,
		fontName:              textSettings.FontName,
		horizAlign:            textSettings.HorizAlign,
		vertAlign:             textSettings.VertAlign,
		padding:               padding,
		placeholderMode:       placeholderMode,
		placeholderText:       placeholderText,
		scrollEnabled:         scrollEnabled,
		scrollGap:             scrollGap,
		autoShowOnTrackChange: autoShowOnTrackChange,
		autoShowOnPlay:        autoShowOnPlay,
		autoShowOnPause:       autoShowOnPause,
		autoShowOnStop:        autoShowOnStop,
		autoShowDuration:      autoShowDuration,
		client:                client,
		fontFace:              fontFace,
		scroller:              scroller,
		previousState:         spotify.StateStopped,
	}, nil
}

// Update fetches current track information from Spotify
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Update scroll position if scrolling is enabled and we have text
	if w.scrollEnabled && w.currentText != "" {
		pos := w.GetPosition()
		contentWidth := pos.W - w.padding*2
		textWidth, _ := bitmap.SmartMeasureText(w.currentText, w.fontFace, w.fontName)
		w.scroller.Update(textWidth, contentWidth)
	}

	// Skip if in error backoff
	if time.Now().Before(w.errorBackoffUntil) {
		return nil
	}

	// Check if authentication is needed
	if w.client.NeedsAuth() {
		// Start auth flow if not already started
		if !w.authStarted {
			w.startAuthFlow()
		}
		return nil
	}

	// Get player state
	state, err := w.client.GetState()
	if err != nil {
		w.handleError(err)
		return nil
	}

	// Reset error counter on success
	w.consecutiveErrors = 0
	w.lastError = nil

	// Handle nil state (no active device)
	if state == nil {
		w.handleStateChange(spotify.StateStopped, nil)
		return nil
	}

	w.handleStateChange(state.State, state)

	return nil
}

// handleError handles API errors with backoff logic
func (w *Widget) handleError(err error) {
	w.consecutiveErrors++
	w.lastError = err

	if w.consecutiveErrors >= maxConsecutiveErrors {
		w.errorBackoffUntil = time.Now().Add(errorBackoffDuration)
		log.Printf("spotify: entering error backoff after %d errors: %v", w.consecutiveErrors, err)
	}
}

// handleStateChange processes state changes and triggers auto-show
func (w *Widget) handleStateChange(newState spotify.PlaybackState, state *spotify.PlayerState) {
	stateChanged := newState != w.previousState

	// Check for play event
	if w.autoShowOnPlay && stateChanged && newState == spotify.StatePlaying {
		w.TriggerAutoHide()
	}

	// Check for pause event
	if w.autoShowOnPause && stateChanged && newState == spotify.StatePaused {
		w.TriggerAutoHide()
	}

	// Check for stop event
	if w.autoShowOnStop && stateChanged && newState == spotify.StateStopped {
		w.TriggerAutoHide()
	}

	// Update previous state
	w.previousState = newState

	// Handle stopped state or no track
	if newState == spotify.StateStopped || state == nil || state.Track == nil {
		w.currentText = ""
		return
	}

	// Format output string
	w.currentText = w.formatOutput(state)

	// Check for track change
	trackKey := state.Track.ID
	if trackKey != w.previousTrack {
		if w.previousTrack != "" && !stateChanged {
			// Reset scroll position on track change
			w.scroller.Reset()
			// Trigger auto-show if enabled
			if w.autoShowOnTrackChange {
				w.TriggerAutoHide()
			}
		}
		w.previousTrack = trackKey
	}
}

// startAuthFlow starts the OAuth authentication flow in the background
func (w *Widget) startAuthFlow() {
	w.authStarted = true
	w.authCtx, w.authCancel = context.WithTimeout(context.Background(), 5*time.Minute)

	go func() {
		log.Println("spotify: starting OAuth authentication flow...")
		if err := w.client.StartAuth(w.authCtx); err != nil {
			log.Printf("spotify: OAuth flow failed: %v", err)
			w.mu.Lock()
			w.authStarted = false
			w.mu.Unlock()
		} else {
			log.Println("spotify: OAuth authentication successful")
		}
	}()
}

// formatOutput replaces placeholders with actual values
func (w *Widget) formatOutput(state *spotify.PlayerState) string {
	if state == nil || state.Track == nil {
		return ""
	}

	track := state.Track

	// Format duration and position
	position := formatDuration(track.Position)
	duration := formatDuration(track.Duration)

	// Format artists
	artist := ""
	artists := ""
	if len(track.Artists) > 0 {
		artist = track.Artists[0]
		artists = strings.Join(track.Artists, ", ")
	}

	// Use TokenFormatter for placeholder replacement
	formatter := render.NewTokenFormatter().
		Set("artist", artist).
		Set("artists", artists).
		Set("title", track.Name).
		Set("album", track.Album).
		Set("position", position).
		Set("duration", duration).
		Set("state", state.State.String()).
		Set("device", state.DeviceName).
		Set("volume", fmt.Sprintf("%d", state.Volume))

	return formatter.Format(w.format)
}

// formatDuration converts time.Duration to MM:SS format
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "--:--"
	}
	totalSeconds := int(d.Seconds())
	mins := totalSeconds / 60
	secs := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// Render creates an image of the widget
func (w *Widget) Render() (image.Image, error) {
	// Check auto-hide
	if w.ShouldHide() {
		return nil, nil
	}

	// Create canvas with background and border
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	w.mu.RLock()
	currentText := w.currentText
	scrollOffset := w.scroller.GetOffset()
	needsAuth := w.client.NeedsAuth()
	lastError := w.lastError
	w.mu.RUnlock()

	// Show auth required message
	if needsAuth {
		w.renderAuthRequired(img)
		return img, nil
	}

	// Show error if in error state
	if lastError != nil && currentText == "" {
		w.renderError(img, lastError)
		return img, nil
	}

	// Check if Spotify is playing
	if currentText == "" {
		w.renderPlaceholder(img)
		return img, nil
	}

	// Render scrolling or static text
	if w.scrollEnabled {
		w.renderScrollingText(img, currentText, scrollOffset)
	} else {
		bitmap.SmartDrawAlignedText(img, currentText, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}

	return img, nil
}

// renderAuthRequired renders the auth required message with icon if there's space
func (w *Widget) renderAuthRequired(img *image.Gray) {
	pos := w.GetPosition()
	contentH := pos.H - w.padding*2

	// Choose icon size based on available height
	var iconSize int
	var iconName string
	if contentH >= 32 {
		iconSize = 32
		iconName = "logo"
	} else if contentH >= 16 {
		iconSize = 16
		iconName = "logo_16"
	} else if contentH >= 12 {
		iconSize = 12
		iconName = "logo_12"
	}

	// If we can fit an icon, show icon + text
	if iconSize > 0 {
		// Calculate layout
		iconX := w.padding + iconSize/2
		iconY := pos.H / 2

		// Draw icon
		w.renderSpotifyIconByName(img, iconName, iconX, iconY)

		// Draw "Not connected" text to the right
		textX := w.padding + iconSize + 4
		textW := pos.W - textX - w.padding

		if textW > 20 { // Only show text if there's reasonable space
			_, fontHeight := bitmap.SmartMeasureText("Not connected", w.fontFace, w.fontName)
			textY := (pos.H - fontHeight) / 2

			bitmap.SmartDrawTextAtPosition(img, "Not connected", w.fontFace, w.fontName,
				textX, textY, textX, 0, textW, pos.H)
		}
	} else {
		// Fallback to text only
		bitmap.SmartDrawAlignedText(img, "Not connected", w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}
}

// renderError renders an error message
func (w *Widget) renderError(img *image.Gray, err error) {
	errMsg := err.Error()
	if len(errMsg) > 30 {
		errMsg = errMsg[:30] + "..."
	}
	bitmap.SmartDrawAlignedText(img, errMsg, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
}

// renderPlaceholder renders the placeholder when Spotify is not playing
func (w *Widget) renderPlaceholder(img *image.Gray) {
	switch w.placeholderMode {
	case placeholderModeHide:
		// Don't render anything
		return
	case placeholderModeIcon:
		// Draw Spotify icon (simple representation)
		w.renderSpotifyIcon(img)
	case placeholderModeText:
		bitmap.SmartDrawAlignedText(img, w.placeholderText, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
	}
}

// renderSpotifyIcon draws the Spotify icon centered in widget, choosing best size
func (w *Widget) renderSpotifyIcon(img *image.Gray) {
	pos := w.GetPosition()
	contentH := pos.H - w.padding*2

	// Choose appropriate icon size
	var iconName string
	if contentH >= 32 {
		iconName = "logo"
	} else if contentH >= 16 {
		iconName = "logo_16"
	} else {
		iconName = "logo_12"
	}

	w.renderSpotifyIconByName(img, iconName, pos.W/2, pos.H/2)
}

// renderSpotifyIconByName draws a specific Spotify icon at the given center position
func (w *Widget) renderSpotifyIconByName(img *image.Gray, iconName string, centerX, centerY int) {
	pos := w.GetPosition()
	white := color.Gray{Y: 255}

	// Get the requested icon
	logo := glyphs.SpotifyIcons.Icons[iconName]
	if logo == nil {
		// Fallback to default
		logo = glyphs.SpotifyIcons.Icons["logo"]
		if logo == nil {
			return
		}
	}

	// Calculate top-left position to center the icon
	startX := centerX - logo.Width/2
	startY := centerY - logo.Height/2

	// Draw the glyph pixel by pixel
	for y := 0; y < logo.Height; y++ {
		for x := 0; x < logo.Width; x++ {
			if logo.Data[y][x] {
				px := startX + x
				py := startY + y
				if px >= 0 && px < pos.W && py >= 0 && py < pos.H {
					img.SetGray(px, py, white)
				}
			}
		}
	}
}

// renderScrollingText renders text with scroll offset
func (w *Widget) renderScrollingText(img *image.Gray, text string, offset float64) {
	pos := w.GetPosition()
	contentX := w.padding
	contentY := w.padding
	contentW := pos.W - w.padding*2
	contentH := pos.H - w.padding*2

	textWidth, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)

	// If text fits, just draw it normally
	if textWidth <= contentW {
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
		return
	}

	// Calculate aligned position using bitmap helper
	textX, textY := bitmap.SmartCalculateTextPosition(text, w.fontFace, w.fontName, contentX, contentY, contentW, contentH, w.horizAlign, w.vertAlign)

	// Get scroller configuration
	scrollCfg := w.scroller.GetConfig()

	// Handle horizontal scrolling
	if w.scroller.IsHorizontal() {
		// Apply horizontal scroll offset
		scrollX := textX - int(offset)

		// For continuous mode, draw text twice for seamless loop
		if scrollCfg.Mode == anim.ScrollContinuous {
			bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, scrollX, textY, contentX, contentY, contentW, contentH)

			// Draw second instance for seamless loop
			if scrollCfg.Direction == anim.ScrollLeft {
				textX2 := scrollX + textWidth + w.scrollGap
				if textX2 < contentX+contentW {
					bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX2, textY, contentX, contentY, contentW, contentH)
				}
			} else {
				textX2 := scrollX - textWidth - w.scrollGap
				if textX2+textWidth > contentX {
					bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX2, textY, contentX, contentY, contentW, contentH)
				}
			}
		} else {
			// For bounce and pause_ends modes, just draw once
			bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, scrollX, textY, contentX, contentY, contentW, contentH)
		}
	} else {
		// Vertical scrolling - apply offset to Y
		scrollY := textY - int(offset)
		bitmap.SmartDrawTextAtPosition(img, text, w.fontFace, w.fontName, textX, scrollY, contentX, contentY, contentW, contentH)
	}
}

// Stop cleans up resources
func (w *Widget) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.authCancel != nil {
		w.authCancel()
	}
}
