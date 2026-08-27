// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// Package ztna_gateway implements the jamfplatform_security_cloud_ztna_gateway
// resource, data sources and list resource backed by the Jamf Security Cloud
// ZTNA gateway API.
//
// A dedicated gateway is the tenant's own egress point into Jamf Security Cloud.
// It comes in exactly two forms, which the admin UI presents as two separate
// sections and the API distinguishes by which of two mutually exclusive fields is
// set:
//
//   - a **dedicated IPsec gateway**, which builds a tunnel to the customer's own
//     VPN concentrator, and
//   - a **dedicated internet gateway**, which routes to the internet through a
//     pair of private egress IPs Jamf provisions.
//
// The provider models the choice as the presence or absence of the `ipsec` block
// rather than as a separate discriminator attribute. That is not a shortcut: the
// API requires exactly one of `ipsec` or the dedicated-egress-IP flag
// ("Gateway must be configured as dedicated (dedicatedIps.enabled=true) or
// ipsec."), refuses both together, and there is no configuration in which the
// flag carries information the `ipsec` block does not already imply. Deriving it
// removes a whole class of config that could only ever be rejected.
//
// Attribute names follow the admin UI rather than the wire wherever the two
// diverge, per STYLE_GUIDE §Attribute names mirror the Jamf Pro admin UI. Because
// the guide also forbids comments inside function bodies, the wire mapping lives
// here rather than beside each attribute — this table is where to look when
// searching for a field name seen in an API response:
//
//	Terraform attribute                    Wire field
//	------------------------------------   --------------------------------
//	egress_region                          datacenter
//	ipsec_source_ip_addresses              availabilityZones
//	dedicated_egress_ip_addresses          dedicatedIps.ips
//	ipsec.key_exchange_protocol            ipsec.keyExchange
//	ipsec.phase_1                          ipsec.ike
//	ipsec.phase_2                          ipsec.esp
//	ipsec.phase_N.diffie_hellman_group     ipsec.{ike,esp}.dhGroups
//	ipsec.phase_N.sa_lifetime_seconds      ipsec.{ike,esp}.lifetimeInSec
//	ipsec.jamf_side                        ipsec.left
//	ipsec.customer_side                    ipsec.right
//	ipsec.*.ike_domain_id                  ipsec.{left,right}.id
//	ipsec.jamf_side.subnet                 ipsec.left.subnets (single element)
//	ipsec.jamf_side.authentication_secret  ipsec.left.secret
//
// Two of those are worth a word. `left`/`right` is strongSwan jargon that appears
// nowhere a Jamf administrator would see — the Encryption Domain step labels the
// two ends "Jamf Security Cloud side" and "Customer side". And `availabilityZones`
// is not merely different from its label but actively misleading: the SDK itself
// notes that despite the name, the values are IPv4 addresses rather than zone
// identifiers.
//
// Enumerated attributes take the admin UI's labels too, translated to stored
// values at the boundary; see mappings.go for the tables and their provenance.
package ztna_gateway

import (
	"context"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	commonvalidators "github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/validators"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/providerdata"
)

// GatewayResource implements the Terraform resource for Jamf Security Cloud ZTNA
// gateways.
type GatewayResource struct {
	client *securitycloud.Client
}

var (
	_ resource.Resource                     = &GatewayResource{}
	_ resource.ResourceWithImportState      = &GatewayResource{}
	_ resource.ResourceWithIdentity         = &GatewayResource{}
	_ resource.ResourceWithConfigValidators = &GatewayResource{}
)

const (
	defaultCreateTimeout = 120 * time.Second
	defaultReadTimeout   = 60 * time.Second
	defaultUpdateTimeout = 120 * time.Second
	defaultDeleteTimeout = 120 * time.Second
)

// NewGatewayResource returns a new instance of GatewayResource.
func NewGatewayResource() resource.Resource {
	return &GatewayResource{}
}

// Metadata sets the resource type name for the Terraform provider.
func (r *GatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_cloud_ztna_gateway"
}

