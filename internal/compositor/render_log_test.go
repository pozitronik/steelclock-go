package compositor

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

func TestLogRenderResult_ThrottlesRepeatedErrors(t *testing.T) {
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})

	c := &Compositor{}
	notConnected := errors.New("send failed: device not connected")
	other := errors.New("composite failed: boom")

	for i := 0; i < 5; i++ {
		c.logRenderResult(notConnected) // e.g. an unplugged device at 10 FPS
	}
	c.logRenderResult(nil) // device back
	c.logRenderResult(nil)
	c.logRenderResult(other)
	c.logRenderResult(notConnected)

	want := []string{
		"Render error: send failed: device not connected",
		"Render error repeated 4 more time(s): send failed: device not connected",
		"Render error: composite failed: boom",
		"Render error: send failed: device not connected",
	}
	got := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("log lines:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestLogRenderResult_SingleErrorHasNoRepeatLine(t *testing.T) {
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})

	c := &Compositor{}
	c.logRenderResult(errors.New("once"))
	c.logRenderResult(nil)

	if got := strings.TrimSpace(buf.String()); got != "Render error: once" {
		t.Errorf("log = %q, want only the error line", got)
	}
}
