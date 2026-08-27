// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"
	"fmt"
	"net"

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

// ipsecSourceAddressesValidator enforces that ipsec_source_ip_addresses is only
// set on an IPsec gateway.
//
// The server refuses the combination with `400 [INVALID_FIELD]
// availabilityZones: availabilityZones must be empty when dedicatedIps.enabled is
// true.` Because the provider derives that flag from the absence of the `ipsec`
// block, the equivalent config-level rule is: source addresses require `ipsec`.
type ipsecSourceAddressesValidator struct{}

var _ resource.ConfigValidator = ipsecSourceAddressesValidator{}

// Description returns a plain-text description of the validator.
func (ipsecSourceAddressesValidator) Description(_ context.Context) string {
	return "ipsec_source_ip_addresses may only be set when the ipsec block is present"
}

// MarkdownDescription returns the markdown description of the validator.
func (v ipsecSourceAddressesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateResource implements resource.ConfigValidator.
func (v ipsecSourceAddressesValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config GatewayResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.IPSec != nil {
		return
	}
	if config.IPSecSourceIPAddresses.IsNull() || config.IPSecSourceIPAddresses.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("ipsec_source_ip_addresses"),
		"IPsec source addresses require an IPsec gateway",
		"`ipsec_source_ip_addresses` names the addresses IPsec traffic originates from, so it only applies to a "+
			"dedicated IPsec gateway. This gateway has no `ipsec` block, which makes it a dedicated internet "+
			"gateway — remove `ipsec_source_ip_addresses`, or add the `ipsec` block.",
	)
}
