package tray

import (
	_ "embed"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/pozitronik/steelclock-go/internal/config"
)

//go:embed icon.ico
var iconData []byte

// Manager handles system tray icon and menu
type Manager struct {
	configPath      string
	onReload        func() error
	onExit          func()
	menuEdit        *systray.MenuItem
	menuReload      *systray.MenuItem
	menuExit        *systray.MenuItem
	readyChan       chan struct{} // Signals when tray is initialized
	onReadyCallback func()        // Called when tray is ready
}

// NewManager creates a new tray manager
func NewManager(configPath string, onReload func() error, onExit func()) *Manager {
	return &Manager{
		configPath: configPath,
		onReload:   onReload,
		onExit:     onExit,
		readyChan:  make(chan struct{}),
	}
}

// Run starts the system tray
func (m *Manager) Run() {
	systray.Run(m.onReady, m.onQuit)
}

// onReady is called when systray is ready
func (m *Manager) onReady() {
	// Set icon and tooltip
	systray.SetIcon(getIcon())
	systray.SetTitle("SteelClock")
	systray.SetTooltip("SteelClock - SteelSeries Display")

	// Create menu items
	m.menuEdit = systray.AddMenuItem("Edit Config", "Open config.json in default editor")
	m.menuReload = systray.AddMenuItem("Reload Config", "Reload configuration from config.json")
	systray.AddSeparator()
	m.menuExit = systray.AddMenuItem("Exit", "Exit SteelClock")

	// Signal that tray is ready
	close(m.readyChan)

	// Call ready callback if set
	if m.onReadyCallback != nil {
		go m.onReadyCallback()
	}

	// Handle menu clicks
	go m.handleMenuClicks()
}

// onQuit is called when systray is quitting
func (m *Manager) onQuit() {
	if m.onExit != nil {
		m.onExit()
	}
}

// handleMenuClicks processes menu item clicks
func (m *Manager) handleMenuClicks() {
	for {
		select {
		case <-m.menuEdit.ClickedCh:
			m.handleEditConfig()
		case <-m.menuReload.ClickedCh:
			m.handleReloadConfig()
		case <-m.menuExit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

// handleEditConfig opens config file in default editor
func (m *Manager) handleEditConfig() {
	// Get absolute path
	absPath, err := filepath.Abs(m.configPath)
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
		cmd = exec.Command("open", absPath)
	default: // linux
		cmd = exec.Command("xdg-open", absPath)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open config: %v", err)
	}
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

// getIcon returns the tray icon bytes
// Returns embedded icon data
func getIcon() []byte {
	if len(iconData) > 0 {
		log.Println("Using embedded tray icon")
		return iconData
	}

	// Fallback: return empty/default icon
	log.Println("No embedded icon found, using default system icon")
	return []byte{}
}

// ValidateConfig checks if config file exists and is valid
