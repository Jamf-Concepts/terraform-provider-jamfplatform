// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ipv4CIDRValidator checks that a string attribute holds an IPv4 CIDR block.
//
// requireNetworkAddress selects between the two constructors below. It is a field
// rather than two types because only the grammar differs, not the diagnostic.
type ipv4CIDRValidator struct {
	requireNetworkAddress bool
}

// IPv4CIDR returns a validator.String enforcing IPv4 CIDR notation.
//
// Why it exists: the Jamf Security Cloud ZTNA app endpoint refuses a bare address
// and an IPv6 block with the same message — `400 [INVALID_FIELD] bareIps[]: IP
// address range must be in CIDR format.` — which is accurate for `10.11.12.13` and
// misleading for `2001:db8::/32`, a perfectly well-formed CIDR block that the
// endpoint simply does not accept (both wire-probed 2026-08-30). Catching the shape
// at plan time lets the second case say what is actually wrong.
//
// A prefix length is required: `10.0.0.0` is refused, `10.0.0.0/32` accepted.
// Colons are rejected outright rather than relying on the parse, for the same
// reason IPv4Address rejects them — `::ffff:10.0.0.0/104` parses and its `To4` is
// non-nil, so the parse alone would let an IPv4-mapped IPv6 block through.
//
// A range must also name its own network address rather than a host inside it.
// `10.99.43.7/24` parses, but Jamf Security Cloud canonicalises it: the write is
// accepted with `204` and the stored value reads back as `10.99.43.0/24`, and a `/0`
// prefix collapses the same way to `0.0.0.0/0` whatever address was written (both
// wire-probed against production EU 2026-08-30). Every attribute fed through this
// validator is Optional or Required rather than Optional+Computed — removing a range
// has to clear it, not leave the server's value in place — so Terraform holds the
// provider to the planned value and the server's rewrite surfaces as `Provider
// produced inconsistent result after apply` partway through the apply, with the
// object already created. Refusing the non-canonical form at plan time costs one
// clear error instead, and names the range actually described, which is worth saying
// on its own account: `10.1.2.3/0` reads in a plan diff as a single host while
// matching the entire IPv4 space.
//
// Null and unknown values defer to the server, per STYLE_GUIDE §Config-time
// validators.
func IPv4CIDR() validator.String {
	return ipv4CIDRValidator{requireNetworkAddress: true}
}

// IPv4CIDRAllowingHostBits returns a validator.String enforcing everything IPv4CIDR
// does except the network-address rule, so `10.10.0.1/16` is accepted.
//
// The concession exists because the network-address rule is a wire fact about one
// endpoint, not a truth about CIDR notation. It was probed on the ZTNA app's
// `bareIps` and nowhere else. `jamfplatform_security_cloud_ztna_gateway` is already
// shipped and its IPsec endpoint has never been probed for canonicalisation, so
// imposing the rule there would refuse, at plan time, a configuration that may well
// apply cleanly today — a regression traded for a defect that may not exist. Probe
// that endpoint and this constructor should go away; until then the shipped
// behaviour stands.
//
// What it does NOT concede is the IPv4-mapped IPv6 bypass: `::ffff:10.0.0.0/104` is
// refused here too, because that is a defect in every caller regardless of endpoint.
func IPv4CIDRAllowingHostBits() validator.String {
	return ipv4CIDRValidator{}
}

// Description returns a plain-text description of the validator.
func (ipv4CIDRValidator) Description(_ context.Context) string {
	return "must be an IPv4 address range in CIDR notation, for example 10.1.2.0/24"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipv4CIDRValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v ipv4CIDRValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if reason := ipv4CIDRProblemWith(value, v.requireNetworkAddress); reason != "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv4 address range",
			reason+" Got: "+value,
		)
	}
}

// ParseIPv4CIDR parses value against the rules IPv4CIDR enforces, returning the
// network it names and that network's prefix length.
//
// Exported so one CIDR grammar can serve the callers that need the parsed network as
// well as the verdict: `internal/resources/security_cloud/ztna_gateway` has to check
// the Jamf-side subnet's prefix against the bounds of the private range it falls in,
// and the net.ParseCIDR it used to do that with was a second grammar free to drift
// from this one — as it had, accepting the IPv4-mapped IPv6 block this one rejects.
//
// The error carries the same sentence the validator puts in a diagnostic, so a caller
// wrapping it in a diagnostic of its own still says what was wrong.
//
// It applies the shape rules only, not the network-address rule — see
// IPv4CIDRAllowingHostBits for why that rule is scoped to the one endpoint it was
// probed on. A caller that wants it should compare the returned network against the
// input itself.
func ParseIPv4CIDR(value string) (*net.IPNet, int, error) {
	network, prefix, reason := ipv4CIDRShape(value)
	if reason != "" {
		return nil, 0, errors.New(reason)
	}
	return network, prefix, nil
}

// ipv4CIDRProblem returns a sentence describing what is wrong with value, or the
// empty string when it is an acceptable IPv4 CIDR block. Split out from
// ValidateString so the cases can be unit-tested without building a framework
// request for each one.
func ipv4CIDRProblem(value string) string {
	return ipv4CIDRProblemWith(value, true)
}

// ipv4CIDRProblemWith applies the shape rules and, when requireNetworkAddress is
// set, the network-address rule on top.
func ipv4CIDRProblemWith(value string, requireNetworkAddress bool) string {
	network, _, reason := ipv4CIDRShape(value)
	if reason != "" {
		return reason
	}
	if !requireNetworkAddress {
		return ""
	}
	if ip, _, err := net.ParseCIDR(value); err == nil && !ip.Equal(network.IP) {
		return "An address range must name its network address, not a host inside it. Jamf Security " +
			"Cloud stores `" + network.String() + "`, so writing it any other way would fail the apply. " +
			"Use `" + network.String() + "`, or a `/32` prefix for a single host."
	}
	return ""
}

// ipv4CIDRShape holds the rules that hold for every caller, returning the network
// value names and that network's prefix length, or a nil network and a sentence
// describing the first rule broken. The reason is empty exactly when the network is
// non-nil. The network-address rule sits in ipv4CIDRProblemWith instead, because it
// is scoped to one endpoint.
func ipv4CIDRShape(value string) (*net.IPNet, int, string) {
	if value == "" {
		return nil, 0, "An address range is required."
	}
	if strings.Contains(value, ":") {
		return nil, 0, "IPv6 address ranges are not accepted; supply an IPv4 range in CIDR notation."
	}
	if !strings.Contains(value, "/") {
		return nil, 0, "An address range must carry a prefix length, as in `10.1.2.0/24`; a bare address is not accepted."
	}
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		return nil, 0, "An address range must be in CIDR notation, as in `10.1.2.0/24`."
	}
	if ip.To4() == nil {
		return nil, 0, "IPv6 address ranges are not accepted; supply an IPv4 range in CIDR notation."
	}
	prefix, _ := network.Mask.Size()
	return network, prefix, ""
}
