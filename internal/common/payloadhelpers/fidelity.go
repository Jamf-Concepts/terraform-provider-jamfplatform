// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"fmt"
	"html"
	"slices"
	"strings"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/plisthelpers"
)

// FidelityPhase selects the remediation sentence for a payload verification
// failure: a failed create is rolled back, a failed update is not.
type FidelityPhase int

const (
	// FidelityPhaseCreate — the profile has been deleted again, nothing is left behind.
	FidelityPhaseCreate FidelityPhase = iota
	// FidelityPhaseUpdate — the profile in Jamf Pro now differs from the configuration.
	FidelityPhaseUpdate
	// FidelityPhaseImport — nothing has been written; the profile was refused at
	// the import gate because a later write-back would corrupt it (see importgate.go).
	FidelityPhaseImport
)

// maxReportedFindings caps how many diverging values the diagnostic names:
// server-injected PayloadContent entries can shift array indices and turn one
// real defect into a column of noise.
const maxReportedFindings = 3

// fidelityClass is the kind of mangling Jamf Pro applied to one value. Each
// class carries a different remedy — see remedyFor.
type fidelityClass int

const (
	classLineBreak   fidelityClass = iota // line feeds and tabs deleted
	classEntityLayer                      // extra entity layer around "&" or "<" (PI-827)
	classAstral                           // non-BMP characters replaced, or the enclosing dict dropped
	classDropped                          // value absent from the stored payload
	classOther                            // unexplained
)

type fidelityFinding struct {
	path     string
	class    fidelityClass
	authored string
	stored   string
	present  bool
}

// PayloadFidelityErrorDetail builds the diagnostic detail for a read-back
// verification failure: it diffs the payload the configuration supplied
// against the one Jamf Pro stored, names each diverging value by its plist
// path, quotes both forms around the first difference, and gives the remedy
// for that class of mangling. Keys the mask drops (identifiers, display names)
// are skipped — Jamf Pro rewrites those by design.
func PayloadFidelityErrorDetail(authored, stored []byte, phase FidelityPhase) string {
	findings, ok := diffPayloadStrings(authored, stored)
	if !ok || len(findings) == 0 {
		return unattributedFidelityDetail(phase)
	}
	return formatFidelityFindings(findings, phase)
}

// formatFidelityFindings renders a non-empty finding list as diagnostic detail.
// Shared by the post-write checks (which diff against what Jamf Pro actually
// stored) and the import gate (which diffs against the form a write-back is
// predicted to produce), so both speak with one voice.
func formatFidelityFindings(findings []fidelityFinding, phase FidelityPhase) string {
	var b strings.Builder
	b.WriteString(findingsPreamble(len(findings), phase))

	shown := findings
	if len(shown) > maxReportedFindings {
		shown = shown[:maxReportedFindings]
	}
	suppliedLabel, storedLabel := "supplied", "stored  "
	if phase == FidelityPhaseImport {
		// Nothing was supplied and nothing is stored yet — the two columns are the
		// profile as it exists today and the form a write-back would leave behind.
		suppliedLabel, storedLabel = "in Jamf Pro now  ", "after a Terraform write"
	}
	for _, f := range shown {
		fmt.Fprintf(&b, "\n  - %s\n", wrapIndented(f.path+" — "+remedyFor(f.class), "    "))
		fmt.Fprintf(&b, "    %s: %s\n", suppliedLabel, excerpt(f.authored, f.stored))
		if f.present {
			fmt.Fprintf(&b, "    %s: %s\n", storedLabel, excerpt(f.stored, f.authored))
		} else {
			fmt.Fprintf(&b, "    %s: (nothing — the value is absent)\n", storedLabel)
		}
	}
	if len(findings) > len(shown) {
		fmt.Fprintf(&b, "\n%d further value(s) also differ.\n", len(findings)-len(shown))
	}

	b.WriteString("\n")
	b.WriteString(remediationTail(phase))
	return b.String()
}

// findingsPreamble is the opening sentence, which differs in tense: the
// post-write phases report what Jamf Pro did, the import phase reports what it
// would do.
func findingsPreamble(n int, phase FidelityPhase) string {
	if phase == FidelityPhaseImport {
		if n == 1 {
			return "This profile holds a payload value Jamf Pro cannot store back as it stands.\n"
		}
		return fmt.Sprintf("This profile holds %d payload values Jamf Pro cannot store back as they stand.\n", n)
	}
	if n == 1 {
		return "Jamf Pro stored a payload value differently than this configuration supplied.\n"
	}
	return fmt.Sprintf("Jamf Pro stored %d payload values differently than this configuration supplied.\n", n)
}

