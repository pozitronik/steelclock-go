package tray

import (
	_ "embed"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/getlantern/systray"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/webeditor"
)

// Note: runtime is still used by handleEditConfig for runtime.GOOS

//go:embed icon.ico
var iconData []byte

// Manager handles system tray icon and menu
type Manager struct {
	// Legacy mode (single config file)
	configPath string

	// Profile mode
	profileMgr *config.ProfileManager

	// Web editor
	webEditor *webeditor.Server

	// Callbacks
	onReload        func() error
	onExit          func()
	onProfileSwitch func(path string) error

	// Menu items
	profileMenuItems []*systray.MenuItem
	menuEdit         *systray.MenuItem
	menuReload       *systray.MenuItem
	menuExit         *systray.MenuItem

	// State
	readyChan       chan struct{}
	onReadyCallback func()
}

// NewManager creates a new tray manager for single config mode (legacy)
func NewManager(configPath string, onReload func() error, onExit func()) *Manager {
	return &Manager{
		configPath: configPath,
		onReload:   onReload,
		onExit:     onExit,
		readyChan:  make(chan struct{}),
	}
}

// NewManagerWithProfiles creates a new tray manager with profile support
func NewManagerWithProfiles(profileMgr *config.ProfileManager, onReload func() error, onProfileSwitch func(path string) error, onExit func()) *Manager {
	return &Manager{
		profileMgr:      profileMgr,
		onReload:        onReload,
		onProfileSwitch: onProfileSwitch,
		onExit:          onExit,
		readyChan:       make(chan struct{}),
	}
}

// SetWebEditor sets the web editor server for browser-based config editing
func (m *Manager) SetWebEditor(editor *webeditor.Server) {
	m.webEditor = editor
}

// Run starts the system tray
func (m *Manager) Run() {
	systray.Run(m.onReady, m.onQuit)
}

// onReady is called when systray is ready
func (m *Manager) onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("SteelClock")
	systray.SetTooltip("SteelClock - SteelSeries Display")

	if m.profileMgr != nil {
		m.buildProfileMenu()
	} else {
		m.buildLegacyMenu()
	}

	close(m.readyChan)

	if m.onReadyCallback != nil {
		go m.onReadyCallback()
	}

	go m.handleMenuClicks()
}

// buildLegacyMenu creates the legacy single-config menu
func (m *Manager) buildLegacyMenu() {
	m.menuEdit = systray.AddMenuItem("Edit Config", "Open config file in default editor")
	m.menuReload = systray.AddMenuItem("Reload Config", "Reload configuration")
	systray.AddSeparator()
	m.menuExit = systray.AddMenuItem("Exit", "Exit SteelClock")
}

// buildProfileMenu creates the profile-aware menu
func (m *Manager) buildProfileMenu() {
	profiles := m.profileMgr.GetProfiles()
	activeProfile := m.profileMgr.GetActiveProfile()

	// Add profile menu items
	for _, profile := range profiles {
		title := profile.Name
		isActive := activeProfile != nil && profile.Path == activeProfile.Path

		// On Linux, add prefix for active profile (checkmarks don't display with AppIndicator)
		if isActive && runtime.GOOS == "linux" {
			title = "✓ " + title
		}

		menuItem := systray.AddMenuItem(title, profile.Path)

		// Use checkmark (works on Windows/macOS)
		if isActive {
			menuItem.Check()
		}

		m.profileMenuItems = append(m.profileMenuItems, menuItem)
	}

	// Separator after profiles
	if len(profiles) > 0 {
		systray.AddSeparator()
	}

	// Edit and Reload items
	m.menuEdit = systray.AddMenuItem("Edit Active Config", "Open active config file in default editor")
	m.menuReload = systray.AddMenuItem("Reload Active Config", "Reload current configuration")

	// Separator before Exit
	systray.AddSeparator()
	m.menuExit = systray.AddMenuItem("Exit", "Exit SteelClock")
}

// onQuit is called when systray is quitting
func (m *Manager) onQuit() {
	if m.onExit != nil {
		m.onExit()
	}
}

