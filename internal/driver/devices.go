package driver

// SteelSeriesVID SteelSeries Vendor ID
const SteelSeriesVID uint16 = 0x1038

// KnownDevice represents a known SteelSeries device with OLED display
type KnownDevice struct {
	VID         uint16
	PID         uint16
	Name        string
	DisplaySize struct {
		Width  int
		Height int
	}
	NewProtocol func() Protocol // Factory for device-specific protocol; nil = ApexProtocol
}

// KnownDevices is a list of known SteelSeries devices with OLED displays
// Add new devices here as they are discovered/tested
var KnownDevices = []KnownDevice{
	{
		VID:  SteelSeriesVID,
		PID:  0x1612,
		Name: "Apex 7",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x1618,
		Name: "Apex 7 TKL",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x1610,
		Name: "Apex Pro",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x1614,
		Name: "Apex Pro TKL",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x161C,
		Name: "Apex 5",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	// Apex Pro (2023 refresh)
	{
		VID:  SteelSeriesVID,
		PID:  0x1630,
		Name: "Apex Pro (2023)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x1632,
		Name: "Apex Pro TKL (2023)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 40},
	},
	// Nova Pro / GameDAC Gen 2 family (128x64 OLED, column-major LSB protocol)
	{
		VID:  SteelSeriesVID,
		PID:  0x12cb,
		Name: "Arctis Nova Pro (Wired)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 64},
		NewProtocol: func() Protocol { return &NovaProProtocol{} },
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x12cd,
		Name: "Arctis Nova Pro Wireless (Base Station)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 64},
		NewProtocol: func() Protocol { return &NovaProProtocol{} },
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x12e0,
		Name: "Arctis Nova Pro Wireless (USB-C Dongle)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 64},
		NewProtocol: func() Protocol { return &NovaProProtocol{} },
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x12e5,
		Name: "Arctis Nova Pro Wireless (Xbox)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 64},
		NewProtocol: func() Protocol { return &NovaProProtocol{} },
	},
	{
		VID:  SteelSeriesVID,
		PID:  0x225d,
		Name: "Arctis Nova 5P (USB-C Dongle)",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 64},
		NewProtocol: func() Protocol { return &NovaProProtocol{} },
	},
	// Arctis Nova Pro Omni — confirmed working via the direct driver. It uses the
	// same Nova Pro image format (command 0x93, column-major LSB), but unlike older
	// units (OLED on mi_04, report ID 0x06) it exposes its 128x64 OLED on mi_03
	// collection 01 with feature report ID 0x01 and a 1036-byte report. The
	// interface and report ID are auto-detected at runtime (capability-based
	// selection + report-ID discovery), so only the protocol/resolution differ here.
	{
		VID:  SteelSeriesVID,
		PID:  0x2290,
		Name: "Arctis Nova Pro Omni",
		DisplaySize: struct {
			Width  int
			Height int
		}{128, 64},
		NewProtocol: func() Protocol { return &NovaProProtocol{} },
	},
}
