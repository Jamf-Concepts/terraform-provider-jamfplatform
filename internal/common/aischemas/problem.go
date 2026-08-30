// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package aischemas

import "fmt"

// ProblemKind classifies a finding and, with Advisory, says how far to trust it.
type ProblemKind int

const (
	// UnrecognisedKey means the settings carry a key the product's schema does not declare, and the
	// schema accepts undeclared keys. Advisory: the key may have been added to the product after
	// this schema version was cut, in which case it is perfectly valid. But Jamf stores such a key
	// verbatim and the product never applies it, so a genuine typo is silently inert — which is the
	// one failure the service itself will never report.
	UnrecognisedKey ProblemKind = iota
	// UndeclaredKey means the settings carry a key the schema does not declare, and the schema
	// refuses undeclared keys. Jamf rejects the write.
	UndeclaredKey
	// WrongType means a value is not one of the types the schema declares for it.
	WrongType
	// NotInEnum means a value is outside the schema's enumerated set.
	NotInEnum
	// NotConst means a value differs from the single value the schema fixes.
	NotConst
	// MissingRequiredKey means a key the schema marks required is absent.
	MissingRequiredKey
	// OutOfRange means a number falls outside the schema's bounds.
	OutOfRange
	// LengthOutOfRange means a string is shorter or longer than the schema allows.
	LengthOutOfRange
	// ItemCountOutOfRange means an array holds too few or too many entries.
	ItemCountOutOfRange
	// DuplicateItems means an array the schema marks unique repeats an entry.
	DuplicateItems
	// PatternMismatch means a string does not match the schema's pattern.
	PatternMismatch
	// InvalidPropertyName means a key does not satisfy the schema's constraint on key names.
	InvalidPropertyName
	// NoBranchMatches means a value satisfies none of the alternative shapes the schema allows.
	NoBranchMatches
)

// Problem is one finding against a settings payload.
type Problem struct {
	Kind ProblemKind
	// Path is a JSON pointer to the offending value within the settings object, so it reads the same
	// way the service's own validation failures do (`/permissions/allow/0`). Empty for a problem
	// about the settings object itself.
	Path string
	// Detail describes the problem in one sentence, ready to drop into a diagnostic.
	Detail string
}

// Advisory reports whether a problem rests on this schema version being the one the product
// actually honours, rather than on a write the service was observed to refuse. Callers should
// surface an advisory problem as a warning and the rest as errors.
//
// Only UnrecognisedKey is advisory, and the asymmetry is the service's, not a choice: a key the
// schema does not declare is accepted and stored when the schema allows extra keys
// (com.anthropic.claudecode) and rejected outright when it does not (com.openai.codex).
func (p Problem) Advisory() bool {
	return p.Kind == UnrecognisedKey
}

// at returns the problem with a JSON pointer segment appended to its path.
func problemAt(kind ProblemKind, path, format string, args ...any) Problem {
	return Problem{Kind: kind, Path: path, Detail: fmt.Sprintf(format, args...)}
}
