// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package aischemas

import "time"

// timeout bounds a test that must prove termination.
func timeout() <-chan time.Time {
	return time.After(5 * time.Second)
}
