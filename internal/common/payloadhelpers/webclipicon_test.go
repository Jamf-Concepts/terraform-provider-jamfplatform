// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The four fixtures are a live capture, not a construction: two icons were
// written to a Jamf Pro 11.3x tenant through
// jamfplatform_pro_mobile_device_configuration_profile on 2026-09-09 and read
// back, so *_jamf_rendered_*.png is byte-for-byte what the server stored for
// the matching *_authored_*.png. Icon A is a 360x360 colour checkerboard with a
// diagonal gradient (downscaled by the server); icon B is a 120x120
// concentric-ring pattern (upscaled), the adversarial case for a resampler.
const (
	authoredA = "webclip_icon_authored_a.png"
	renderedA = "webclip_icon_jamf_rendered_a.png"
	authoredB = "webclip_icon_authored_b.png"
	renderedB = "webclip_icon_jamf_rendered_b.png"
)

func icon(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return b
}

func TestIconsEquivalent_JamfRenderOfTheAuthoredIcon(t *testing.T) {
	for _, tc := range []struct{ name, authored, rendered string }{
		{"downscaled 360x360 checkerboard", authoredA, renderedA},
		{"upscaled 120x120 ring pattern", authoredB, renderedB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, r := icon(t, tc.authored), icon(t, tc.rendered)
			if bytes.Equal(a, r) {
				t.Fatal("fixture pair is byte-identical, so it no longer exercises the re-render")
			}
			if !iconsEquivalent(a, r) {
				t.Errorf("Jamf Pro's own re-render of this icon must compare equal, got not equal")
			}
		})
	}
}

func TestIconsEquivalent_DifferentPicture(t *testing.T) {
	// Cross-pairing an authored icon with the server's render of the *other*
	// icon is the case that must still be reported: a real icon change.
	if iconsEquivalent(icon(t, authoredA), icon(t, renderedB)) {
		t.Error("icon A must not compare equal to Jamf Pro's render of icon B")
	}
	if iconsEquivalent(icon(t, authoredB), icon(t, renderedA)) {
		t.Error("icon B must not compare equal to Jamf Pro's render of icon A")
	}
}

// TestIconsEquivalent_ToleranceHeadroom pins the measurement the tolerance was
// chosen from, so a change to the grid, the box filter or the threshold has to
// keep the two populations an order of magnitude apart rather than merely
// keeping the assertions above green.
func TestIconsEquivalent_ToleranceHeadroom(t *testing.T) {
	norm := func(name string) []float64 {
		v, ok := normaliseIcon(icon(t, name))
		if !ok {
			t.Fatalf("%s did not normalise", name)
		}
		return v
	}
	same := max(
		meanChannelDelta(norm(authoredA), norm(renderedA)),
		meanChannelDelta(norm(authoredB), norm(renderedB)),
	)
	different := min(
		meanChannelDelta(norm(authoredA), norm(renderedB)),
		meanChannelDelta(norm(authoredB), norm(renderedA)),
	)
	if same > iconMeanDeltaTolerance/2 {
		t.Errorf("re-render noise %.2f leaves less than half the tolerance %.2f as headroom", same, iconMeanDeltaTolerance)
	}
	if different < iconMeanDeltaTolerance*2 {
		t.Errorf("two different icons score %.2f, within a factor of two of the tolerance %.2f", different, iconMeanDeltaTolerance)
	}
}