// handleMenuClicks processes menu item clicks
func (m *Manager) handleMenuClicks() {
	// Build select cases once - menu structure doesn't change at runtime
	// Cases: [edit, reload, exit, profile0, profile1, ...]
	cases := make([]reflect.SelectCase, 0, 3+len(m.profileMenuItems))

	// Add fixed menu items
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(m.menuEdit.ClickedCh),
	})
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(m.menuReload.ClickedCh),
	})
	cases = append(cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(m.menuExit.ClickedCh),
	})

	// Add profile menu items
	for _, item := range m.profileMenuItems {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(item.ClickedCh),
		})
	}

	for {
		// Wait for any channel to receive
		chosen, _, _ := reflect.Select(cases)

		switch chosen {
		case 0: // Edit
			m.handleEditConfig()
		case 1: // Reload
			m.handleReloadConfig()
		case 2: // Exit
			systray.Quit()
			return
		default: // Profile item (index = chosen - 3)
			profileIndex := chosen - 3
			m.handleProfileSwitch(profileIndex)
		}
	}
}

// handleProfileSwitch handles clicking on a profile menu item
func (m *Manager) handleProfileSwitch(index int) {
	if m.profileMgr == nil {
		return
	}

	profiles := m.profileMgr.GetProfiles()
	if index < 0 || index >= len(profiles) {
		return
	}

	profile := profiles[index]
	activeProfile := m.profileMgr.GetActiveProfile()

	// Don't switch if already active
	if activeProfile != nil && profile.Path == activeProfile.Path {
		return
	}

	log.Printf("Switching to profile: %s (%s)", profile.Name, profile.Path)

	if m.onProfileSwitch != nil {
		if err := m.onProfileSwitch(profile.Path); err != nil {
			log.Printf("Failed to switch profile: %v", err)
			return
		}
	}

	// Update checkmarks and titles
	for i, item := range m.profileMenuItems {
		if i == index {
			item.Check()
			// On Linux, add prefix for active profile
			if runtime.GOOS == "linux" {
				item.SetTitle("✓ " + profiles[i].Name)
			}
		} else {
			item.Uncheck()
			// On Linux, remove prefix from inactive profiles
			if runtime.GOOS == "linux" {
				item.SetTitle(profiles[i].Name)
			}
		}
	}
}

// handleEditConfig opens configuration editor in browser
func (m *Manager) handleEditConfig() {
	// If web editor is available, use it
	if m.webEditor != nil {
		m.handleEditInBrowser()
		return
	}

	// Fallback to text editor if web editor not configured
	m.handleEditConfigInTextEditor()
}

// handleEditInBrowser starts web editor and opens browser
func (m *Manager) handleEditInBrowser() {
	if m.webEditor == nil {
		log.Println("Web editor not available, falling back to text editor")
		m.handleEditConfigInTextEditor()
		return
	}

	// Start server if not running
	if !m.webEditor.IsRunning() {
		if err := m.webEditor.Start(); err != nil {
			log.Printf("Failed to start web editor: %v", err)
			// Fall back to text editor
			m.handleEditConfigInTextEditor()
			return
		}
	}

	// Open browser
	url := m.webEditor.GetURL()
	if err := openBrowser(url); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// openBrowser opens the default browser with the given URL
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// handleEditConfigInTextEditor opens config file in default text editor (fallback)
func (m *Manager) handleEditConfigInTextEditor() {
	var absPath string
	var err error

	if m.profileMgr != nil {
		activeProfile := m.profileMgr.GetActiveProfile()
		if activeProfile != nil {
			absPath, err = filepath.Abs(activeProfile.Path)
		} else {
			log.Println("No active profile to edit")
			return
		}
	} else {
		absPath, err = filepath.Abs(m.configPath)
	}

	if err != nil {
		log.Printf("Failed to get absolute path: %v", err)
		return
	}

	// Check if config file exists, create if it doesn't
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("Config file doesn't exist, creating default: %s", absPath)
		if err := config.SaveDefault(absPath); err != nil {
			log.Printf("Failed to create default config: %v", err)
			return
		}
		log.Printf("Created default config: %s", absPath)
	}

	// Open with default editor
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
	case "darwin":
		cmd = exec.Command("open", "-t", absPath) // -t opens with default text editor
	default: // linux
		cmd = findLinuxEditor(absPath)
	}

	if cmd == nil {
		log.Printf("No suitable editor found")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open config: %v", err)
	}
}