// remedyFor is the per-class explanation and fix, worded to name the
// representation to use rather than just report that a defect exists.
func remedyFor(c fidelityClass) string {
	switch c {
	case classLineBreak:
		return "line breaks removed. Jamf Pro discards line feeds and tabs in this position, so the words either side run together. " +
			"Write a line break as the character reference &#13; (a carriage return — the form the Jamf Pro admin UI itself writes) or &#8232;, " +
			"or move the value into an \"Application & Custom Settings\" payload, which keeps line feeds as supplied. " +
			"Do not decoratively line-wrap a long value: every line feed and indent tab in it is dropped."
	case classEntityLayer:
		return "an extra layer of XML escaping kept around \"&\" or \"<\" (Jamf product issue PI-827), so a device would receive \"&amp;\" where \"&\" was intended. " +
			"No client can work around this. Remove \"&\" and \"<\" from the value, or move it into an \"Application & Custom Settings\" payload, which stores them correctly."
	case classAstral:
		return "characters outside the basic multilingual plane (emoji, for example) replaced or the enclosing dictionary dropped. " +
			"macOS itself handles these correctly, so this is a Jamf Pro limitation with no client-side workaround — remove them from the value."
	case classDropped:
		return "not stored at all."
	default:
		return "stored with different content, for a reason the provider does not recognise. Compare the two forms below."
	}
}

func remediationTail(phase FidelityPhase) string {
	switch phase {
	case FidelityPhaseCreate:
		return "The profile just created has been rolled back, so nothing is left behind in Jamf Pro. " +
			"Correct the value(s) above and apply again, or manage this profile outside Terraform."
	case FidelityPhaseImport:
		return "Nothing has been imported and nothing in Jamf Pro has changed. Importing would leave Terraform " +
			"managing a profile it cannot write back: the first apply that touches this resource — even a change to " +
			"an unrelated field — would rewrite the payload into the corrupted form shown above, and the Classic API " +
			"then refuses to accept the original value back, so the damage can only be undone in the Jamf Pro admin " +
			"UI. Either remove the character(s) named above in the admin UI and import again, move the value into an " +
			"\"Application & Custom Settings\" payload (which Jamf Pro stores faithfully), or leave this profile out " +
			"of Terraform and read it with the jamfplatform_pro_macos_configuration_profile / " +
			"jamfplatform_pro_mobile_device_configuration_profile data source instead. There is deliberately no " +
			"override: a profile Jamf Pro cannot store faithfully cannot be managed safely. If you believe this " +
			"particular payload does round-trip cleanly, please report it with the PayloadType named above so the " +
			"provider's storage-category table can be corrected for everyone."
	default:
		return "The profile in Jamf Pro now holds the stored form shown above, so it no longer matches this configuration. " +
			"Correct the value(s) and apply again, or manage this profile outside Terraform."
	}
}

func unattributedFidelityDetail(phase FidelityPhase) string {
	return "Jamf Pro stored the payload differently than this configuration supplied, and the provider could not attribute the difference to a single value. " +
		"Common causes are line feeds or tabs inside a string value (deleted by the payload types Jamf Pro stores as-is — use &#13; instead), " +
		"\"&\" or \"<\" inside a string value (kept with an extra layer of escaping — Jamf product issue PI-827), " +
		"and characters outside the basic multilingual plane such as emoji (replaced or dropped). " +
		remediationTail(phase)
}

// diffPayloadStrings flattens both payloads to their string leaves and returns
// every leaf Jamf Pro did not store as supplied. Only leaves the configuration
// supplied are examined — extra leaves on the stored side are Jamf Pro's own
// injections. ok=false when either side will not parse, so the caller falls
// back to the generic text rather than naming a culprit it cannot identify.
func diffPayloadStrings(authored, stored []byte) ([]fidelityFinding, bool) {
	authoredTree, _, err := plisthelpers.ParsePlist(authored)
	if err != nil {
		return nil, false
	}
	storedTree, _, err := plisthelpers.ParsePlist(stored)
	if err != nil {
		return nil, false
	}
	return diffPayloadTrees(authoredTree, storedTree), true
}

