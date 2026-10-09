package gamesense

import (
	"errors"
	"fmt"
	"net"
	"testing"
)

func TestProbe(t *testing.T) {
	origFunc := findCorePropsPathFunc
	t.Cleanup(func() { findCorePropsPathFunc = origFunc })

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	liveAddr := listener.Addr().String()

	// A port that was listening and is now closed, like a stale coreProps.json
	// left behind by a closed SteelSeries GG.
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedAddr := closed.Addr().String()
	_ = closed.Close()

	tests := []struct {
		name  string
		props func(t *testing.T) (string, error)
		want  bool
	}{
		{"server accepting connections", func(t *testing.T) (string, error) {
			return writeTempFile(t, t.TempDir(), "coreProps.json", fmt.Sprintf(`{"address":%q}`, liveAddr)), nil
		}, true},
		{"stale address, GG closed", func(t *testing.T) (string, error) {
			return writeTempFile(t, t.TempDir(), "coreProps.json", fmt.Sprintf(`{"address":%q}`, closedAddr)), nil
		}, false},
		{"no coreProps.json", func(*testing.T) (string, error) {
			return "", errors.New("coreProps.json not found")
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, perr := tt.props(t)
			findCorePropsPathFunc = func() (string, error) { return path, perr }

			if got := probe(nil); got != tt.want {
				t.Errorf("probe() = %v, want %v", got, tt.want)
			}
		})
	}
}