// TestIconsEquivalent_PinsTheTolerance calibrates the threshold from the third
// direction the fixture pair cannot reach. The two fixtures pin re-render noise
// from below and a wholly different icon from above, which between them leave
// the tolerance free to be raised several-fold without failing anything. A
// synthetic badge painted over a known fraction of the icon supplies a
// measurement in the gap: the smaller one is inside the tolerance and is the
// sensitivity floor this comparison honestly has, the larger one is outside it
// and must stay outside, so doubling iconMeanDeltaTolerance fails here.
func TestIconsEquivalent_PinsTheTolerance(t *testing.T) {
	const size = 240
	base := gradient(size, size)
	badged := func(denominator int) image.Image {
		out := image.NewRGBA(base.Bounds())
		draw.Draw(out, out.Bounds(), base, image.Point{}, draw.Src)
		side := size / denominator
		draw.Draw(out, image.Rect(size-side, size-side, size, size), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
		return out
	}
	encode := func(img image.Image) []byte {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	norm := func(blob []byte) []float64 {
		v, ok := normaliseIcon(blob)
		if !ok {
			t.Fatal("synthetic icon did not normalise")
		}
		return v
	}
	basePNG := encode(base)
	for _, tc := range []struct {
		name             string
		denominator      int
		low, high        float64
		wantedEquivalent bool
	}{
		{"badge over 2.8% of the area reads as unchanged", 6, 2.0, 3.0, true},
		{"badge over 11.1% of the area reads as changed", 3, 9.5, 13.0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			edited := encode(badged(tc.denominator))
			got := meanChannelDelta(norm(basePNG), norm(edited))
			if got < tc.low || got > tc.high {
				t.Errorf("mean delta %.2f is outside the pinned band [%.2f, %.2f]; the grid or the box filter changed", got, tc.low, tc.high)
			}
			if iconsEquivalent(basePNG, edited) != tc.wantedEquivalent {
				t.Errorf("mean delta %.2f against tolerance %.2f: equivalent = %t, want %t", got, iconMeanDeltaTolerance, !tc.wantedEquivalent, tc.wantedEquivalent)
			}
		})
	}
}

func TestIconsEquivalent_UndecodableAndEmpty(t *testing.T) {
	real := icon(t, renderedA)
	notAnImage := []byte("this is not an image at all, not even close")
	for _, tc := range []struct {
		name             string
		authored, stored []byte
	}{
		{"authored is not an image", notAnImage, real},
		{"stored is not an image", real, notAnImage},
		{"two different undecodable blobs", notAnImage, []byte("a different blob that is also not an image")},
		{"authored empty", nil, real},
		{"stored empty", real, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if iconsEquivalent(tc.authored, tc.stored) {
				t.Error("an icon the provider cannot decode must never compare equal")
			}
		})
	}
	// Byte-identical sides are the one exception, empty ones included: nothing
	// was re-rendered, so there is no difference to report.
	if !iconsEquivalent(notAnImage, notAnImage) {
		t.Error("byte-identical blobs must compare equal without being decoded")
	}
	if !iconsEquivalent(nil, nil) {
		t.Error("two absent icons must compare equal")
	}
	if !iconsEquivalent([]byte{}, []byte{}) {
		t.Error("two empty icon blobs must compare equal")
	}
}

// TestIconsEquivalent_AcrossFormats covers the conversion Jamf Pro performs on
// a JPEG or GIF icon: it stores a PNG either way, so the two sides of a
// comparison are routinely in different formats.
func TestIconsEquivalent_AcrossFormats(t *testing.T) {
	src := gradient(240, 240)
	var asPNG, asJPEG, asGIF bytes.Buffer
	if err := png.Encode(&asPNG, src); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&asJPEG, src, &jpeg.Options{Quality: 92}); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&asGIF, src, nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		blob []byte
	}{
		{"jpeg against png", asJPEG.Bytes()},
		{"gif against png", asGIF.Bytes()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !iconsEquivalent(tc.blob, asPNG.Bytes()) {
				t.Error("the same picture in another format must compare equal")
			}
		})
	}
}

// TestNormaliseIcon_RefusesAnOversizedImage keeps the decode bounded: the
// dimension cap is read from the header, so an enormous declared size is
// refused without allocating for it.
func TestNormaliseIcon_RefusesAnOversizedImage(t *testing.T) {
	blob := icon(t, renderedA)
	oversized := withDeclaredSize(t, blob, iconMaxDimension+1, iconMaxDimension+1)
	if _, ok := normaliseIcon(oversized); ok {
		t.Errorf("an icon declaring %dx%d must be refused", iconMaxDimension+1, iconMaxDimension+1)
	}
}

