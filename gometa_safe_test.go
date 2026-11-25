//go:build gometa

package main

import (
	"strings"
	"testing"
)

type bloksLiteral struct{ v any }

func (l *bloksLiteral) value() any { return l.v }

// What took the service down: a method called on a nil pointer deep inside the login.
func TestGometaRecoveredCatchesANilDereference(t *testing.T) {
	var got any
	var stack []byte
	gometaRecovered(func() {
		var lit *bloksLiteral
		_ = lit.value()
	}, func(r any, s []byte) { got, stack = r, s })
	if got == nil {
		t.Fatal("the panic was not handed to onPanic")
	}
	if !strings.Contains(string(stack), "bloksLiteral") {
		t.Errorf("stack does not show where it panicked:\n%s", stack)
	}
}

func TestGometaRecoveredLeavesANormalRunAlone(t *testing.T) {
	ran, panicked := false, false
	gometaRecovered(func() { ran = true }, func(any, []byte) { panicked = true })
	if !ran || panicked {
		t.Fatalf("ran=%v panicked=%v", ran, panicked)
	}
}

// The login runs on its own goroutine; the panic must be caught there too.
func TestGometaRecoveredOnAGoroutine(t *testing.T) {
	done := make(chan any, 1)
	go gometaRecovered(func() { panic("bloks") }, func(r any, _ []byte) { done <- r })
	if r := <-done; r != "bloks" {
		t.Fatalf("got %v", r)
	}
}
