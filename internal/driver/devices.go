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
}

// FindKnownDevice searches for a known device by VID and PID
func FindKnownDevice(vid, pid uint16) *KnownDevice {
	for i := range KnownDevices {
		if KnownDevices[i].VID == vid && KnownDevices[i].PID == pid {
			return &KnownDevices[i]
		}
	}
	return nil
}

// GetAllKnownPIDs returns all known product IDs for SteelSeries devices
func GetAllKnownPIDs() []uint16 {
	pids := make([]uint16, len(KnownDevices))
	for i, d := range KnownDevices {
		pids[i] = d.PID
	}
	return pids
}
