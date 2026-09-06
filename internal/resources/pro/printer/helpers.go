// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package printer

// Sentinel that the classic /printers endpoint stores when a printer has no
// category assigned. Visible on every GET; the provider decodes it to null
// in state and rejects it as a literal user input to avoid confusion.
const categoryUnassignedSentinel = "No category assigned"
