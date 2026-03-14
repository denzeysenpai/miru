package miru

import (
	"time"
)

// Sleep pauses execution for the given number of milliseconds.
// Example: debug.Sleep(500) // sleeps for 500ms
func (de *Debugger) Delay(ms int) {
	d := time.Duration(ms) * time.Millisecond
	time.Sleep(d)
}
