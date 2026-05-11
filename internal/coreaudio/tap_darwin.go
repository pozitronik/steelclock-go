//go:build darwin

package coreaudio

/*
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox

#include <CoreAudio/CoreAudio.h>
#include <AudioToolbox/AudioToolbox.h>
#include <stdlib.h>
#include <string.h>
#include <stdatomic.h>

// Ring buffer for audio samples (lock-free single producer, single consumer)
#define RING_SIZE 16384

typedef struct {
	float left[RING_SIZE];
	float right[RING_SIZE];
	_Atomic uint64_t writePos;
	uint64_t readPos; // only accessed by Go consumer

	// Peak levels updated by IOProc via atomics
	_Atomic uint32_t peakLeft;  // float bits stored as uint32
	_Atomic uint32_t peakRight;

	AudioDeviceID aggregateID;
	AudioDeviceIOProcID ioProcID;
	int sampleRate;
	int channels;
	int running;
} AudioTapState;

// floatToUint32 / uint32ToFloat for atomic peak storage
static inline uint32_t floatBitsToUint32(float f) {
	uint32_t r;
	memcpy(&r, &f, sizeof(r));
	return r;
}

static inline float uint32BitsToFloat(uint32_t u) {
	float r;
	memcpy(&r, &u, sizeof(r));
	return r;
}

// IOProc callback - called on real-time audio thread
static OSStatus tapIOProc(
	AudioDeviceID device,
	const AudioTimeStamp *now,
	const AudioBufferList *inputData,
	const AudioTimeStamp *inputTime,
	AudioBufferList *outputData,
	const AudioTimeStamp *outputTime,
	void *clientData
) {
	AudioTapState *state = (AudioTapState *)clientData;
	if (!state || !inputData || inputData->mNumberBuffers == 0) return noErr;

	AudioBuffer buf = inputData->mBuffers[0];
	float *samples = (float *)buf.mData;
	UInt32 numChannels = buf.mNumberChannels;
	UInt32 numFrames = buf.mDataByteSize / (numChannels * sizeof(float));

	float peakL = 0.0f, peakR = 0.0f;
	uint64_t wp = atomic_load_explicit(&state->writePos, memory_order_relaxed);

	for (UInt32 i = 0; i < numFrames; i++) {
		float left = samples[i * numChannels];
		float right = (numChannels >= 2) ? samples[i * numChannels + 1] : left;

		uint64_t idx = (wp + i) % RING_SIZE;
		state->left[idx] = left;
		state->right[idx] = right;

		float absL = left < 0 ? -left : left;
		float absR = right < 0 ? -right : right;
		if (absL > peakL) peakL = absL;
		if (absR > peakR) peakR = absR;
	}

	atomic_store_explicit(&state->writePos, wp + numFrames, memory_order_release);
	atomic_store_explicit(&state->peakLeft, floatBitsToUint32(peakL), memory_order_release);
	atomic_store_explicit(&state->peakRight, floatBitsToUint32(peakR), memory_order_release);

	// Silence output
	for (UInt32 i = 0; i < outputData->mNumberBuffers; i++) {
		memset(outputData->mBuffers[i].mData, 0, outputData->mBuffers[i].mDataByteSize);
	}

	return noErr;
}

// getDefaultOutputDeviceID returns the default output audio device
static AudioDeviceID getDefaultOutputDevice(void) {
	AudioDeviceID deviceID = kAudioObjectUnknown;
	UInt32 size = sizeof(deviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);
	return deviceID;
}

// createAudioTap creates an aggregate device tap for system audio capture
static AudioTapState* createAudioTap(void) {
	AudioDeviceID outputDevice = getDefaultOutputDevice();
	if (outputDevice == kAudioObjectUnknown) return NULL;

	AudioTapState *state = (AudioTapState *)calloc(1, sizeof(AudioTapState));
	if (!state) return NULL;

	// Get the UID of the default output device
	CFStringRef outputUID = NULL;
	UInt32 size = sizeof(outputUID);
	AudioObjectPropertyAddress uidAddr = {
		kAudioDevicePropertyDeviceUID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	OSStatus err = AudioObjectGetPropertyData(outputDevice, &uidAddr, 0, NULL, &size, &outputUID);
	if (err != noErr || !outputUID) {
		free(state);
		return NULL;
	}

	// Create aggregate device description with tap
	CFMutableDictionaryRef aggDesc = CFDictionaryCreateMutable(
		kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);

	// Unique UID for our aggregate
	CFStringRef aggUID = CFSTR("com.steelclock.audiotap");
	CFDictionarySetValue(aggDesc, CFSTR(kAudioAggregateDeviceUIDKey), aggUID);
	CFDictionarySetValue(aggDesc, CFSTR(kAudioAggregateDeviceNameKey), CFSTR("SteelClock Audio Tap"));

	// Mark as private (not shown in system prefs)
	int isPrivate = 1;
	CFNumberRef privateVal = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &isPrivate);
	CFDictionarySetValue(aggDesc, CFSTR(kAudioAggregateDeviceIsPrivateKey), privateVal);
	CFRelease(privateVal);

	// Create tap sub-device referencing the output device
	CFMutableDictionaryRef tapSubDevice = CFDictionaryCreateMutable(
		kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(tapSubDevice, CFSTR(kAudioSubDeviceUIDKey), outputUID);

	CFMutableArrayRef tapList = CFArrayCreateMutable(kCFAllocatorDefault, 0, &kCFTypeArrayCallBacks);
	CFArrayAppendValue(tapList, tapSubDevice);
	CFDictionarySetValue(aggDesc, CFSTR(kAudioAggregateDeviceTapListKey), tapList);

	// Also set as sub-device list for compatibility
	CFMutableDictionaryRef subDevice = CFDictionaryCreateMutable(
		kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(subDevice, CFSTR(kAudioSubDeviceUIDKey), outputUID);

	CFMutableArrayRef subDeviceList = CFArrayCreateMutable(kCFAllocatorDefault, 0, &kCFTypeArrayCallBacks);
	CFArrayAppendValue(subDeviceList, subDevice);
	CFDictionarySetValue(aggDesc, CFSTR(kAudioAggregateDeviceSubDeviceListKey), subDeviceList);

	// Create the aggregate device
	AudioDeviceID aggDevice = kAudioObjectUnknown;
	err = AudioHardwareCreateAggregateDevice(aggDesc, &aggDevice);

	CFRelease(aggDesc);
	CFRelease(tapSubDevice);
	CFRelease(tapList);
	CFRelease(subDevice);
	CFRelease(subDeviceList);
	CFRelease(outputUID);

	if (err != noErr || aggDevice == kAudioObjectUnknown) {
		free(state);
		return NULL;
	}

	state->aggregateID = aggDevice;

	// Get sample rate from the aggregate device
	Float64 sampleRate = 0;
	size = sizeof(sampleRate);
	AudioObjectPropertyAddress srAddr = {
		kAudioDevicePropertyNominalSampleRate,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain
	};
	err = AudioObjectGetPropertyData(aggDevice, &srAddr, 0, NULL, &size, &sampleRate);
	if (err == noErr && sampleRate > 0) {
		state->sampleRate = (int)sampleRate;
	} else {
		state->sampleRate = 48000;
	}
	state->channels = 2;

	// Install IOProc
	err = AudioDeviceCreateIOProcID(aggDevice, tapIOProc, state, &state->ioProcID);
	if (err != noErr) {
		AudioHardwareDestroyAggregateDevice(aggDevice);
		free(state);
		return NULL;
	}

	// Start the device
	err = AudioDeviceStart(aggDevice, state->ioProcID);
	if (err != noErr) {
		AudioDeviceDestroyIOProcID(aggDevice, state->ioProcID);
		AudioHardwareDestroyAggregateDevice(aggDevice);
		free(state);
		return NULL;
	}

	state->running = 1;
	return state;
}

// destroyAudioTap stops and cleans up the audio tap
static void destroyAudioTap(AudioTapState *state) {
	if (!state) return;

	if (state->running) {
		AudioDeviceStop(state->aggregateID, state->ioProcID);
		state->running = 0;
	}

	if (state->ioProcID) {
		AudioDeviceDestroyIOProcID(state->aggregateID, state->ioProcID);
	}

	if (state->aggregateID != kAudioObjectUnknown) {
		AudioHardwareDestroyAggregateDevice(state->aggregateID);
	}

	free(state);
}

// readPeaks reads the current peak levels atomically
static void readPeaks(AudioTapState *state, float *left, float *right) {
	*left = uint32BitsToFloat(atomic_load_explicit(&state->peakLeft, memory_order_acquire));
	*right = uint32BitsToFloat(atomic_load_explicit(&state->peakRight, memory_order_acquire));
	// Reset peaks after reading
	atomic_store_explicit(&state->peakLeft, 0, memory_order_release);
	atomic_store_explicit(&state->peakRight, 0, memory_order_release);
}

// readSamples copies available samples from the ring buffer
// Returns number of frames copied
static uint64_t readSamples(AudioTapState *state, float *outLeft, float *outRight, uint64_t maxFrames, uint64_t *readPos) {
	uint64_t wp = atomic_load_explicit(&state->writePos, memory_order_acquire);
	uint64_t rp = *readPos;

	uint64_t available = wp - rp;
	if (available > RING_SIZE) {
		// Reader fell behind, skip to recent data
		rp = wp - RING_SIZE;
		available = RING_SIZE;
	}
	if (available > maxFrames) {
		// Skip older samples to only return most recent
		rp = wp - maxFrames;
		available = maxFrames;
	}

	for (uint64_t i = 0; i < available; i++) {
		uint64_t idx = (rp + i) % RING_SIZE;
		outLeft[i] = state->left[idx];
		outRight[i] = state->right[idx];
	}

	*readPos = rp + available;
	return available;
}

static int tapIsRunning(AudioTapState *state) {
	return state ? state->running : 0;
}

static int tapGetSampleRate(AudioTapState *state) {
	return state ? state->sampleRate : 0;
}

static int tapGetChannels(AudioTapState *state) {
	return state ? state->channels : 0;
}
*/
import "C"

