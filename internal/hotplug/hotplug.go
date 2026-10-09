// Package hotplug reports when USB HID devices of a given vendor are connected
// or disconnected, so a device that is waiting for its hardware can pick it up
// at once instead of at its next poll.
package hotplug

import (
	"errors"
	"time"
)

// ErrUnsupported is returned where device notifications are not available.
// Callers should fall back to polling.
var ErrUnsupported = errors.New("device hotplug notifications are not supported on this platform")

const (
	// settleDelay lets a burst of events settle (one device announces several
	// HID interfaces) before notifying.
	settleDelay = 300 * time.Millisecond
	// lateRecheckDelay notifies once more after a burst, for devices that are
	// announced slightly before they can be opened (Windows arrival, or Linux
	// before udev has applied device-node permissions).
	lateRecheckDelay = 1500 * time.Millisecond
)

// Watcher delivers debounced notifications about device arrivals and removals.
type Watcher struct {
	notify     func()
	settle     time.Duration
	late       time.Duration
	events     chan struct{}
	stop       chan struct{}
	done       chan struct{}
	stopSource func()
}

// Start begins watching for HID devices with the given USB vendor ID. notify is
// called from a background goroutine shortly after devices of that vendor are
// connected or disconnected. It returns an error (wrapping ErrUnsupported where
// applicable) when notifications cannot be set up.
func Start(vendorID uint16, notify func()) (*Watcher, error) {
	w := newWatcher(notify, settleDelay, lateRecheckDelay)
	stopSource, err := startSource(vendorID, w.signal)
	if err != nil {
		w.stopDebouncer()
		return nil, err
	}
	w.stopSource = stopSource
	return w, nil
}

// Stop stops watching. notify is not called after Stop returns.
func (w *Watcher) Stop() {
	if w.stopSource != nil {
		w.stopSource()
	}
	w.stopDebouncer()
}

// newWatcher creates a watcher with its debouncer running but no event source.
func newWatcher(notify func(), settle, late time.Duration) *Watcher {
	w := &Watcher{
		notify: notify,
		settle: settle,
		late:   late,
		events: make(chan struct{}, 1),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go w.debounce()
	return w
}

// signal records a raw device event. It never blocks, so it is safe to call
// from OS callbacks.
func (w *Watcher) signal() {
	select {
	case w.events <- struct{}{}:
	default: // an event is already pending
	}
}

// debounce turns raw events into notifications: one when a burst has settled,
// and one more a little later.
func (w *Watcher) debounce() {
	defer close(w.done)
	var settle, late <-chan time.Time
	for {
		select {
		case <-w.stop:
			return
		case <-w.events:
			settle = time.After(w.settle)
			late = nil
		case <-settle:
			settle = nil
			late = time.After(w.late)
			w.notify()
		case <-late:
			late = nil
			w.notify()
		}
	}
}

func (w *Watcher) stopDebouncer() {
	close(w.stop)
	<-w.done
}
