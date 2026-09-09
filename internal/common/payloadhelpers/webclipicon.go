// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package payloadhelpers

import (
	"bytes"
	"image"
	"math"

	// Registered for image.Decode: Jamf Pro accepts an icon in any of these
	// formats and stores a PNG, so the authored and stored sides of a comparison
	// are routinely in different formats.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// webClipPayloadType is the one payload type carrying an icon. Apple's
// device-management schema (Release-v26.4, the snapshot in
// internal/common/appleprofiles) declares exactly thirteen data-typed payload
// keys, and only this one is an image: the others are certificates
// (com.apple.security.{pem,pkcs1,pkcs12,root,scep}, com.apple.eas.account,
// com.apple.MCX.FileVault2, com.apple.systempolicy.rule), a font
// (com.apple.font), two VPN secrets (com.apple.vpn.managed IKEv2.PPK and
// IPSec.SharedSecret) and an SSO group identifier. Wire-probed 2026-09-09
// against Jamf Pro 11.3x in eu: a certificate, a font and a VPN shared secret
// all come back byte-identical, so this exception is genuinely icon-only and
// must not be widened to data blobs in general — a re-rendered certificate
// would be real corruption, and tolerating it would hide that.
const webClipPayloadType = "com.apple.webClip.managed"

// Icon re-render wire law, probed 2026-09-09 against Jamf Pro 11.3x (eu) over
// eight icons on jamfplatform_pro_mobile_device_configuration_profile:
//
//   - Jamf Pro ALWAYS re-renders the icon. It never stores the submitted bytes,
//     not even when they are already a 180x180 PNG (a 415-byte 180x180 RGBA PNG
//     came back as 558 bytes).
//   - The stored form is always a PNG carrying only IHDR/IDAT/IEND — every
//     ancillary chunk is stripped — scaled so its longest side is 180px with the
//     aspect ratio preserved (8x8 -> 180x180 upscaled, 120x120 -> 180x180,
//     200x100 -> 180x90) and the source colour model kept (an RGB source stays
//     colour type 2, an RGBA source stays 6).
//   - A JPEG or GIF is accepted and converted to PNG.
//   - The re-render is deterministic: storing one icon twice yields
//     byte-identical blobs, which is why an icon read back out of Jamf Pro and
//     re-submitted verbatim does round-trip (the byte fast path below).
//   - Bytes that are not an image at all — including an empty blob — are
//     replaced by a fixed 180x180 placeholder PNG. That must NOT compare equal
//     to whatever was authored, and it does not: an undecodable authored side
//     falls through to the byte comparison, which fails, and the operator is
//     told the icon is not a decodable image.
//
// So a byte comparison of an authored icon against the stored one can only ever
// succeed for an icon that came out of Jamf Pro in the first place. Replicating
// the re-render is not an option either — it would mean matching another
// encoder's zlib settings and filter choices exactly. The comparison therefore
// normalises both sides and compares content.
const (
	// iconGrid is the resolution both icons are box-filtered down to before
	// comparison. Small enough that neither side's resampler shows through,
	// large enough to keep two different icons far apart.
	iconGrid = 16

	// iconMeanDeltaTolerance is the mean per-channel difference (0-255) below
	// which two normalised icons count as the same image.
	//
	// Measured on the same probe run, over a 6x6 colour checkerboard with a
	// diagonal gradient and a high-frequency concentric-ring pattern (the
	// adversarial case for a resampler): the same icon compared against Jamf
	// Pro's re-render of it scored a mean delta of 0.65 and 2.38, while two
	// different icons scored 45.1. The threshold sits an order of magnitude
	// above the observed re-render noise and a factor of five below a real icon
	// change, so it does not need retuning for a slightly different resampler
	// on either side.
	//
	// Trade-off: an icon edited into a near-identical variant out of band —
	// a few pixels, an imperceptible colour shift — reads as unchanged. That
	// buys tolerance of a re-render no client can reproduce, and it costs
	// nothing in the common direction: an icon changed in the *configuration*
	// changes the payloads string itself, which Terraform diffs directly.
	iconMeanDeltaTolerance = 8.0

	// iconMaxDimension bounds what is decoded. An icon is at most a few hundred
	// pixels square; anything larger is either a mistake or a decompression
	// bomb, and is left to the byte comparison rather than expanded in memory.
	iconMaxDimension = 4096
)

// iconsEquivalent reports whether a stored web clip icon is Jamf Pro's
// re-render of the authored one. Both sides are decoded and box-filtered to a
// fixed grid, then compared with a tolerance — see the wire law above for why
// neither a byte comparison nor a reproduction of the re-render can work.
//
// It answers false whenever it cannot tell: either side undecodable, absurdly
// large, or empty. That keeps the failure direction safe — an unrecognisable
// icon surfaces as a verification failure the operator is told about, rather
// than being waved through as equivalent.
func iconsEquivalent(authored, stored []byte) bool {
	if len(authored) == 0 || len(stored) == 0 {
		return false
	}
	if bytes.Equal(authored, stored) {
		return true
	}
	a, ok := normaliseIcon(authored)
	if !ok {
		return false
	}
	s, ok := normaliseIcon(stored)
	if !ok {
		return false
	}
	return meanChannelDelta(a, s) <= iconMeanDeltaTolerance
}

// normaliseIcon decodes an icon and box-filters it to an iconGrid x iconGrid
// grid of RGBA channel means, the form two icons are compared in. Scaling both
// sides to the same square grid is what erases Jamf Pro's rescale: the stored
// icon has a different pixel count but the same picture, and the aspect ratio
// it preserves is erased along with it.
func normaliseIcon(blob []byte) ([]float64, bool) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(blob))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > iconMaxDimension || cfg.Height > iconMaxDimension {
		return nil, false
	}
	img, _, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return nil, false
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, false
	}
	out := make([]float64, 0, iconGrid*iconGrid*4)
	for gy := range iconGrid {
		for gx := range iconGrid {
			x0, x1 := b.Min.X+gx*b.Dx()/iconGrid, b.Min.X+(gx+1)*b.Dx()/iconGrid
			y0, y1 := b.Min.Y+gy*b.Dy()/iconGrid, b.Min.Y+(gy+1)*b.Dy()/iconGrid
			x1 = max(x1, x0+1)
			y1 = max(y1, y0+1)
			var sr, sg, sb, sa, n float64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, bl, al := img.At(x, y).RGBA()
					sr += float64(r >> 8)
					sg += float64(g >> 8)
					sb += float64(bl >> 8)
					sa += float64(al >> 8)
					n++
				}
			}
			out = append(out, sr/n, sg/n, sb/n, sa/n)
		}
	}
	return out, true
}

// meanChannelDelta is the mean absolute per-channel difference between two
// normalised icons, on the 0-255 scale the channels themselves use.
func meanChannelDelta(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return math.MaxFloat64
	}
	var sum float64
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return sum / float64(len(a))
}

// iconLeafKey is the key holding a web clip's icon, a direct child of the
// top-level PayloadContent entry declaring the web clip payload type.
const iconLeafKey = "Icon"

// isWebClipIcon reports whether a key inside a payload entry of the given
// PayloadType is that entry's icon — the only leaf iconsEquivalent applies to.
func isWebClipIcon(payloadType, key string) bool {
	return payloadType == webClipPayloadType && key == iconLeafKey
}

// iconBlobs extracts both sides of an icon comparison, ok=false when either is
// not a data blob (an icon authored as, say, a base64 string is a different
// defect and belongs to the ordinary comparison).
func iconBlobs(a, b any) (authored, stored []byte, ok bool) {
	authored, aok := a.([]byte)
	stored, bok := b.([]byte)
	return authored, stored, aok && bok
}
