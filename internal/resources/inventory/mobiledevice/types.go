package mobiledevice

import (
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceMobileDevice defines the data source implementation.
type DataSourceMobileDevice struct {
	client *client.Client
}

// MobileDeviceDataSourceModel maps the data source schema data.
type MobileDeviceDataSourceModel struct {
	ID                          types.String `tfsdk:"id"`
	MobileDeviceId              types.String `tfsdk:"mobile_device_id"`
	DeviceType                  types.String `tfsdk:"device_type"`
	Sections                    types.List   `tfsdk:"sections"`
	Udid                        types.String `tfsdk:"udid"`
	DisplayName                 types.String `tfsdk:"display_name"`
	AssetTag                    types.String `tfsdk:"asset_tag"`
	SiteId                      types.String `tfsdk:"site_id"`
	LastInventoryUpdateDate     types.String `tfsdk:"last_inventory_update_date"`
	OsVersion                   types.String `tfsdk:"os_version"`
	OsBuild                     types.String `tfsdk:"os_build"`
	IpAddress                   types.String `tfsdk:"ip_address"`
	Managed                     types.Bool   `tfsdk:"managed"`
	Supervised                  types.Bool   `tfsdk:"supervised"`
	DeviceOwnershipType         types.String `tfsdk:"device_ownership_type"`
	LastEnrolledDate            types.String `tfsdk:"last_enrolled_date"`
	MdmProfileExpiration        types.String `tfsdk:"mdm_profile_expiration"`
	TimeZone                    types.String `tfsdk:"time_zone"`
	CapacityMb                  types.Int64  `tfsdk:"capacity_mb"`
	AvailableSpaceMb            types.Int64  `tfsdk:"available_space_mb"`
	UsedSpacePercentage         types.Int64  `tfsdk:"used_space_percentage"`
	BatteryLevel                types.Int64  `tfsdk:"battery_level"`
	BatteryHealth               types.String `tfsdk:"battery_health"`
	SerialNumber                types.String `tfsdk:"serial_number"`
	HardwareWifiMacAddress      types.String `tfsdk:"hardware_wifi_mac_address"`
	BluetoothMacAddress         types.String `tfsdk:"bluetooth_mac_address"`
	Model                       types.String `tfsdk:"model"`
	ModelIdentifier             types.String `tfsdk:"model_identifier"`
	ModelNumber                 types.String `tfsdk:"model_number"`
	DeviceId                    types.String `tfsdk:"device_id"`
	Username                    types.String `tfsdk:"username"`
	RealName                    types.String `tfsdk:"real_name"`
	EmailAddress                types.String `tfsdk:"email_address"`
	Position                    types.String `tfsdk:"position"`
	PhoneNumber                 types.String `tfsdk:"phone_number"`
	DepartmentId                types.String `tfsdk:"department_id"`
	BuildingId                  types.String `tfsdk:"building_id"`
	Room                        types.String `tfsdk:"room"`
	Building                    types.String `tfsdk:"building"`
	Department                  types.String `tfsdk:"department"`
	Purchased                   types.Bool   `tfsdk:"purchased"`
	Leased                      types.Bool   `tfsdk:"leased"`
	PoNumber                    types.String `tfsdk:"po_number"`
	Vendor                      types.String `tfsdk:"vendor"`
	AppleCareId                 types.String `tfsdk:"apple_care_id"`
	PurchasePrice               types.String `tfsdk:"purchase_price"`
	PurchasingAccount           types.String `tfsdk:"purchasing_account"`
	PoDate                      types.String `tfsdk:"po_date"`
	WarrantyExpiresDate         types.String `tfsdk:"warranty_expires_date"`
	LeaseExpiresDate            types.String `tfsdk:"lease_expires_date"`
	LifeExpectancy              types.Int64  `tfsdk:"life_expectancy"`
	PurchasingContact           types.String `tfsdk:"purchasing_contact"`
	DataProtected               types.Bool   `tfsdk:"data_protected"`
	BlockLevelEncryptionCapable types.Bool   `tfsdk:"block_level_encryption_capable"`
	FileLevelEncryptionCapable  types.Bool   `tfsdk:"file_level_encryption_capable"`
	PasscodePresent             types.Bool   `tfsdk:"passcode_present"`
	PasscodeCompliant           types.Bool   `tfsdk:"passcode_compliant"`
	ActivationLockEnabled       types.Bool   `tfsdk:"activation_lock_enabled"`
	JailBreakDetected           types.Bool   `tfsdk:"jail_break_detected"`
	LostModeEnabled             types.Bool   `tfsdk:"lost_mode_enabled"`
	LostModeMessage             types.String `tfsdk:"lost_mode_message"`
	LostModePhoneNumber         types.String `tfsdk:"lost_mode_phone_number"`
	CellularTechnology          types.String `tfsdk:"cellular_technology"`
	Iccid                       types.String `tfsdk:"iccid"`
	Carrier                     types.String `tfsdk:"carrier"`
	SimPhoneNumber              types.String `tfsdk:"sim_phone_number"`
	NetworkWifiMacAddress       types.String `tfsdk:"network_wifi_mac_address"`
	BluetoothMac                types.String `tfsdk:"bluetooth_mac"`
	EthernetMac                 types.String `tfsdk:"ethernet_mac"`
	Applications                types.List   `tfsdk:"applications"`
	Profiles                    types.List   `tfsdk:"profiles"`
	Certificates                types.List   `tfsdk:"certificates"`
}
