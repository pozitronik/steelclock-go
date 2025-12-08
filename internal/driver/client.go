package driver

import (
	"fmt"
	"log"

	"github.com/pozitronik/steelclock-go/internal/display"
)

// Client wraps HIDDriver and implements display.Backend interface
// This allows the direct driver to be used interchangeably with the GameSense client
type Client struct {
	driver           *HIDDriver
	width            int
	height           int
	disconnectLogged bool // prevents log spam on disconnect
}

// Ensure Client implements display.Backend
var _ display.Backend = (*Client)(nil)

// NewClient creates a new direct driver client
func NewClient(cfg Config) (*Client, error) {
	driver := NewDriver(cfg)

	if err := driver.Open(); err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}

	log.Printf("Direct driver connected to device: VID_%04X PID_%04X path=%s",
		driver.deviceInfo.VID, driver.deviceInfo.PID, driver.deviceInfo.Path)

	return &Client{
		driver: driver,
		width:  cfg.Width,
		height: cfg.Height,
	}, nil
}

// RegisterGame is a no-op for direct driver (device doesn't need registration)
func (c *Client) RegisterGame(_ string, _ int) error {
	log.Printf("Direct driver: RegisterGame (no-op)")
	return nil
}

// BindScreenEvent is a no-op for direct driver
func (c *Client) BindScreenEvent(_, _ string) error {
	log.Printf("Direct driver: BindScreenEvent (no-op)")
	return nil
}

// SendScreenData sends the bitmap data directly to the display
// bitmapData is an array of 640 bytes, each representing packed pixels
func (c *Client) SendScreenData(_ string, bitmapData []byte) error {
	if !c.driver.IsConnected() {
		// Log disconnection only once to avoid spam
		if !c.disconnectLogged {
			log.Printf("Direct driver: device disconnected, skipping frames until reconnected")
			c.disconnectLogged = true
		}
		return fmt.Errorf("device not connected")
	}

	if err := c.driver.SendFrame(bitmapData); err != nil {
		// Log disconnection only once to avoid spam
		if !c.disconnectLogged {
			log.Printf("Direct driver: device disconnected: %v", err)
			c.disconnectLogged = true
		}
		return err
	}

	return nil
}

// SendHeartbeat checks connection and attempts to reconnect if needed
func (c *Client) SendHeartbeat() error {
	if !c.driver.IsConnected() {
		log.Printf("Direct driver: attempting reconnect...")
		if err := c.driver.Reconnect(); err != nil {
			log.Printf("Direct driver: reconnect failed: %v", err)
			return err
		}
		log.Printf("Direct driver: reconnected successfully")
		c.disconnectLogged = false // reset flag so next disconnect gets logged
	}
	return nil
}

// RemoveGame closes the driver connection
func (c *Client) RemoveGame() error {
	log.Printf("Direct driver: closing connection")
	return c.driver.Close()
}

// IsConnected returns true if the device is connected
func (c *Client) IsConnected() bool {
	return c.driver.IsConnected()
}

// DeviceInfo returns information about the connected device
func (c *Client) DeviceInfo() DeviceInfo {
	return c.driver.DeviceInfo()
}

// Driver returns the underlying HID driver
func (c *Client) Driver() *HIDDriver {
	return c.driver
}

// SupportsMultipleEvents returns false - USB HID doesn't benefit from HTTP batching optimization
func (c *Client) SupportsMultipleEvents() bool {
	return false
}

// SendScreenDataMultiRes sends screen data for the resolution matching the driver's configured dimensions.
// Other resolutions in the map are ignored.
func (c *Client) SendScreenDataMultiRes(_ string, resolutionData map[string][]byte) error {
	// Find our resolution in the map
	key := fmt.Sprintf("image-data-%dx%d", c.width, c.height)
	if data, ok := resolutionData[key]; ok {
		return c.SendScreenData("", data)
	}
	return fmt.Errorf("resolution %dx%d not found in data", c.width, c.height)
}

// SendMultipleScreenData sends the last frame from the batch.
// USB HID doesn't benefit from batching (no HTTP overhead), so only the most recent frame is sent.
func (c *Client) SendMultipleScreenData(_ string, frames [][]byte) error {
	if len(frames) > 0 {
		return c.SendScreenData("", frames[len(frames)-1])
	}
	return nil
}
