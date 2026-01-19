// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !wasip1

package runtime

// isReactorMode reports whether go_tick is currently executing.
// Always returns false on non-wasip1 platforms.
func isReactorMode() bool {
	return false
}

// reactorScheduleReturn is a no-op on non-wasip1 platforms.
func reactorScheduleReturn() {
}