import (
	"fmt"
	"log"
	"sync"
	"unsafe"
)

const ringSize = 16384

// AudioTap provides real-time system audio capture on macOS via CoreAudio aggregate device tap
type AudioTap struct {
	mu      sync.Mutex
	state   *C.AudioTapState
	readPos C.uint64_t
}

// NewAudioTap creates and starts a new audio tap
func NewAudioTap() (*AudioTap, error) {
	state := C.createAudioTap()
	if state == nil {
		return nil, fmt.Errorf("failed to create CoreAudio aggregate device tap")
	}

	log.Printf("[COREAUDIO-TAP] Audio tap started (sample rate: %d, channels: %d)",
		int(state.sampleRate), int(state.channels))

	return &AudioTap{
		state: state,
	}, nil
}

// GetPeakLevels returns the current peak levels (0.0-1.0) since the last call
func (t *AudioTap) GetPeakLevels() (left, right float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == nil {
		return 0, 0
	}

	var l, r C.float
	C.readPeaks(t.state, &l, &r)
	return float64(l), float64(r)
}

// ReadSamples returns available audio samples from the ring buffer
func (t *AudioTap) ReadSamples() (left, right []float32, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == nil {
		return nil, nil, fmt.Errorf("audio tap not running")
	}

	var outLeft [ringSize]C.float
	var outRight [ringSize]C.float

	count := C.readSamples(t.state,
		(*C.float)(unsafe.Pointer(&outLeft[0])),
		(*C.float)(unsafe.Pointer(&outRight[0])),
		C.uint64_t(ringSize),
		&t.readPos)

	if count == 0 {
		return nil, nil, nil
	}

	n := int(count)
	left = make([]float32, n)
	right = make([]float32, n)
	for i := 0; i < n; i++ {
		left[i] = float32(outLeft[i])
		right[i] = float32(outRight[i])
	}

	return left, right, nil
}

// GetRecentSamples returns the most recent N samples
func (t *AudioTap) GetRecentSamples(count int) (left, right []float32) {
	l, r, err := t.ReadSamples()
	if err != nil || len(l) == 0 {
		return nil, nil
	}

	if count >= len(l) {
		return l, r
	}

	start := len(l) - count
	return l[start:], r[start:]
}

// SampleRate returns the capture sample rate
func (t *AudioTap) SampleRate() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == nil {
		return 0
	}
	return int(C.tapGetSampleRate(t.state))
}

// Channels returns the number of capture channels
func (t *AudioTap) Channels() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == nil {
		return 0
	}
	return int(C.tapGetChannels(t.state))
}

// IsRunning returns true if the audio tap is active
func (t *AudioTap) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == nil {
		return false
	}
	return C.tapIsRunning(t.state) != 0
}

// Close stops and destroys the audio tap
func (t *AudioTap) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state != nil {
		C.destroyAudioTap(t.state)
		t.state = nil
		log.Println("[COREAUDIO-TAP] Audio tap stopped")
	}
}