// IdentitySchema defines the identifier used for import and list results.
func (r *GatewayResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "Gateway ID used to uniquely reference the gateway.",
				RequiredForImport: true,
			},
		},
	}
}

// ConfigValidators enforces the cross-field rules the API applies to the IPsec
// block, at plan time rather than mid-apply.
func (r *GatewayResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		ipsecSourceAddressesValidator{},
	}
}

// Schema returns the Terraform schema for the ZTNA gateway resource.
func (r *GatewayResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a dedicated Jamf Security Cloud ZTNA gateway — the tenant's own egress point " +
			"into Jamf Security Cloud, and what a custom DNS zone's name servers and a ZTNA app's routing are " +
			"reachable through.\n\n" +
			"A gateway takes one of two forms, chosen by whether the `ipsec` block is present:\n\n" +
			"- **Dedicated IPsec gateway** — set `ipsec` to build a tunnel to your own VPN concentrator.\n" +
			"- **Dedicated internet gateway** — omit `ipsec` to route to the internet through a pair of private " +
			"egress IP addresses Jamf provisions, reported in `dedicated_egress_ip_addresses`.\n\n" +
			"The form is fixed for the life of the gateway: Jamf Security Cloud refuses to convert one into the " +
			"other, so adding or removing `ipsec` replaces the gateway. Deleting a gateway that a custom DNS zone " +
			"or a grouped gateway still references is also refused — drop the reference in a separate apply first." +
			resourcePrivileges,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Gateway ID assigned by Jamf Security Cloud.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "**\"Gateway name\"** in the Jamf Security Cloud admin UI.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"egress_region": schema.StringAttribute{
				MarkdownDescription: "**\"Egress region\"** in the Jamf Security Cloud admin UI — the region this " +
					"gateway is deployed to. Changing it re-provisions the gateway in the new region: connectivity " +
					"drops, the reported status returns to `PENDING`, and any dedicated egress IP addresses stay " +
					"stale until provisioning completes. Valid values: " + markdownList(egressRegionValues()) + ".",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(egressRegionValues()...),
				},
			},
			"contact": schema.SingleNestedAttribute{
				MarkdownDescription: "**\"Contact name\"** and **\"Contact email\"** in the Jamf Security Cloud " +
					"admin UI — who Jamf should reach about this gateway's operation.",
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Contact name, or a team name.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"email": schema.StringAttribute{
						MarkdownDescription: "Contact email address.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the deployment is active. A disabled gateway reports its status as " +
					"`DISABLED` and carries no traffic. Defaults to `true`.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"tenant_ids": schema.SetAttribute{
				MarkdownDescription: "IDs of the tenants granted access to this gateway. At least one, and every " +
					"one must belong to the same organization as the credentials the provider is configured with — " +
					"a tenant outside it is refused.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"ipsec_source_ip_addresses": schema.SetAttribute{
				MarkdownDescription: "**\"Jamf Security Cloud IPsec source IP addresses\"** in the Jamf Security " +
					"Cloud admin UI — the addresses IPsec traffic from Jamf Security Cloud originates from, which " +
					"your firewall must allow. Supply both addresses your egress region offers for dynamic " +
					"addressing, or one to pin a single source address. Only valid on an IPsec gateway: a dedicated " +
					"internet gateway must leave this unset.\n\n" +
					"The accepted addresses are fixed per egress region and are the ones the admin UI lists when " +
					"you pick the region. The provider does not check them at plan time because the accepted set " +
					"is not published anywhere it can read.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(commonvalidators.IPv4Address()),
				},
			},
			"dedicated_egress_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "The private egress IP addresses Jamf provisions for a dedicated internet " +
					"gateway. Empty until provisioning completes, and always empty on an IPsec gateway. Read-only.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipsec":  ipsecAttribute(),
			"status": statusAttribute(),
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// ipsecAttribute builds the IPsec tunnel block. Its presence selects the gateway
// form, and Jamf Security Cloud refuses to change the form of an existing gateway
// (`GATEWAY_TYPE_CHANGE_NOT_SUPPORTED`), so adding or removing the block has to
// replace the resource rather than attempt an update that cannot succeed.
func ipsecAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "IPsec tunnel configuration. Present on a dedicated IPsec gateway, absent on a " +
			"dedicated internet gateway. Adding or removing the whole block replaces the gateway.",
		Optional: true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.RequiresReplaceIf(
				requiresReplaceOnIpsecPresenceChange,
				"Adding or removing the `ipsec` block changes the gateway's form, which Jamf Security Cloud does not support in place.",
				"Adding or removing the `ipsec` block changes the gateway's form, which Jamf Security Cloud does not support in place.",
			),
		},
		Attributes: map[string]schema.Attribute{
			"key_exchange_protocol": schema.StringAttribute{
				MarkdownDescription: "**\"Key exchange protocol\"** in the Jamf Security Cloud admin UI. Valid " +
					"values: " + markdownList(keyExchangeValues()) + ".",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(keyExchangeValues()...),
				},
			},
			"phase_1": cipherSuiteAttribute(
				"**\"Phase 1\"** in the Jamf Security Cloud admin UI — the cipher suite protecting the key exchange itself.",
			),
			"phase_2": cipherSuiteAttribute(
				"**\"Phase 2\"** in the Jamf Security Cloud admin UI — the cipher suite protecting the tunnelled traffic.",
			),
			"jamf_side":     jamfSideAttribute(),
			"customer_side": customerSideAttribute(),
		},
	}
}

