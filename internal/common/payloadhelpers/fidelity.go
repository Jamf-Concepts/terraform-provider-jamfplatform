// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"maps"
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
	classIcon                             // a web clip icon that is not a re-render of the authored one
	classOther                            // unexplained
)

type fidelityFinding struct {
	path     string
	class    fidelityClass
	authored string
	stored   string
	present  bool
	// described marks a finding whose two sides are already summaries rather
	// than the values themselves (a data blob, a list, a dictionary). Those are
	// printed as they stand: excerpting a summary around its first differing
	// character quotes half a sentence and reads as corruption.
	described bool
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
		authored, stored := excerpt(f.authored, f.stored), excerpt(f.stored, f.authored)
		if f.described {
			authored, stored = f.authored, f.stored
		}
		fmt.Fprintf(&b, "    %s: %s\n", suppliedLabel, authored)
		if f.present {
			fmt.Fprintf(&b, "    %s: %s\n", storedLabel, stored)
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
	case classIcon:
		return "stored as a different picture. Jamf Pro rescales a web clip icon to 180 pixels on its longest side and re-encodes it as a PNG, " +
			"and the provider allows for that, so the difference here is in the image itself. " +
			"Either Jamf Pro could not read the icon and stored a placeholder in its place (it reads PNG, JPEG and GIF), " +
			"or someone replaced the icon in the Jamf Pro admin UI. " +
			"Supply an icon Jamf Pro can read, or copy the stored icon back into the payload."
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
	aligned := alignPayloadContentOrder(authoredTree, dropInjectedPayloadEntries(storedTree))

	authoredFlat := map[string]string{}
	storedFlat := map[string]string{}
	flattenStringLeaves("", authoredTree, authoredFlat)
	flattenStringLeaves("", aligned, storedFlat)

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
	// Non-string leaves are reported after the string ones: the string classes
	// carry a specific remedy, so they lead. Reporting these at all is what
	// keeps a difference in a data blob, a boolean or a number from producing an
	// unattributed failure — the whole tree can compare unequal while the string
	// pass finds nothing, which is exactly how a re-rendered web clip icon used
	// to surface (issue #418).
	findings = append(findings, diffNonStringLeaves("", "", authoredTree, aligned)...)
	return findings
}

// diffNonStringLeaves walks both trees in parallel and reports every non-string
// leaf that would make LenientEqualPlist answer false: intersection semantics
// (a key on only one side is Jamf Pro's own injection or the user's omission,
// neither of which is a fidelity failure), the same numeric leniency, and the
// same web clip icon equivalence. Strings are skipped — the flatten-based pass
// above owns those, and it classifies them.
//
// Intersection semantics hold for ordinary dict keys — a key on only one side
// is Jamf Pro's own injection or the operator's omission, neither of which is a
// fidelity failure — with the one exception LenientEqualPlist itself carves
// out: an MCX entry's inner PayloadContent, handled by diffMCXPreferences. A
// dict on the authored side facing a non-dict on the stored side is likewise
// blamed rather than skipped, because the equality check fails the whole tree
// on it.
//
// payloadType is the PayloadType of the enclosing payload entry, threaded down
// so the icon exception can be applied to exactly the leaf it belongs to.
func diffNonStringLeaves(path, payloadType string, authored, stored any) []fidelityFinding {
	switch av := authored.(type) {
	case string:
		return nil

	case map[string]any:
		bv, ok := stored.(map[string]any)
		if !ok {
			return []fidelityFinding{describedFinding(path, classOther, authored, stored)}
		}
		ownType, _ := av["PayloadType"].(string)
		entryType := payloadType
		if ownType != "" {
			entryType = ownType
		}
		_, isMCX := mcxLikePayloadTypes[ownType]
		var out []fidelityFinding
		if isMCX {
			out = append(out, diffMCXPreferences(path, av, bv)...)
		}
		for _, k := range slices.Sorted(maps.Keys(av)) {
			vb, exists := bv[k]
			if !exists || maskedLeafPath(k) {
				continue
			}
			if isMCX && k == "PayloadContent" {
				continue
			}
			out = append(out, diffNonStringLeaves(path+"."+k, entryType, av[k], vb)...)
		}
		return out

	case []any:
		bv, ok := stored.([]any)
		if !ok || len(av) != len(bv) {
			return []fidelityFinding{describedFinding(path, classOther, authored, stored)}
		}
		var out []fidelityFinding
		for i := range av {
			out = append(out, diffNonStringLeaves(fmt.Sprintf("%s[%d]", path, i), payloadType, av[i], bv[i])...)
		}
		return out

	default:
		if leafEquivalent(payloadType, lastPathKey(path), authored, stored) {
			return nil
		}
		return []fidelityFinding{describedFinding(path, leafClass(payloadType, lastPathKey(path)), authored, stored)}
	}
}

// diffMCXPreferences reports the divergences LenientEqualPlist's MCX branch
// fails on but an intersection walk cannot see. That branch treats an
// "Application & Custom Settings" entry's inner PayloadContent — opaque vendor
// preference data Jamf Pro only transports — strictly: the key has to be
// present on both sides or neither, and when both carry it the subtree goes
// through plisthelpers.Equal, which requires matching keysets at every depth.
// A one-sided key there is real drift, and it is exactly what a walk skipping
// keys the stored side lacks steps over, leaving the whole failure
// unattributed (issue #418).
//
// Value differences at keys both sides carry are deliberately left to the
// string pass and the ordinary non-string walk, which already name them with a
// classified remedy — naming them a second time here would push the real
// finding past maxReportedFindings.
func diffMCXPreferences(path string, authored, stored map[string]any) []fidelityFinding {
	ai, aHas := authored["PayloadContent"]
	bi, bHas := stored["PayloadContent"]
	switch {
	case aHas && !bHas:
		return []fidelityFinding{describedFinding(path+".PayloadContent", classDropped, ai, nil)}
	case bHas && !aHas:
		return []fidelityFinding{describedFinding(path+".PayloadContent", classOther, nil, bi)}
	case !aHas:
		return nil
	}
	if plisthelpers.Equal(ai, bi) {
		return nil
	}
	return oneSidedPreferenceKeys(path+".PayloadContent", ai, bi)
}

// oneSidedPreferenceKeys walks a vendor preference subtree both ways and names
// every key that exists on one side only, in either direction: a key the
// operator authored and Jamf Pro did not store, and a key Jamf Pro holds that
// the configuration never wrote. Both fail the strict compare, so both have to
// be nameable. Arrays are walked only over their common prefix — a length
// mismatch is already reported by the caller's own array branch.
func oneSidedPreferenceKeys(path string, authored, stored any) []fidelityFinding {
	switch av := authored.(type) {
	case map[string]any:
		bv, ok := stored.(map[string]any)
		if !ok {
			return nil
		}
		var out []fidelityFinding
		for _, k := range slices.Sorted(maps.Keys(av)) {
			if vb, exists := bv[k]; exists {
				out = append(out, oneSidedPreferenceKeys(path+"."+k, av[k], vb)...)
				continue
			}
			out = append(out, describedFinding(path+"."+k, classDropped, av[k], nil))
		}
		for _, k := range slices.Sorted(maps.Keys(bv)) {
			if _, exists := av[k]; !exists {
				out = append(out, describedFinding(path+"."+k, classOther, nil, bv[k]))
			}
		}
		return out

	case []any:
		bv, ok := stored.([]any)
		if !ok {
			return nil
		}
		var out []fidelityFinding
		for i := range min(len(av), len(bv)) {
			out = append(out, oneSidedPreferenceKeys(fmt.Sprintf("%s[%d]", path, i), av[i], bv[i])...)
		}
		return out

	default:
		return nil
	}
}

// describedFinding builds a finding whose two sides are summaries rather than
// the values themselves (see fidelityFinding.described), which is every
// non-string leaf and every whole subtree this walk can blame.
func describedFinding(path string, class fidelityClass, authored, stored any) fidelityFinding {
	return fidelityFinding{
		path:      strings.TrimPrefix(path, "."),
		class:     class,
		authored:  describeLeaf(authored),
		stored:    describeLeaf(stored),
		present:   stored != nil,
		described: true,
	}
}

// leafEquivalent applies the comparison LenientEqualPlist would to one scalar
// leaf, including the web clip icon exception, so the differ can never name a
// leaf the equality check was happy with.
func leafEquivalent(payloadType, key string, authored, stored any) bool {
	if isWebClipIcon(payloadType, key) {
		if a, s, ok := iconBlobs(authored, stored); ok {
			return iconsEquivalent(a, s)
		}
	}
	return LenientEqualPlist(authored, stored)
}

// leafClass names the icon case specifically; every other non-string leaf gets
// the generic class, because Jamf Pro has no known transform on one.
func leafClass(payloadType, key string) fidelityClass {
	if isWebClipIcon(payloadType, key) {
		return classIcon
	}
	return classOther
}

// describeLeaf renders a non-string leaf for the diagnostic. A data blob is
// summarised rather than dumped: an icon is tens of kilobytes of base64 and
// quoting it would bury the finding.
func describeLeaf(v any) string {
	switch t := v.(type) {
	case nil:
		return "(nothing)"
	case []byte:
		if cfg, format, err := image.DecodeConfig(bytes.NewReader(t)); err == nil {
			return fmt.Sprintf("%d bytes of data (%s image, %dx%d)", len(t), strings.ToUpper(format), cfg.Width, cfg.Height)
		}
		return fmt.Sprintf("%d bytes of data (not a readable image)", len(t))
	case []any:
		return fmt.Sprintf("a list of %s", plural(len(t), "item", "items"))
	case map[string]any:
		return fmt.Sprintf("a dictionary of %s", plural(len(t), "key", "keys"))
	default:
		return fmt.Sprint(v)
	}
}

// plural renders a count with the right noun form, so a summary reads as a
// sentence rather than carrying an "(s)".
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// lastPathKey is the final dict key in a flattened path, ignoring any array
// index that follows it.
func lastPathKey(path string) string {
	key := path
	if i := strings.LastIndexByte(key, '.'); i >= 0 {
		key = key[i+1:]
	}
	if i := strings.IndexByte(key, '['); i >= 0 {
		key = key[:i]
	}
	return key
}

// alignPayloadContentOrder returns storedTree with its top-level PayloadContent
// array permuted back into the order authoredTree used, pairing entries by
// PayloadType in order of appearance. Entries with no counterpart keep their
// relative order at the end.
//
// The paths this diagnostic quotes are indexed (`PayloadContent[1].Foo`), so both
// sides have to agree on which entry index 1 is. Jamf Pro reorders the array on
// store — it stably partitions verbatim-stored entries ahead of re-rendered ones
// (see canonicalisePayloadContentOrder for the wire law) — so without this step
// every leaf below a moved entry is diffed against an unrelated payload: values
// that stored perfectly get reported as "not stored at all", a phantom
// PayloadType mismatch appears, and the one real defect is pushed past
// maxReportedFindings. Wire-confirmed 2026-08-11 on an authored
// [MCX, loginwindow] profile carrying a single line feed: four findings reported,
// none of them the line feed.
//
// Aligning to the *authored* order rather than sorting both sides keeps the
// reported indices matching the payload the operator wrote, which is the whole
// point of naming a path in the message.
//
// The returned tree shares everything except the PayloadContent slice with
// storedTree — nothing here mutates the caller's trees, so the import gate's
// predicted tree (which is already in authored order, making this a no-op)
// continues to diff through exactly the comparison the post-write checks use.
func alignPayloadContentOrder(authoredTree, storedTree map[string]any) map[string]any {
	authored, aOK := authoredTree["PayloadContent"].([]any)
	stored, sOK := storedTree["PayloadContent"].([]any)
	if !aOK || !sOK || len(authored) < 2 || len(stored) < 2 {
		return storedTree
	}

	remaining := make(map[string][]any, len(stored))
	var order []string
	for _, entry := range stored {
		pt := payloadTypeOf(entry)
		if _, seen := remaining[pt]; !seen {
			order = append(order, pt)
		}
		remaining[pt] = append(remaining[pt], entry)
	}

	aligned := make([]any, 0, len(stored))
	for _, entry := range authored {
		pt := payloadTypeOf(entry)
		if queue := remaining[pt]; len(queue) > 0 {
			aligned = append(aligned, queue[0])
			remaining[pt] = queue[1:]
		}
	}
	// Anything the authored side had no slot for — an entry Jamf Pro injected, or
	// a type whose count grew — keeps its stored relative order at the end.
	for _, pt := range order {
		aligned = append(aligned, remaining[pt]...)
	}

	out := make(map[string]any, len(storedTree))
	maps.Copy(out, storedTree)
	out["PayloadContent"] = aligned
	return out
}

// dropInjectedPayloadEntries returns tree with every top-level PayloadContent
// entry Jamf Pro injects as a side-effect of a classic-API field removed,
// mirroring the filter MaskPayload applies (see serverInjectedPayloadTypes) so
// the differ and the equality check cannot disagree about which entries exist.
//
// Without it the array lengths differ by one and the array branch of
// diffNonStringLeaves reports a bare length mismatch *instead of* recursing, so
// the one real defect below is never named: a profile setting
// self_service.authorization_password materialises a
// com.apple.profileRemovalPassword entry, and a genuinely diverging web clip
// icon on the same profile came out as "a list of 1 item" against "a list of 2
// items" with no mention of the icon.
//
// The filtered tree feeds the string pass too, which is correct — an injected
// entry's string leaves were never authored, so no finding can belong to them.
// The returned tree shares everything except the PayloadContent slice with
// tree, and tree itself is returned unchanged when nothing is dropped.
func dropInjectedPayloadEntries(tree map[string]any) map[string]any {
	entries, ok := tree["PayloadContent"].([]any)
	if !ok {
		return tree
	}
	kept := make([]any, 0, len(entries))
	for _, entry := range entries {
		if _, injected := serverInjectedPayloadTypes[payloadTypeOf(entry)]; injected {
			continue
		}
		kept = append(kept, entry)
	}
	if len(kept) == len(entries) {
		return tree
	}
	out := make(map[string]any, len(tree))
	maps.Copy(out, tree)
	out["PayloadContent"] = kept
	return out
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
