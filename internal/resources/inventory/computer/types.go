package computer

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceComputer defines the data source implementation.
type DataSourceComputer struct {
	client *client.Client
}

// ComputerDataSourceModel maps the data source schema data.
type ComputerDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	UDID                  types.String `tfsdk:"udid"`
	Name                  types.String `tfsdk:"name"`
	LastIpAddress         types.String `tfsdk:"last_ip_address"`
	LastContactTime       types.String `tfsdk:"last_contact_time"`
	LastEnrolledDate      types.String `tfsdk:"last_enrolled_date"`
	Platform              types.String `tfsdk:"platform"`
	Supervised            types.Bool   `tfsdk:"supervised"`
	AssetTag              types.String `tfsdk:"asset_tag"`
	JamfBinaryVersion     types.String `tfsdk:"jamf_binary_version"`
	ManagementId          types.String `tfsdk:"management_id"`
	Make                  types.String `tfsdk:"make"`
	Model                 types.String `tfsdk:"model"`
	ModelIdentifier       types.String `tfsdk:"model_identifier"`
	SerialNumber          types.String `tfsdk:"serial_number"`
	ProcessorType         types.String `tfsdk:"processor_type"`
	ProcessorSpeedMhz     types.Int64  `tfsdk:"processor_speed_mhz"`
	TotalRamMegabytes     types.Int64  `tfsdk:"total_ram_megabytes"`
	MacAddress            types.String `tfsdk:"mac_address"`
	OsName                types.String `tfsdk:"os_name"`
	OsVersion             types.String `tfsdk:"os_version"`
	OsBuild               types.String `tfsdk:"os_build"`
	Username              types.String `tfsdk:"username"`
	Realname              types.String `tfsdk:"realname"`
	Email                 types.String `tfsdk:"email"`
	Position              types.String `tfsdk:"position"`
	Phone                 types.String `tfsdk:"phone"`
	DepartmentId          types.String `tfsdk:"department_id"`
	BuildingId            types.String `tfsdk:"building_id"`
	Room                  types.String `tfsdk:"room"`
	Purchased             types.Bool   `tfsdk:"purchased"`
	Leased                types.Bool   `tfsdk:"leased"`
	PoNumber              types.String `tfsdk:"po_number"`
	Vendor                types.String `tfsdk:"vendor"`
	WarrantyDate          types.String `tfsdk:"warranty_date"`
	PurchasePrice         types.String `tfsdk:"purchase_price"`
	SipStatus             types.String `tfsdk:"sip_status"`
	GatekeeperStatus      types.String `tfsdk:"gatekeeper_status"`
	ActivationLockEnabled types.Bool   `tfsdk:"activation_lock_enabled"`
	RecoveryLockEnabled   types.Bool   `tfsdk:"recovery_lock_enabled"`
	Applications          types.List   `tfsdk:"applications"`
	ConfigurationProfiles types.List   `tfsdk:"configuration_profiles"`
	LocalUserAccounts     types.List   `tfsdk:"local_user_accounts"`
}