// cipherSuiteAttribute builds one cipher-suite block. Each of the three
// algorithm fields is a single value, not a list: the wire shape is an array but
// the server rejects anything other than exactly one element, so a list would
// offer users only invalid extra room.
func cipherSuiteAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"encryption": schema.StringAttribute{
				MarkdownDescription: "**\"Encryption\"** in the Jamf Security Cloud admin UI. Valid values: " +
					markdownList(encryptionValues()) + ".",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(encryptionValues()...),
				},
			},
			"integrity": schema.StringAttribute{
				MarkdownDescription: "**\"Integrity\"** in the Jamf Security Cloud admin UI. Valid values: " +
					markdownList(integrityValues()) + ".",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(integrityValues()...),
				},
			},
			"diffie_hellman_group": schema.StringAttribute{
				MarkdownDescription: "**\"Diffie-Hellman Group\"** in the Jamf Security Cloud admin UI. Valid " +
					"values: " + markdownList(diffieHellmanGroupValues()) + ".",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(diffieHellmanGroupValues()...),
				},
			},
			"sa_lifetime_seconds": schema.Int64Attribute{
				MarkdownDescription: "**\"Security Association (SA) Lifetime\"** in the Jamf Security Cloud admin " +
					"UI, in seconds.",
				Required: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
		},
	}
}

// jamfSideAttribute builds the Jamf-side tunnel endpoint — the wire's `left`.
func jamfSideAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "**\"Jamf Security Cloud side\"** in the Jamf Security Cloud admin UI — the endpoint " +
			"Jamf presents to your VPN concentrator.",
		Required: true,
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "Endpoint address, or `%any` to accept any address.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"ike_domain_id": schema.StringAttribute{
				MarkdownDescription: "**\"Jamf Security Cloud IKE domain ID\"** in the Jamf Security Cloud admin " +
					"UI — the IKE identity Jamf presents, for example `wpa.wandera.com`.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "**\"Jamf Security Cloud subnet\"** in the Jamf Security Cloud admin UI — the " +
					"range all end-user traffic originates from through the tunnel, in CIDR notation. Must be a " +
					"private range: `10.0.0.0/8` with a `/8`–`/30` prefix, `172.16.0.0/12` with `/12`–`/30`, or " +
					"`192.168.0.0/16` with `/16`–`/30`. The range must not exist anywhere else on your network.",
				Required: true,
				Validators: []validator.String{
					privateCIDR(),
				},
			},
			"authentication_secret": schema.StringAttribute{
				MarkdownDescription: "**\"Authentication secret\"** in the Jamf Security Cloud admin UI — the IPsec " +
					"pre-shared key, applied to both ends of the tunnel. `WriteOnly` — sent to Jamf Security Cloud " +
					"on writes but **never persisted in Terraform state**, because Jamf never returns it. Pair with " +
					"`authentication_secret_wo_version` to rotate it. It can be rotated but not cleared.",
				Required:  true,
				Sensitive: true,
				WriteOnly: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"authentication_secret_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Rotation trigger for the `WriteOnly` `authentication_secret`. Bump this integer to " +
					"force an update that re-sends the secret. Set it to `1` on create. Leaving it unset or " +
					"unchanged means \"leave the stored key alone\" — the provider omits the secret from the next " +
					"update so Jamf Security Cloud retains the existing one.",
				Optional: true,
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "Authentication method Jamf Security Cloud reports for this endpoint. " +
					"Read-only.",
				Computed: true,
			},
		},
	}
}

