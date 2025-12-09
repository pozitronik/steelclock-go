package shared

// Audio visualizer display mode constants
const (
	AudioDisplayModeSpectrum     = "spectrum"
	AudioDisplayModeOscilloscope = "oscilloscope"
)

// Audio visualizer frequency scale constants
//
//goland:noinspection GoUnusedConst
const (
	AudioFrequencyScaleLogarithmic = "logarithmic"
	AudioFrequencyScaleLinear      = "linear"
)

// Audio visualizer bar style constants (for spectrum mode)
const (
	AudioBarStyleBars = "bars"
	AudioBarStyleLine = "line"
)

// Audio visualizer waveform style constants (for oscilloscope mode)
const (
	AudioWaveformStyleLine   = "line"
	AudioWaveformStyleFilled = "filled"
)

// Audio visualizer channel mode constants
const (
	AudioChannelModeMono            = "mono"
	AudioChannelModeStereoSeparated = "stereo_separated"
	AudioChannelModeStereoCombined  = "stereo_combined"
)
