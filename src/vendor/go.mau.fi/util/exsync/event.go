// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exsync

import (
	"context"
	"sync"
	"time"
)

// Event is a wrapper around a channel that can be used to notify multiple waiters that some event has happened.
//
// It's modelled after Python's asyncio.Event: https://docs.python.org/3/library/asyncio-sync.html#asyncio.Event
type Event struct {
	ch  chan empty
	set bool
	l   sync.RWMutex
}

// NewEvent creates a new event. It will initially be unset.
func NewEvent() *Event {
	return &Event{
		ch: make(chan empty),
	}
}

type EventChan = <-chan empty

// GetChan returns the channel that will be closed when the event is set.
func (e *Event) GetChan() EventChan {
	e.l.RLock()
	defer e.l.RUnlock()
	return e.ch
}

// Wait waits for either the event to happen or the given context to be done.
// If the context is done first, the error is returned, otherwise the return value is nil.
func (e *Event) Wait(ctx context.Context) error {
	select {
	case <-e.GetChan():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitTimeout waits for either the event to happen within the given timeout.
// If the timeout expires first, the return value is false, otherwise it's true.
func (e *Event) WaitTimeout(timeout time.Duration) bool {
	select {
	case <-e.GetChan():
		return true
	case <-time.After(timeout):
		return false
	}
}

// IsSet returns true if the event has been set.
func (e *Event) IsSet() bool {
	e.l.RLock()
	defer e.l.RUnlock()
	return e.set
}

// Set sets the event, notifying all waiters.
func (e *Event) Set() {
	e.l.Lock()
	defer e.l.Unlock()
	if !e.set {
		close(e.ch)
		e.set = true
	}
}

// Clear clears the event, making it unset. Future calls to Wait will now block until Set is called again.
// If the event is not already set, this is a no-op and existing calls to Wait will keep working.
func (e *Event) Clear() {
	e.l.Lock()
	defer e.l.Unlock()
	if e.set {
		e.ch = make(chan empty)
		e.set = false
	}
}