// diffPayloadTrees is the comparison itself, over already-parsed trees. Split
// out so the import gate can diff a payload against a *predicted* stored form it
// builds in memory (see importgate.go) using exactly the comparison, mask and
// classifier the post-write checks use — the two can therefore never disagree
// about whether a given value survives a write.
func diffPayloadTrees(authoredTree, storedTree map[string]any) []fidelityFinding {
	authoredFlat := map[string]string{}
	storedFlat := map[string]string{}
	flattenStringLeaves("", authoredTree, authoredFlat)
	flattenStringLeaves("", storedTree, storedFlat)

	findings := make([]fidelityFinding, 0, 4)
	for _, path := range sortedKeys(authoredFlat) {
		if maskedLeafPath(path) {
			continue
		}
		want := authoredFlat[path]
		got, present := storedFlat[path]
		if present && strings.TrimSpace(normalizeLineEndings(got)) == strings.TrimSpace(normalizeLineEndings(want)) {
			continue
		}
		findings = append(findings, fidelityFinding{
			path:     strings.TrimPrefix(path, "."),
			class:    classify(want, got, present),
			authored: want,
			stored:   got,
			present:  present,
		})
	}
	return findings
}

// classify picks the wire law that explains one divergence. Order is
// load-bearing: astral characters can *cause* a dropped dictionary, so they
// are tested before absence.
func classify(authored, stored string, present bool) fidelityClass {
	if hasAstral(authored) && (!present || strings.ContainsRune(stored, '�')) {
		return classAstral
	}
	if !present {
		return classDropped
	}
	if strings.ContainsAny(authored, "\n\t") && stripDeletedWhitespace(authored) == strings.TrimSpace(stored) {
		return classLineBreak
	}
	if html.UnescapeString(stored) == authored {
		return classEntityLayer
	}
	return classOther
}

// stripDeletedWhitespace removes exactly what Jamf Pro deletes from a
// verbatim-stored value: line feeds and tabs. Carriage returns survive, which
// is why they are the recommended representation.
func stripDeletedWhitespace(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\n", "", "\t", "").Replace(s))
}

func hasAstral(s string) bool {
	for _, r := range s {
		if r > 0xFFFF {
			return true
		}
	}
	return false
}

// wrapIndented hard-wraps text to a terminal-friendly width, indenting every
// line after the first. Terraform re-wraps flush-left paragraphs in a
// diagnostic but leaves indented lines exactly as given, so a long bullet
// has to arrive pre-wrapped or it runs off the screen.
func wrapIndented(text, indent string) string {
	const width = 72
	var (
		b    strings.Builder
		line int
	)
	for i, word := range strings.Fields(text) {
		switch {
		case i == 0:
			b.WriteString(word)
			line = len(word)
		case line+1+len(word) > width:
			b.WriteString("\n" + indent + word)
			line = len(indent) + len(word)
		default:
			b.WriteString(" " + word)
			line += 1 + len(word)
		}
	}
	return b.String()
}

// excerpt quotes s around its first difference from other, Go-quoted so the
// invisible characters this diagnostic is about show as \n, \t and \r. Left
// unwrapped — a wrapped quoted string is harder to compare than one that
// overflows slightly.
func excerpt(s, other string) string {
	const context = 32
	at := firstDifference(s, other)
	start := max(at-context, 0)
	end := min(at+context, len(s))
	// Keep the window on rune boundaries so quoting cannot split a character.
	for start > 0 && !utf8Start(s[start]) {
		start--
	}
	for end < len(s) && !utf8Start(s[end]) {
		end++
	}
	out := fmt.Sprintf("%q", s[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(s) {
		out += "…"
	}
	return out
}

func firstDifference(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// utf8Start reports whether b can begin a UTF-8 encoded rune.
func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

// maskedLeafPath reports whether a flattened path ends in a key the mask drops
// on both sides — Jamf Pro rewrites those on every write, so they are never
// the reason verification failed.
func maskedLeafPath(path string) bool {
	key := path
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		key = path[i+1:]
	}
	if _, masked := maskedTopLevelKeys[key]; masked {
		return true
	}
	_, masked := maskedPayloadContentKeys[key]
	return masked
}

// flattenStringLeaves records every string leaf keyed by its plist path
// (`.PayloadContent[0].LoginwindowText`).
func flattenStringLeaves(prefix string, v any, out map[string]string) {
	switch t := v.(type) {
	case string:
		out[prefix] = t
	case map[string]any:
		for k, val := range t {
			flattenStringLeaves(prefix+"."+k, val, out)
		}
	case []any:
		for i, item := range t {
			flattenStringLeaves(fmt.Sprintf("%s[%d]", prefix, i), item, out)
		}
	}
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
