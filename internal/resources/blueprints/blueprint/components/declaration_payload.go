// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package components

// AppleDeclarationsBehaviour documents what the provider checks in a declaration payload and why,
// appended to the payload attribute description of both declaration-bearing components. It covers
// what an author cannot learn from a diagnostic: that the platform itself validates none of this,
// so the check exists only during plan.
//
// It sits in this package because both surfaces need it and only one of them lives here:
// custom_declarations is an object component whose schema is in this package, while
// apple_declarations is a list the blueprint package declares inline. The blueprint package imports
// this one; the reverse would be a cycle.
const AppleDeclarationsBehaviour = " Jamf Pro stores a payload without validating it and drops any " +
	"key it does not recognise, so a misspelled key never reaches a device. The provider checks each " +
	"payload against Apple's schemas during `plan` and reports an unrecognised or miscased key, a " +
	"wrong value type, a missing required key, a value outside a declared set, and a number outside " +
	"a declared range. The schemas cover Apple's release and current seed branches, so they include " +
	"keys Apple has published but not yet released, and they are embedded in the provider release " +
	"you have installed: a key newer than that release reads as unrecognised until you upgrade the " +
	"provider. To skip the check, use `raw_component`."
