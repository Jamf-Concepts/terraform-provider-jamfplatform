// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// privateRange is one of the three RFC 1918 blocks Jamf Security Cloud accepts
// for the Jamf-side encryption domain, with the prefix lengths it allows inside
// it.
type privateRange struct {
	cidr      string
	minPrefix int
	maxPrefix int
}

// jamfSidePrivateRanges is the accepted set for the Jamf-side subnet, quoted from
// the server's own rejection message (wire-probed 2026-08-27): "IPSec left subnet
// must be a private range: 10.0.0.0/8 (/8–/30), 172.16.0.0/12 (/12–/30) or
// 192.168.0.0/16 (/16–/30)."
var jamfSidePrivateRanges = []privateRange{
	{cidr: "10.0.0.0/8", minPrefix: 8, maxPrefix: 30},
	{cidr: "172.16.0.0/12", minPrefix: 12, maxPrefix: 30},
	{cidr: "192.168.0.0/16", minPrefix: 16, maxPrefix: 30},
}

// ipv4AddressValidator checks that a string attribute holds an IPv4 address in
// dotted-quad form.
type ipv4AddressValidator struct{}

// ipv4Address returns a validator.String enforcing dotted-quad IPv4 form.
func ipv4Address() validator.String {
	return ipv4AddressValidator{}
}

// Description returns a plain-text description of the validator.
func (ipv4AddressValidator) Description(_ context.Context) string {
	return "must be an IPv4 address in dotted-quad form"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipv4AddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v ipv4AddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if ip := net.ParseIP(value); ip != nil && ip.To4() != nil && strings.Count(value, ".") == 3 {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid IPv4 address",
		"Expected an IPv4 address in dotted-quad form. Got: "+value,
	)
}

// cidrBlockValidator checks that a string attribute holds an IPv4 CIDR block.
type cidrBlockValidator struct{}

// cidrBlock returns a validator.String enforcing IPv4 CIDR notation.
//
// Why plan time: an invalid subnet reaches the user as
// `400 [INVALID_FIELD] ipsec: IPSec configuration is not valid.` — the whole
// block named, nothing about which address or why (wire-probed 2026-08-27).
func cidrBlock() validator.String {
	return cidrBlockValidator{}
}

// Description returns a plain-text description of the validator.
func (cidrBlockValidator) Description(_ context.Context) string {
	return "must be an IPv4 CIDR block, for example 10.10.0.0/16"
}

// MarkdownDescription returns the markdown description of the validator.
func (v cidrBlockValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v cidrBlockValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if _, _, err := parseIPv4CIDR(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR block",
			"Expected an IPv4 CIDR block such as `10.10.0.0/16`. Got: "+value,
		)
	}
}

// privateCIDRValidator checks that a string attribute holds an IPv4 CIDR block
// inside one of the private ranges Jamf Security Cloud accepts for the Jamf-side
// encryption domain, with a prefix length that range allows.
type privateCIDRValidator struct{}

// privateCIDR returns a validator.String enforcing the Jamf-side subnet rules.
//
// The prefix bounds matter as much as the range: the server accepts
// `172.16.0.0/12` but not `172.16.0.0/31`, and rejects both an out-of-range
// address and an out-of-bounds prefix with the same message.
func privateCIDR() validator.String {
	return privateCIDRValidator{}
}

// Description returns a plain-text description of the validator.
func (privateCIDRValidator) Description(_ context.Context) string {
	return "must be a private IPv4 CIDR block: 10.0.0.0/8 (/8-/30), 172.16.0.0/12 (/12-/30) or 192.168.0.0/16 (/16-/30)"
}

// MarkdownDescription returns the markdown description of the validator.
func (v privateCIDRValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString implements validator.String.
func (v privateCIDRValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()

	ip, prefix, err := parseIPv4CIDR(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Jamf Security Cloud subnet",
			"Expected an IPv4 CIDR block such as `172.16.0.0/12`. Got: "+value,
		)
		return
	}

	for _, r := range jamfSidePrivateRanges {
		_, network, parseErr := net.ParseCIDR(r.cidr)
		if parseErr != nil || !network.Contains(ip) {
			continue
		}
		if prefix < r.minPrefix || prefix > r.maxPrefix {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Jamf Security Cloud subnet prefix out of range",
				fmt.Sprintf("%s allows a prefix between /%d and /%d. Got: /%d.", r.cidr, r.minPrefix, r.maxPrefix, prefix),
			)
		}
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Jamf Security Cloud subnet is not a private range",
		"Jamf Security Cloud requires this subnet to fall inside 10.0.0.0/8 (/8-/30), 172.16.0.0/12 (/12-/30) or "+
			"192.168.0.0/16 (/16-/30). Got: "+value,
	)
}

// parseIPv4CIDR parses an IPv4 CIDR block, returning the address and prefix
// length. It rejects IPv6, which the gateway endpoints do not accept.
func parseIPv4CIDR(value string) (net.IP, int, error) {
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		return nil, 0, err
	}
	if ip.To4() == nil {
		return nil, 0, fmt.Errorf("%q is not an IPv4 CIDR block", value)
	}
	prefix, _ := network.Mask.Size()
	return ip, prefix, nil
}

// ipsecAvailabilityZonesValidator enforces that availability_zones is only set on
// an IPsec gateway.
//
// The server refuses the combination with `400 [INVALID_FIELD]
// availabilityZones: availabilityZones must be empty when dedicatedIps.enabled is
// true.` Because the provider derives that flag from the absence of the `ipsec`
// block, the equivalent config-level rule is: availability zones require `ipsec`.
type ipsecAvailabilityZonesValidator struct{}

var _ resource.ConfigValidator = ipsecAvailabilityZonesValidator{}

// Description returns a plain-text description of the validator.
func (ipsecAvailabilityZonesValidator) Description(_ context.Context) string {
	return "availability_zones may only be set when the ipsec block is present"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipsecAvailabilityZonesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
func (v ipsecAvailabilityZonesValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config GatewayResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.IPSec != nil {
		return
	}
	if config.AvailabilityZones.IsNull() || config.AvailabilityZones.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("availability_zones"),
		"Availability zones require an IPsec gateway",
		"`availability_zones` names the source addresses of IPsec traffic, so it only applies to a dedicated "+
			"IPsec gateway. This gateway has no `ipsec` block, which makes it a dedicated internet gateway — "+
			"remove `availability_zones`, or add the `ipsec` block.",
	)
}
