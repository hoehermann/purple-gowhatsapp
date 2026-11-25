//go:build gometa

package main

import "runtime/debug"

// gometaRecovered runs f and, if it panics, hands the panic value and stack to onPanic instead of
// letting it unwind further. Go code in this plugin runs on a Go runtime inside the messaging
// service, so a panic in one of its own goroutines aborts the whole service, every account with it.
func gometaRecovered(f func(), onPanic func(r any, stack []byte)) {
	defer func() {
		if r := recover(); r != nil {
			onPanic(r, debug.Stack())
		}
	}()
	f()
}
