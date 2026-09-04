// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package dns_hostname_mappings

import (
	"context"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ipv6AddressValidator checks that a string attribute holds an IPv6 address.
type ipv6AddressValidator struct{}

// IPv6Address returns a validator.String enforcing IPv6 form.
//
// Package-local rather than in internal/common/validators: this is its only
// consumer in the tree, and STYLE_GUIDE §Shared abstractions puts the trigger for a
// non-trivial code helper at two. Its IPv4 counterpart is shared because three
// resources need it. Move this one out when a second consumer appears — the natural
// candidate is whatever next models a AAAA record.
//
// Why it exists: the endpoint refuses an IPv4 address in ipv6_addresses, and an IPv6
// address in ipv4_addresses, with the same opaque 400 [INVALID_FIELD] "Invalid field
// value." carrying a null field — wire-probed 2026-08-29. Nothing in the response
// says which of the two lists was wrong, or which entry.
//
// net.ParseIP does the parsing; the discriminator is that To4() returns non-nil for
// a dotted quad and for an IPv4-mapped literal such as ::ffff:203.0.113.53, neither
// of which the endpoint takes here.
//
// Null and unknown values defer to the server, per STYLE_GUIDE §Config-time
// validators.
func IPv6Address() validator.String {
	return ipv6AddressValidator{}
}

// Description returns a plain-text description of the validator.
func (ipv6AddressValidator) Description(_ context.Context) string {
	return "must be an IPv6 address; an IPv4 address, including an IPv4-mapped IPv6 literal, is not accepted"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipv6AddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v ipv6AddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if strings.ContainsRune(value, ':') {
		if ip := net.ParseIP(value); ip != nil && ip.To4() == nil {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid IPv6 address",
		"Expected an IPv6 address. An IPv4 address, including an IPv4-mapped IPv6 literal such as "+
			"`::ffff:203.0.113.53`, is not accepted here — use `ipv4_addresses` for those. Got: "+value,
	)
}

// noTrailingRootDotValidator checks that a string attribute holds a host name that
// does not end in the root dot.
type noTrailingRootDotValidator struct{}

// NoTrailingRootDot returns a validator.String refusing a host name written with a
// trailing root dot.
//
// Package-local for the same reason as IPv6Address above: one consumer, and
// STYLE_GUIDE §Shared abstractions puts the extraction trigger at two. This one is
// unlikely ever to reach two, because the behaviour it guards is per construct.
//
// Why it exists: this endpoint strips a trailing root dot on write. `Trail.Example.COM.`
// sent, `Trail.Example.COM` stored — letter case untouched, the dot the only
// transform. Wire-probed against production EU on 2026-08-29. The sibling custom
// search domain endpoint keeps the dot (`trailing.example.com.` stored verbatim), so
// the shared commonvalidators.DNSHostname is right to accept the form and must keep
// accepting it: the divergence belongs here, not there.
//
// Refusing at plan time is the only remedy Terraform leaves. `mappings` is Required,
// so core compares the planned value — which is the configuration's, unchanged — with
// the state the apply produces, and the state comes from the read-back, which carries
// no dot. The result is a hard "Provider produced inconsistent result after apply" on
// every apply. Normalising the value on the way out does not help, because the plan
// still holds the dot. A plan modifier cannot normalise it either: core requires a
// non-Computed Required attribute's planned value to equal its configured value.
// Keeping the planned string in state rather than the read-back fails on refresh,
// where there is no plan to compare against.
//
// Only the trailing dot is judged here. Every other grammar rule stays with
// commonvalidators.DNSHostname, which runs alongside this validator on the same
// attribute.
//
// Null and unknown values defer to the server, per STYLE_GUIDE §Config-time
// validators.
func NoTrailingRootDot() validator.String {
	return noTrailingRootDotValidator{}
}

// Description returns a plain-text description of the validator.
func (noTrailingRootDotValidator) Description(_ context.Context) string {
	return "must be a host name written without a trailing dot"
}

// MarkdownDescription returns the markdown description of the validator.
func (v noTrailingRootDotValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v noTrailingRootDotValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if !strings.HasSuffix(value, ".") {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Host name ends in a trailing dot",
		"Jamf Security Cloud stores a hostname mapping without the trailing dot, so a configuration "+
			"carrying one can never match the value read back and every apply would fail. Write the host "+
			"name without the trailing dot. Got: "+value,
	)
}

// eachMappingHasAnAddressValidator checks that every mapping in the set resolves to
// at least one address.
type eachMappingHasAnAddressValidator struct{}

// EachMappingHasAnAddress returns a validator.Set enforcing that each mapping
// carries at least one IPv4 or IPv6 address.
//
// This is the cross-field rule from STYLE_GUIDE §Cross-field validation, and it is
// here rather than left to the server for two reasons. The endpoint refuses a
// mapping with neither list populated, and it always blames `aRecords` — even when
// ipv6_addresses is the list that was supplied and ipv4_addresses the one omitted.
// Wire-probed 2026-08-29: `aRecords` absent, `aRecords: []`, and both lists empty
// all answer 400 [INVALID_FIELD] on field `[0].aRecords`. So the server's own
// attribution sends an operator to the wrong attribute half the time, and the
// diagnostic here has to name both.
//
// Unknown values defer: an address list computed from another resource is not
// something this can rule on, and treating unknown as empty would reject a valid
// configuration.
func EachMappingHasAnAddress() validator.Set {
	return eachMappingHasAnAddressValidator{}
}

// Description returns a plain-text description of the validator.
func (eachMappingHasAnAddressValidator) Description(_ context.Context) string {
	return "each mapping must set at least one of ipv4_addresses or ipv6_addresses"
}

// MarkdownDescription returns the markdown description of the validator.
func (v eachMappingHasAnAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateSet implements validator.Set.
func (v eachMappingHasAnAddressValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var mappings []MappingModel
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &mappings, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, mapping := range mappings {
		if hasUnknownAddresses(mapping) || hasAnyAddress(mapping) {
			continue
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Hostname mapping resolves to no address",
			"The mapping for "+describeHostname(mapping)+" sets neither `ipv4_addresses` nor `ipv6_addresses`. "+
				"Every mapping must resolve to at least one address; set one of the two lists.",
		)
	}
}

// hasUnknownAddresses reports whether either address list is still unknown, in which
// case the mapping cannot be judged at plan time.
func hasUnknownAddresses(mapping MappingModel) bool {
	return mapping.IPv4Addresses.IsUnknown() || mapping.IPv6Addresses.IsUnknown()
}

// hasAnyAddress reports whether the mapping carries at least one address.
func hasAnyAddress(mapping MappingModel) bool {
	return addressCount(mapping.IPv4Addresses)+addressCount(mapping.IPv6Addresses) > 0
}

// addressCount returns the number of entries in an address list, treating null as
// empty. Absent and empty are the same thing on this endpoint.
func addressCount(addresses types.Set) int {
	if addresses.IsNull() || addresses.IsUnknown() {
		return 0
	}
	return len(addresses.Elements())
}

// describeHostname names the offending mapping in a diagnostic, falling back to a
// phrase rather than printing an empty string when the hostname itself is unknown.
func describeHostname(mapping MappingModel) string {
	if mapping.Hostname.IsNull() || mapping.Hostname.IsUnknown() {
		return "one of the mappings"
	}
	return "\"" + mapping.Hostname.ValueString() + "\""
}
