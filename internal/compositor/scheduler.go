package compositor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/widget"
)

// WidgetScheduler manages widget update goroutines.
// It handles starting, stopping, and coordinating widget update loops.
type WidgetScheduler struct {
	widgets  []widget.Widget
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewWidgetScheduler creates a new scheduler for the given widgets.
func NewWidgetScheduler(widgets []widget.Widget) *WidgetScheduler {
	return &WidgetScheduler{
		widgets: widgets,
	}
}

// Start begins update loops for all widgets.
// Each widget runs in its own goroutine at its configured update interval.
func (s *WidgetScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.stopChan = make(chan struct{})
	s.running = true

	for _, w := range s.widgets {
		s.wg.Add(1)
		go s.widgetUpdateLoop(w)
	}

	log.Printf("Widget scheduler started with %d widget(s)", len(s.widgets))
}

// Stop signals all widget update loops to terminate and waits for completion.
// Also calls Stop() on any widgets that implement the Stoppable interface.
func (s *WidgetScheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	// Wait for all update loops to finish
	s.wg.Wait()

	// Stop any widgets that need cleanup (goroutines, subscriptions, etc.)
	widget.StopWidgets(s.widgets)

	log.Println("Widget scheduler stopped")
}

// IsRunning returns whether the scheduler is currently active.
func (s *WidgetScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// WidgetCount returns the number of managed widgets.
func (s *WidgetScheduler) WidgetCount() int {
	return len(s.widgets)
}

// widgetUpdateLoop runs the update loop for a single widget.
func (s *WidgetScheduler) widgetUpdateLoop(w widget.Widget) {
	defer s.wg.Done()
	defer logPanic(fmt.Sprintf("widgetUpdateLoop for %s", w.Name()))

	ticker := time.NewTicker(w.GetUpdateInterval())
	defer ticker.Stop()

	// Initial update
	if err := w.Update(); err != nil {
		log.Printf("Widget %s update error: %v", w.Name(), err)
	}

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := w.Update(); err != nil {
				log.Printf("Widget %s update error: %v", w.Name(), err)
			}
		}
	}
}
