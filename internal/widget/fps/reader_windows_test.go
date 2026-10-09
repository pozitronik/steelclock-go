//go:build windows

package fps

import (
	"encoding/binary"
	"errors"
	"testing"
	"unsafe"
)

// fakeRTSSSegment builds a byte buffer laid out like a real RTSS shared
// memory segment, with a header and a single app entry, for testing GetFPS's
// parsing/validation logic without a real RTSS process running.
type fakeRTSSSegment struct {
	buf []byte
}

const fakeAppArrOffset = 128 // arbitrary, past the header fields we care about

func newFakeRTSSSegment(signature, version uint32, appArrSize uint32) *fakeRTSSSegment {
	entrySize := entryOffFrameTime + 4
	buf := make([]byte, fakeAppArrOffset+int(appArrSize)*entrySize)
	binary.LittleEndian.PutUint32(buf[offSignature:], signature)
	binary.LittleEndian.PutUint32(buf[offVersion:], version)
	binary.LittleEndian.PutUint32(buf[offAppEntrySize:], uint32(entrySize))
	binary.LittleEndian.PutUint32(buf[offAppArrOffset:], fakeAppArrOffset)
	binary.LittleEndian.PutUint32(buf[offAppArrSize:], appArrSize)
	return &fakeRTSSSegment{buf: buf}
}

func (f *fakeRTSSSegment) setEntry(index int, pid, frameTime, time0, time1, frames uint32) {
	entrySize := entryOffFrameTime + 4
	base := fakeAppArrOffset + index*entrySize
	binary.LittleEndian.PutUint32(f.buf[base+entryOffProcessID:], pid)
	binary.LittleEndian.PutUint32(f.buf[base+entryOffTime0:], time0)
	binary.LittleEndian.PutUint32(f.buf[base+entryOffTime1:], time1)
	binary.LittleEndian.PutUint32(f.buf[base+entryOffFrames:], frames)
	binary.LittleEndian.PutUint32(f.buf[base+entryOffFrameTime:], frameTime)
}

func (f *fakeRTSSSegment) reader() *rtssReader {
	return &rtssReader{base: unsafe.Pointer(&f.buf[0])}
}

func now() uint32 {
	tick, _, _ := procGetTickCount.Call()
	return uint32(tick)
}

func TestGetFPS_InvalidSignature_ReturnsErrRTSSGone(t *testing.T) {
	seg := newFakeRTSSSegment(0xDEADBEEF, rtssMinVersion, 1)
	r := seg.reader()

	_, _, err := r.GetFPS()
	if !errors.Is(err, errRTSSGone) {
		t.Errorf("GetFPS() error = %v, want errRTSSGone", err)
	}
}

func TestGetFPS_OldVersion_ReturnsErrRTSSGone(t *testing.T) {
	seg := newFakeRTSSSegment(rtssSignature, 0x00010000, 1) // v1.0, pre-dates this layout
	r := seg.reader()

	_, _, err := r.GetFPS()
	if !errors.Is(err, errRTSSGone) {
		t.Errorf("GetFPS() error = %v, want errRTSSGone", err)
	}
}

func TestGetFPS_ClosedReader_ReturnsErrRTSSGone(t *testing.T) {
	r := &rtssReader{}

	_, _, err := r.GetFPS()
	if !errors.Is(err, errRTSSGone) {
		t.Errorf("GetFPS() error = %v, want errRTSSGone", err)
	}
}

func TestGetFPS_FreshEntry_Reported(t *testing.T) {
	seg := newFakeRTSSSegment(rtssSignature, rtssMinVersion, 1)
	n := now()
	// 60 frames over 1000ms = 60 FPS; window just closed, so it's fresh.
	seg.setEntry(0, 1234, 16, n-1000, n, 60)
	r := seg.reader()

	fps, _, err := r.GetFPS()
	if err != nil {
		t.Fatalf("GetFPS() error = %v", err)
	}
	if fps != 60.0 {
		t.Errorf("GetFPS() fps = %v, want 60.0", fps)
	}
}

func TestGetFPS_StaleEntry_Skipped(t *testing.T) {
	seg := newFakeRTSSSegment(rtssSignature, rtssMinVersion, 1)
	n := now()
	// Measurement window closed 10s ago and never advanced since: the app is
	// minimized/paused/hung, so this entry must not be reported.
	seg.setEntry(0, 1234, 16, n-11000, n-10000, 60)
	r := seg.reader()

	fps, name, err := r.GetFPS()
	if err != nil {
		t.Fatalf("GetFPS() error = %v", err)
	}
	if fps != 0 || name != "" {
		t.Errorf("GetFPS() = (%v, %q), want (0, \"\") for a stale entry", fps, name)
	}
}
