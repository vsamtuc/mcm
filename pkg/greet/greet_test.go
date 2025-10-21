package greet_test

import (
	"testing"

	"github.com/vsamtuc/mcm/pkg/greet"
)

func TestHello(t *testing.T) {
	got := greet.Hello("Go")
	want := "Hello, Go!"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