// customerSideAttribute builds the remote-peer endpoint — the wire's `right`.
func customerSideAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "**\"Customer side\"** in the Jamf Security Cloud admin UI — your own VPN " +
			"concentrator, and the subnets reachable through it.",
		Required: true,
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "**\"Your IPsec gateway IP address\"** in the Jamf Security Cloud admin UI.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"ike_domain_id": schema.StringAttribute{
				MarkdownDescription: "**\"Your IKE domain ID\"** in the Jamf Security Cloud admin UI — the IKE " +
					"identity your concentrator presents.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"subnets": schema.SetAttribute{
				MarkdownDescription: "**\"Customer subnets\"** in the Jamf Security Cloud admin UI — the subnets " +
					"reachable through this gateway, in CIDR notation, usually where your applications live. At " +
					"least one. `0.0.0.0/0` is accepted, and narrowing access on the firewall instead is the " +
					"documented approach for most firewalls.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(cidrBlock()),
				},
			},
			"vendor": schema.StringAttribute{
				MarkdownDescription: "**\"IPsec network vendor\"** in the Jamf Security Cloud admin UI — the VPN " +
					"vendor of your concentrator. Case-sensitive. Valid values: " + markdownList(vendorValues()) + ".",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(vendorValues()...),
				},
			},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "Authentication method Jamf Security Cloud reports for this endpoint. " +
					"Read-only.",
				Computed: true,
			},
		},
	}
}

// statusAttribute builds the read-only status block.
//
// The wire also carries a `updatedAt` timestamp, which this deliberately does not
// expose. It advances every time Jamf Security Cloud re-evaluates the gateway,
// so surfacing it would make every single refresh report the object as changed
// outside Terraform — noise about a value no configuration can act on. `state`
// and `tunnel_state` do settle once provisioning finishes, and both are worth
// reading, so they stay.
func statusAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Operational status Jamf Security Cloud reports for this gateway. Read-only, and " +
			"live: expect `PENDING` immediately after a create or an egress-region change, settling to `UP` once " +
			"the infrastructure is provisioned.",
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"state": schema.StringAttribute{
				MarkdownDescription: "Overall gateway state: `PENDING` while provisioning, `UP` when operational, " +
					"`DOWN` when unreachable or degraded, `DISABLED` when `enabled` is `false`.",
				Computed: true,
			},
			"tunnel_state": schema.StringAttribute{
				MarkdownDescription: "IPsec tunnel health, `UP` or `DOWN`. Null on a dedicated internet gateway, " +
					"and on an IPsec gateway until the first tunnel report arrives.",
				Computed: true,
			},
		},
	}
}

// Configure wires the Jamf Security Cloud client into the resource via the shared
// providerdata.ConfigureSecurityCloud helper.
func (r *GatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, diags := providerdata.ConfigureSecurityCloud(ctx, req.ProviderData, "jamfplatform_security_cloud_ztna_gateway")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

// ImportState handles import by the Jamf Security Cloud gateway ID.
//
// The IPsec pre-shared key cannot come back on import — Jamf Security Cloud never
// returns it — so an imported IPsec gateway needs `ipsec.jamf_side.authentication_secret`
// supplied in configuration before the next apply will succeed.
func (r *GatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