// findLinuxEditor finds an appropriate text editor on Linux
func findLinuxEditor(filePath string) *exec.Cmd {
	// Check VISUAL and EDITOR environment variables first
	if editor := os.Getenv("VISUAL"); editor != "" {
		return exec.Command(editor, filePath)
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return exec.Command(editor, filePath)
	}

	// Try to get the default text editor from xdg-mime
	if editor := getXdgTextEditor(); editor != "" {
		return exec.Command(editor, filePath)
	}

	// Fall back to xdg-open as last resort
	return exec.Command("xdg-open", filePath)
}

// getXdgTextEditor queries xdg-mime for the default text/plain handler
func getXdgTextEditor() string {
	// Query default application for text/plain
	out, err := exec.Command("xdg-mime", "query", "default", "text/plain").Output()
	if err != nil {
		return ""
	}

	desktop := strings.TrimSpace(string(out))
	if desktop == "" {
		return ""
	}

	// Find the .desktop file and extract the Exec line
	// Check standard locations
	locations := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/applications"),
		"/usr/local/share/applications",
		"/usr/share/applications",
	}

	for _, loc := range locations {
		desktopPath := filepath.Join(loc, desktop)
		if _, err := os.Stat(desktopPath); err == nil {
			if cmd := parseDesktopExec(desktopPath); cmd != "" {
				return cmd
			}
		}
	}

	return ""
}

// parseDesktopExec extracts the executable from a .desktop file
func parseDesktopExec(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Exec=") {
			execLine := strings.TrimPrefix(line, "Exec=")
			// Extract just the command (first word), removing any %f, %u, etc.
			parts := strings.Fields(execLine)
			if len(parts) > 0 {
				cmd := parts[0]
				// Verify the command exists
				if _, err := exec.LookPath(cmd); err == nil {
					return cmd
				}
			}
			break
		}
	}

	return ""
}

// handleReloadConfig reloads the configuration
func (m *Manager) handleReloadConfig() {
	log.Println("Reloading configuration...")

	if m.onReload != nil {
		if err := m.onReload(); err != nil {
			log.Printf("Failed to reload config: %v", err)
			return
		}
	}

	// Refresh profile name in case config_name changed
	if m.profileMgr != nil {
		activeProfile := m.profileMgr.GetActiveProfile()
		if activeProfile != nil {
			m.profileMgr.RefreshProfile(activeProfile.Path)
			// Update menu item text
			for i, profile := range m.profileMgr.GetProfiles() {
				if i < len(m.profileMenuItems) {
					m.profileMenuItems[i].SetTitle(profile.Name)
				}
			}
		}
	}

	log.Println("Configuration reloaded successfully")
}

// Quit stops the system tray
func (m *Manager) Quit() {
	systray.Quit()
}

// OnReady sets a callback to be called when the tray is ready
func (m *Manager) OnReady(callback func()) {
	m.onReadyCallback = callback
}

// WaitReady blocks until the tray is ready
func (m *Manager) WaitReady() {
	<-m.readyChan
}

// UpdateActiveProfile updates the checkmark to reflect the current active profile
func (m *Manager) UpdateActiveProfile() {
	if m.profileMgr == nil {
		return
	}

	activeProfile := m.profileMgr.GetActiveProfile()
	profiles := m.profileMgr.GetProfiles()

	for i, profile := range profiles {
		if i < len(m.profileMenuItems) {
			isActive := activeProfile != nil && profile.Path == activeProfile.Path
			if isActive {
				m.profileMenuItems[i].Check()
				if runtime.GOOS == "linux" {
					m.profileMenuItems[i].SetTitle("✓ " + profile.Name)
				}
			} else {
				m.profileMenuItems[i].Uncheck()
				if runtime.GOOS == "linux" {
					m.profileMenuItems[i].SetTitle(profile.Name)
				}
			}
		}
	}
}

// getIcon returns the tray icon bytes
func getIcon() []byte {
	if len(iconData) > 0 {
		log.Println("Using embedded tray icon")
		return iconData
	}

	log.Println("No embedded icon found, using default system icon")
	return []byte{}
}
