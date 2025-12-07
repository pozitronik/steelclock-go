package shared

// ByteUnit represents a data rate unit for display
type ByteUnit struct {
	Name     string  // Display name (e.g., "MB/s", "Mbps")
	Divisor  float64 // Divisor to convert from bytes per second
	IsBits   bool    // True for bit-based units (bps, Kbps, Mbps, Gbps)
	IsBinary bool    // True for binary units (KiB/s, MiB/s, GiB/s)
}

// UnitFamily represents a group of related units for auto-scaling
type UnitFamily int

const (
	// UnitFamilyBytesDecimal uses B/s, KB/s, MB/s, GB/s
	UnitFamilyBytesDecimal UnitFamily = iota
	// UnitFamilyBytesBinary uses B/s, KiB/s, MiB/s, GiB/s
	UnitFamilyBytesBinary
	// UnitFamilyBits uses bps, Kbps, Mbps, Gbps
	UnitFamilyBits
)

// Predefined byte-based units (decimal)
var ByteUnitsDecimal = map[string]ByteUnit{
	"B/s":  {Name: "B/s", Divisor: 1, IsBits: false, IsBinary: false},
	"KB/s": {Name: "KB/s", Divisor: 1000, IsBits: false, IsBinary: false},
	"MB/s": {Name: "MB/s", Divisor: 1000000, IsBits: false, IsBinary: false},
	"GB/s": {Name: "GB/s", Divisor: 1000000000, IsBits: false, IsBinary: false},
}

// Predefined byte-based units (binary)
var ByteUnitsBinary = map[string]ByteUnit{
	"B/s":   {Name: "B/s", Divisor: 1, IsBits: false, IsBinary: false},
	"KiB/s": {Name: "KiB/s", Divisor: 1024, IsBits: false, IsBinary: true},
	"MiB/s": {Name: "MiB/s", Divisor: 1048576, IsBits: false, IsBinary: true},
	"GiB/s": {Name: "GiB/s", Divisor: 1073741824, IsBits: false, IsBinary: true},
}

// Predefined bit-based units (for network)
var BitUnits = map[string]ByteUnit{
	"bps":  {Name: "bps", Divisor: 1.0 / 8, IsBits: true, IsBinary: false},
	"Kbps": {Name: "Kbps", Divisor: 1000.0 / 8, IsBits: true, IsBinary: false},
	"Mbps": {Name: "Mbps", Divisor: 1000000.0 / 8, IsBits: true, IsBinary: false},
	"Gbps": {Name: "Gbps", Divisor: 1000000000.0 / 8, IsBits: true, IsBinary: false},
}

// AllUnits combines all predefined units for lookup
var AllUnits = func() map[string]ByteUnit {
	result := make(map[string]ByteUnit)
	for k, v := range ByteUnitsDecimal {
		result[k] = v
	}
	for k, v := range ByteUnitsBinary {
		result[k] = v
	}
	for k, v := range BitUnits {
		result[k] = v
	}
	return result
}()

// Auto-scale unit orders
var autoScaleBytesDecimal = []string{"B/s", "KB/s", "MB/s", "GB/s"}
var autoScaleBytesBinary = []string{"B/s", "KiB/s", "MiB/s", "GiB/s"}
var autoScaleBits = []string{"bps", "Kbps", "Mbps", "Gbps"}

// ByteRateConverter handles conversion between byte rate units
type ByteRateConverter struct {
	units       map[string]ByteUnit
	defaultUnit string
	family      UnitFamily
}

// NewByteRateConverter creates a new converter with the specified default unit
func NewByteRateConverter(defaultUnit string) *ByteRateConverter {
	family := DetermineUnitFamily(defaultUnit)
	return &ByteRateConverter{
		units:       AllUnits,
		defaultUnit: defaultUnit,
		family:      family,
	}
}

// DetermineUnitFamily determines which unit family a unit belongs to
func DetermineUnitFamily(unitName string) UnitFamily {
	if unit, ok := AllUnits[unitName]; ok {
		if unit.IsBits {
			return UnitFamilyBits
		}
		if unit.IsBinary {
			return UnitFamilyBytesBinary
		}
	}
	return UnitFamilyBytesDecimal
}

// IsValidUnit checks if a unit name is valid
func IsValidUnit(unitName string) bool {
	_, ok := AllUnits[unitName]
	return ok
}

// Convert converts bytes per second to the specified unit
func (c *ByteRateConverter) Convert(bps float64, unitName string) (float64, string) {
	if unitName == "auto" {
		return c.AutoScale(bps)
	}

	unit, ok := c.units[unitName]
	if !ok {
		unit = c.units[c.defaultUnit]
	}

	return bps / unit.Divisor, unit.Name
}

// AutoScale automatically selects the best unit based on the value
func (c *ByteRateConverter) AutoScale(bps float64) (float64, string) {
	unitNames := c.getAutoScaleUnits()

	// Find the best unit (largest unit where value >= 1)
	selectedUnit := unitNames[0]
	for _, unitName := range unitNames {
		unit := c.units[unitName]
		if bps/unit.Divisor >= 1 {
			selectedUnit = unitName
		} else {
			break
		}
	}

	unit := c.units[selectedUnit]
	return bps / unit.Divisor, unit.Name
}

// SetFamily changes the unit family for auto-scaling
func (c *ByteRateConverter) SetFamily(family UnitFamily) {
	c.family = family
}

// GetFamily returns the current unit family
func (c *ByteRateConverter) GetFamily() UnitFamily {
	return c.family
}

// getAutoScaleUnits returns the ordered list of units for auto-scaling
func (c *ByteRateConverter) getAutoScaleUnits() []string {
	switch c.family {
	case UnitFamilyBytesBinary:
		return autoScaleBytesBinary
	case UnitFamilyBits:
		return autoScaleBits
	default:
		return autoScaleBytesDecimal
	}
}

// --- Standalone functions for simple conversions ---

// AutoScaleBytes converts bytes per second to the best readable unit
func AutoScaleBytes(bps float64, useBinary bool) (float64, string) {
	var unitNames []string
	var units map[string]ByteUnit

	if useBinary {
		unitNames = autoScaleBytesBinary
		units = ByteUnitsBinary
	} else {
		unitNames = autoScaleBytesDecimal
		units = ByteUnitsDecimal
	}

	selectedUnit := unitNames[0]
	for _, unitName := range unitNames {
		unit := units[unitName]
		if bps/unit.Divisor >= 1 {
			selectedUnit = unitName
		} else {
			break
		}
	}

	unit := units[selectedUnit]
	return bps / unit.Divisor, unit.Name
}

// AutoScaleBits converts bytes per second to the best readable bit-based unit
func AutoScaleBits(bps float64) (float64, string) {
	selectedUnit := autoScaleBits[0]
	for _, unitName := range autoScaleBits {
		unit := BitUnits[unitName]
		if bps/unit.Divisor >= 1 {
			selectedUnit = unitName
		} else {
			break
		}
	}

	unit := BitUnits[selectedUnit]
	return bps / unit.Divisor, unit.Name
}

// ConvertToUnit converts bytes per second to a specific unit
func ConvertToUnit(bps float64, unitName string) (float64, string) {
	unit, ok := AllUnits[unitName]
	if !ok {
		return bps, "B/s"
	}
	return bps / unit.Divisor, unit.Name
}