// TestLenientEqualPlist_WebClipIcon wires the equivalence through the
// comparison the resources actually call, including the negative direction and
// the neighbouring keys, which must still compare exactly.
func TestLenientEqualPlist_WebClipIcon(t *testing.T) {
	entry := func(icon []byte, label string) map[string]any {
		return map[string]any{
			"PayloadContent": []any{
				map[string]any{
					"PayloadType": webClipPayloadType,
					"Label":       label,
					"URL":         "https://example.com/",
					"Icon":        icon,
				},
			},
		}
	}
	a, r, other := icon(t, authoredA), icon(t, renderedA), icon(t, renderedB)

	if !LenientEqualPlist(entry(a, "Portal"), entry(r, "Portal")) {
		t.Error("a web clip whose icon Jamf Pro re-rendered must compare equal")
	}
	if LenientEqualPlist(entry(a, "Portal"), entry(other, "Portal")) {
		t.Error("a web clip whose icon was replaced must not compare equal")
	}
	if LenientEqualPlist(entry(a, "Portal"), entry(r, "Renamed")) {
		t.Error("the icon exception must not extend to the other keys of the entry")
	}
	// The same blobs under any other payload type stay a byte comparison: a
	// re-rendered certificate is corruption, not a tolerated normalisation.
	notAWebClip := func(blob []byte) map[string]any {
		return map[string]any{
			"PayloadContent": []any{
				map[string]any{"PayloadType": "com.apple.security.root", "Icon": blob},
			},
		}
	}
	if LenientEqualPlist(notAWebClip(a), notAWebClip(r)) {
		t.Error("the icon exception must be scoped to com.apple.webClip.managed")
	}
}

// TestDiffPayloadTrees_NamesTheIcon is the regression for issue #418: the
// differ walked authored *string* leaves only, so a diverging icon produced no
// findings at all and the operator got the unattributed message with a list of
// causes that had nothing to do with it.
func TestDiffPayloadTrees_NamesTheIcon(t *testing.T) {
	authored := map[string]any{
		"PayloadContent": []any{
			map[string]any{"PayloadType": webClipPayloadType, "Icon": icon(t, authoredA)},
		},
	}
	stored := map[string]any{
		"PayloadContent": []any{
			map[string]any{"PayloadType": webClipPayloadType, "Icon": icon(t, renderedB)},
		},
	}
	findings := diffPayloadTrees(authored, stored)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding, got %d: %+v", len(findings), findings)
	}
	if got, want := findings[0].path, "PayloadContent[0].Icon"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if findings[0].class != classIcon {
		t.Errorf("class = %v, want classIcon", findings[0].class)
	}
	if !findings[0].described {
		t.Error("a data-blob finding must be marked described so the diagnostic does not excerpt the summary")
	}
}

// TestDiffPayloadTrees_TolerantOfServerInjectedKeys guards the other half of
// issue #418: data_files, the integer handle Jamf Pro assigns a stored icon, is
// on the stored side only and must not be reported — the intersection compare
// already tolerates it, and naming it would send the operator after a key they
// cannot author.
func TestDiffPayloadTrees_TolerantOfServerInjectedKeys(t *testing.T) {
	blob := icon(t, authoredA)
	authored := map[string]any{
		"PayloadContent": []any{
			map[string]any{"PayloadType": webClipPayloadType, "Icon": blob},
		},
	}
	stored := map[string]any{
		"PayloadContent": []any{
			map[string]any{
				"PayloadType": webClipPayloadType,
				"Icon":        icon(t, renderedA),
				"data_files":  []any{uint64(1554432350)},
				"Precomposed": false,
			},
		},
	}
	if findings := diffPayloadTrees(authored, stored); len(findings) != 0 {
		t.Errorf("expected no findings, got %+v", findings)
	}
	if !LenientEqualPlist(authored, stored) {
		t.Error("server-injected keys beside a re-rendered icon must still compare equal")
	}
}

