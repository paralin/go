// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build wasip1

package runtime

import (
	"internal/runtime/sys"
)

// Cooperative scheduling support for WASI reactor builds.
//
// When a Go program is built with -buildmode=c-shared for wasip1,
// it becomes a WASI reactor that exports _initialize instead of _start.
// This file provides go_start_main and go_tick exports that allow the
// host to drive the Go scheduler cooperatively.
//
// Usage from the host:
//
//	instance.exports._initialize()
//	instance.exports.go_start_main()
//	for {
//	    ms := instance.exports.go_tick()
//	    if ms == -1 { break }      // idle, no more work
//	    if ms > 0 { sleep(ms) }    // wait for timer
//	}

var reactorState struct {
	active     bool  // in go_tick call
	mainQueued bool  // go_start_main was called
	result     int32 // return value for go_tick
}

// go_start_main queues the main goroutine for execution.
// It is a no-op if not in library mode or if already called.
//
//go:wasmexport go_start_main
func go_start_main() {
	if !islibrary || reactorState.mainQueued {
		return
	}
	reactorState.mainQueued = true
	go main_main()
}

// go_tick runs the scheduler until it becomes idle or needs to wait for a timer.
//
// Return values:
//   -1: idle, no pending work
//    0: more goroutines runnable
//   >0: milliseconds until next timer
//
//go:wasmexport go_tick
func go_tick() int32 {
	if !islibrary {
		return -1
	}

	reactorState.active = true
	reactorState.result = -1

	// Enter the scheduler. In reactor mode, findRunnable calls
	// reactorScheduleReturn instead of blocking.
	Gosched()

	result := reactorState.result
	reactorState.active = false
	return result
}

// reactorScheduleReturn is called from findRunnable when in reactor mode
// and no goroutines are immediately runnable. It computes when to wake up
// and pauses execution to return control to the host.
func reactorScheduleReturn() {
	if !reactorState.active {
		return
	}

	mp := getg().m
	pp := mp.p.ptr()
	now := nanotime()
	when := pp.timers.wakeTime()

	if when == 0 {
		if !runqempty(pp) || !sched.runq.empty() {
			reactorState.result = 0
		} else {
			reactorState.result = -1
		}
	} else if when <= now {
		reactorState.result = 0
	} else {
		ms := (when - now) / 1_000_000
		if ms <= 0 {
			ms = 1
		}
		if ms > 0x7FFFFFFF {
			ms = 0x7FFFFFFF
		}
		reactorState.result = int32(ms)
	}

	pause(sys.GetCallerSP() - 16)
}

// isReactorMode reports whether go_tick is currently executing.
func isReactorMode() bool {
	return islibrary && reactorState.active
}