// TestDiffPayloadTrees_NamesNonStringLeaves covers the general hardening: any
// non-string leaf that differs is now attributed, whatever its type.
func TestDiffPayloadTrees_NamesNonStringLeaves(t *testing.T) {
	authored := map[string]any{
		"PayloadContent": []any{
			map[string]any{
				"PayloadType":            "com.apple.wifi.managed",
				"AutoJoin":               true,
				"Priority":               uint64(3),
				"PayloadCertificateUUID": []byte{1, 2, 3},
			},
		},
	}
	stored := map[string]any{
		"PayloadContent": []any{
			map[string]any{
				"PayloadType":            "com.apple.wifi.managed",
				"AutoJoin":               false,
				"Priority":               uint64(7),
				"PayloadCertificateUUID": []byte{4, 5, 6, 7},
			},
		},
	}
	findings := diffPayloadTrees(authored, stored)
	got := map[string]string{}
	for _, f := range findings {
		got[f.path] = f.authored + " -> " + f.stored
	}
	for _, want := range []string{
		"PayloadContent[0].AutoJoin",
		"PayloadContent[0].Priority",
		"PayloadContent[0].PayloadCertificateUUID",
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("%s was not reported; findings: %+v", want, got)
		}
	}
	if len(findings) != 3 {
		t.Errorf("expected 3 findings, got %d: %+v", len(findings), got)
	}
}

func TestDescribeLeaf(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "(nothing)"},
		{"bool", true, "true"},
		{"integer", uint64(7), "7"},
		{"list", []any{1, 2}, "a list of 2 items"},
		{"dictionary", map[string]any{"a": 1}, "a dictionary of 1 key"},
		{"undecodable data", []byte("nope"), "4 bytes of data (not a readable image)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := describeLeaf(tc.in); got != tc.want {
				t.Errorf("describeLeaf() = %q, want %q", got, tc.want)
			}
		})
	}
	if got, want := describeLeaf(icon(t, renderedA)), "6372 bytes of data (PNG image, 180x180)"; got != want {
		t.Errorf("describeLeaf(icon) = %q, want %q", got, want)
	}
}

func TestLastPathKey(t *testing.T) {
	for in, want := range map[string]string{
		".PayloadContent[0].Icon": "Icon",
		".PayloadContent[0]":      "PayloadContent",
		".Icon":                   "Icon",
		"Icon":                    "Icon",
		"":                        "",
	} {
		if got := lastPathKey(in); got != want {
			t.Errorf("lastPathKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// gradient is a deterministic test picture with enough colour variation that
// two formats of it are distinguishable from two different pictures.
func gradient(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			r := math.Hypot(float64(x-w/2), float64(y-h/2))
			v := uint8(127 + 120*math.Sin(r/11))
			img.Set(x, y, color.RGBA{R: v, G: uint8(255 - int(v)), B: uint8((int(v) * 3) % 256), A: 255})
		}
	}
	return img
}

// withDeclaredSize rewrites a PNG's IHDR width and height, fixing the chunk
// CRC, so the header declares an image far larger than the data. Decoding it
// would fail; the point is that the dimension cap refuses it before that.
func withDeclaredSize(t *testing.T, blob []byte, w, h int) []byte {
	t.Helper()
	out := bytes.Clone(blob)
	if len(out) < 33 || string(out[12:16]) != "IHDR" {
		t.Fatalf("fixture is not a PNG with a leading IHDR chunk")
	}
	for i, v := range []int{w, h} {
		off := 16 + i*4
		out[off] = byte(v >> 24)
		out[off+1] = byte(v >> 16)
		out[off+2] = byte(v >> 8)
		out[off+3] = byte(v)
	}
	crc := crc32IEEE(out[12:29])
	for i := range 4 {
		out[29+i] = byte(crc >> (24 - 8*i))
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(out)); err != nil {
		t.Fatalf("rewritten header does not parse, so the cap is not what is being tested: %v", err)
	}
	return out
}

func crc32IEEE(b []byte) uint32 {
	var table [256]uint32
	for i := range table {
		c := uint32(i)
		for range 8 {
			if c&1 != 0 {
				c = 0xEDB88320 ^ (c >> 1)
			} else {
				c >>= 1
			}
		}
		table[i] = c
	}
	crc := ^uint32(0)
	for _, v := range b {
		crc = table[byte(crc)^v] ^ (crc >> 8)
	}
	return ^crc
}
