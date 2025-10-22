// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/reference/get_v1-mobile-devices
// https://developer.jamf.com/platform-api/reference/get_v1-mobile-devices-id

package client

import (
	"context"
	"fmt"
	"net/url"
)

// Constants used for the Jamf Inventory API
const (
	inventoryMobileDevicesV1Prefix          = "/api/devices/v1/mobile-devices"
	MobileDeviceSectionGeneral              = "GENERAL"
	MobileDeviceSectionHardware             = "HARDWARE"
	MobileDeviceSectionUserAndLocation      = "USER_AND_LOCATION"
	MobileDeviceSectionPurchasing           = "PURCHASING"
	MobileDeviceSectionSecurity             = "SECURITY"
	MobileDeviceSectionApplications         = "APPLICATIONS"
	MobileDeviceSectionEbooks               = "EBOOKS"
	MobileDeviceSectionNetwork              = "NETWORK"
	MobileDeviceSectionServiceSubscriptions = "SERVICE_SUBSCRIPTIONS"
	MobileDeviceSectionCertificates         = "CERTIFICATES"
	MobileDeviceSectionProfiles             = "PROFILES"
	MobileDeviceSectionUserProfiles         = "USER_PROFILES"
	MobileDeviceSectionProvisioningProfiles = "PROVISIONING_PROFILES"
	MobileDeviceSectionSharedUsers          = "SHARED_USERS"
	MobileDeviceSectionExtensionAttributes  = "EXTENSION_ATTRIBUTES"
)

// ValidMobileDeviceSectionsV1 returns a list of valid section names for mobile device inventory requests
func ValidMobileDeviceSectionsV1() []string {
	return []string{
		MobileDeviceSectionGeneral,
		MobileDeviceSectionHardware,
		MobileDeviceSectionUserAndLocation,
		MobileDeviceSectionPurchasing,
		MobileDeviceSectionSecurity,
		MobileDeviceSectionApplications,
		MobileDeviceSectionEbooks,
		MobileDeviceSectionNetwork,
		MobileDeviceSectionServiceSubscriptions,
		MobileDeviceSectionCertificates,
		MobileDeviceSectionProfiles,
		MobileDeviceSectionUserProfiles,
		MobileDeviceSectionProvisioningProfiles,
		MobileDeviceSectionSharedUsers,
		MobileDeviceSectionExtensionAttributes,
	}
}

// InventoryMobileDeviceV1 represents a mobile device record from the Jamf Inventory API.
type InventoryMobileDeviceV1 struct {
	MobileDeviceId       string                                        `json:"mobileDeviceId"`
	DeviceType           string                                        `json:"deviceType"`
	General              InventoryMobileDeviceGeneralV1                `json:"general"`
	Purchasing           InventoryMobileDevicePurchasingV1             `json:"purchasing"`
	UserAndLocation      InventoryMobileDeviceUserAndLocationV1        `json:"userAndLocation"`
	Hardware             InventoryMobileDeviceHardwareV1               `json:"hardware"`
	Security             InventoryMobileDeviceSecurityV1               `json:"security"`
	Applications         []InventoryMobileDeviceApplicationV1          `json:"applications"`
	Profiles             []InventoryMobileDeviceProfileV1              `json:"profiles"`
	Certificates         []InventoryMobileDeviceCertificateV1          `json:"certificates"`
	ProvisioningProfiles []InventoryMobileDeviceProvisioningProfileV1  `json:"provisioningProfiles"`
	ServiceSubscriptions []InventoryMobileDeviceServiceSubscriptionsV1 `json:"serviceSubscriptions"`
	SharedUsers          []InventoryMobileDeviceSharedUserV1           `json:"sharedUsers"`
	ExtensionAttributes  []InventoryMobileDeviceExtensionAttributeV1   `json:"extensionAttributes"`
	Ebooks               []InventoryMobileDeviceEbookV1                `json:"ebooks"`
	Network              InventoryMobileDeviceNetworkV1                `json:"network"`
	UserProfiles         []InventoryMobileDeviceUserProfileV1          `json:"userProfiles"`
	SchoolDetails        InventoryJamfSchoolDeviceDetailsV1            `json:"schoolDetails"`
}

// InventoryMobileDeviceSearchResultsV1 represents a paginated list of inventory mobile devices.
type InventoryMobileDeviceSearchResultsV1 struct {
	TotalCount int                       `json:"totalCount"`
	Results    []InventoryMobileDeviceV1 `json:"results"`
}

// InventoryMobileDeviceGeneralV1 contains general information about the mobile device.
type InventoryMobileDeviceGeneralV1 struct {
	Udid                               string                                      `json:"udid"`
	DisplayName                        string                                      `json:"displayName"`
	AssetTag                           string                                      `json:"assetTag"`
	SiteId                             string                                      `json:"siteId"`
	LastInventoryUpdateDate            string                                      `json:"lastInventoryUpdateDate"`
	OsVersion                          string                                      `json:"osVersion"`
	OsRapidSecurityResponse            string                                      `json:"osRapidSecurityResponse"`
	OsBuild                            string                                      `json:"osBuild"`
	OsSupplementalBuildVersion         string                                      `json:"osSupplementalBuildVersion"`
	SoftwareUpdateDeviceId             string                                      `json:"softwareUpdateDeviceId"`
	IpAddress                          string                                      `json:"ipAddress"`
	Managed                            bool                                        `json:"managed"`
	Supervised                         bool                                        `json:"supervised"`
	DeviceOwnershipType                string                                      `json:"deviceOwnershipType"`
	EnrollmentMethodPrestage           InventoryEnrollmentMethodPrestageV1         `json:"enrollmentMethodPrestage"`
	EnrollmentSessionTokenValid        bool                                        `json:"enrollmentSessionTokenValid"`
	LastEnrolledDate                   string                                      `json:"lastEnrolledDate"`
	MdmProfileExpiration               string                                      `json:"mdmProfileExpiration"`
	TimeZone                           string                                      `json:"timeZone"`
	DeclarativeDeviceManagementEnabled bool                                        `json:"declarativeDeviceManagementEnabled"`
	ExtensionAttributes                []InventoryMobileDeviceExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryMobileDevicePurchasingV1 contains purchasing and warranty information for the mobile device.
type InventoryMobileDevicePurchasingV1 struct {
	Purchased           bool                                        `json:"purchased"`
	Leased              bool                                        `json:"leased"`
	PoNumber            string                                      `json:"poNumber"`
	Vendor              string                                      `json:"vendor"`
	AppleCareId         string                                      `json:"appleCareId"`
	PurchasePrice       string                                      `json:"purchasePrice"`
	PurchasingAccount   string                                      `json:"purchasingAccount"`
	PoDate              string                                      `json:"poDate"`
	WarrantyExpiresDate string                                      `json:"warrantyExpiresDate"`
	LeaseExpiresDate    string                                      `json:"leaseExpiresDate"`
	LifeExpectancy      int                                         `json:"lifeExpectancy"`
	PurchasingContact   string                                      `json:"purchasingContact"`
	ExtensionAttributes []InventoryMobileDeviceExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryMobileDeviceUserAndLocationV1 contains user and location information for the mobile device.
type InventoryMobileDeviceUserAndLocationV1 struct {
	Username            string                                      `json:"username"`
	RealName            string                                      `json:"realName"`
	EmailAddress        string                                      `json:"emailAddress"`
	Position            string                                      `json:"position"`
	PhoneNumber         string                                      `json:"phoneNumber"`
	DepartmentId        string                                      `json:"departmentId"`
	BuildingId          string                                      `json:"buildingId"`
	Room                string                                      `json:"room"`
	Building            string                                      `json:"building"`
	Department          string                                      `json:"department"`
	ExtensionAttributes []InventoryMobileDeviceExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryMobileDeviceHardwareV1 contains hardware details for the mobile device.
type InventoryMobileDeviceHardwareV1 struct {
	CapacityMb                int                                         `json:"capacityMb"`
	AvailableSpaceMb          int                                         `json:"availableSpaceMb"`
	UsedSpacePercentage       int                                         `json:"usedSpacePercentage"`
	BatteryLevel              int                                         `json:"batteryLevel"`
	BatteryHealth             string                                      `json:"batteryHealth"`
	SerialNumber              string                                      `json:"serialNumber"`
	WifiMacAddress            string                                      `json:"wifiMacAddress"`
	BluetoothMacAddress       string                                      `json:"bluetoothMacAddress"`
	ModemFirmwareVersion      string                                      `json:"modemFirmwareVersion"`
	Model                     string                                      `json:"model"`
	ModelIdentifier           string                                      `json:"modelIdentifier"`
	ModelNumber               string                                      `json:"modelNumber"`
	BluetoothLowEnergyCapable bool                                        `json:"bluetoothLowEnergyCapable"`
	DeviceId                  string                                      `json:"deviceId"`
	ExtensionAttributes       []InventoryMobileDeviceExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryMobileDeviceNetworkV1 contains network details for the mobile device.
type InventoryMobileDeviceNetworkV1 struct {
	CellularTechnology string `json:"cellularTechnology"`
	Iccid              string `json:"iccid"`
	Carrier            string `json:"carrier"`
	SimPhoneNumber     string `json:"simPhoneNumber"`
	WifiMacAddress     string `json:"wifiMacAddress"`
	BluetoothMac       string `json:"bluetoothMac"`
	EthernetMac        string `json:"ethernetMac"`
}

// InventoryMobileDeviceSecurityV1 contains security status and settings for the mobile device.
type InventoryMobileDeviceSecurityV1 struct {
	DataProtected                          bool                                    `json:"dataProtected"`
	BlockLevelEncryptionCapable            bool                                    `json:"blockLevelEncryptionCapable"`
	FileLevelEncryptionCapable             bool                                    `json:"fileLevelEncryptionCapable"`
	PasscodePresent                        bool                                    `json:"passcodePresent"`
	PasscodeCompliant                      bool                                    `json:"passcodeCompliant"`
	PasscodeCompliantWithProfile           bool                                    `json:"passcodeCompliantWithProfile"`
	HardwareEncryption                     int                                     `json:"hardwareEncryption"`
	ActivationLockEnabled                  bool                                    `json:"activationLockEnabled"`
	JailBreakDetected                      bool                                    `json:"jailBreakDetected"`
	AttestationStatus                      string                                  `json:"attestationStatus"`
	LastAttestationAttemptDate             string                                  `json:"lastAttestationAttemptDate"`
	LastSuccessfulAttestationDate          string                                  `json:"lastSuccessfulAttestationDate"`
	PasscodeLockGracePeriodEnforcedSeconds int                                     `json:"passcodeLockGracePeriodEnforcedSeconds"`
	PersonalDeviceProfileCurrent           bool                                    `json:"personalDeviceProfileCurrent"`
	LostModeEnabled                        bool                                    `json:"lostModeEnabled"`
	LostModeMessage                        string                                  `json:"lostModeMessage"`
	LostModePhoneNumber                    string                                  `json:"lostModePhoneNumber"`
	LostModeFootnote                       string                                  `json:"lostModeFootnote"`
	LostModeLocation                       InventoryMobileDeviceLostModeLocationV1 `json:"lostModeLocation"`
}

// InventoryMobileDeviceApplicationV1 represents an application installed on the mobile device.
type InventoryMobileDeviceApplicationV1 struct {
	Identifier       string `json:"identifier"`
	Name             string `json:"name"`
	Version          string `json:"version"`
	ShortVersion     string `json:"shortVersion"`
	ManagementStatus string `json:"managementStatus"`
	ValidationStatus bool   `json:"validationStatus"`
	BundleSize       string `json:"bundleSize"`
	DynamicSize      string `json:"dynamicSize"`
}

// InventoryMobileDeviceProfileV1 represents a configuration profile installed on the mobile device.
type InventoryMobileDeviceProfileV1 struct {
	DisplayName   string `json:"displayName"`
	Version       string `json:"version"`
	Uuid          string `json:"uuid"`
	Identifier    string `json:"identifier"`
	Removable     bool   `json:"removable"`
	LastInstalled string `json:"lastInstalled"`
	Username      string `json:"username"`
}

// InventoryMobileDeviceCertificateV1 represents a certificate installed on the mobile device.
type InventoryMobileDeviceCertificateV1 struct {
	CommonName     string `json:"commonName"`
	Identity       bool   `json:"identity"`
	ExpirationDate string `json:"expirationDate"`
}

// InventoryMobileDeviceProvisioningProfileV1 represents a provisioning profile on the mobile device.
type InventoryMobileDeviceProvisioningProfileV1 struct {
	DisplayName    string `json:"displayName"`
	Uuid           string `json:"uuid"`
	ExpirationDate string `json:"expirationDate"`
}

// InventoryMobileDeviceServiceSubscriptionsV1 contains service subscription details for the mobile device.
type InventoryMobileDeviceServiceSubscriptionsV1 struct {
	CarrierSettingsVersion   string `json:"carrierSettingsVersion"`
	CurrentCarrierNetwork    string `json:"currentCarrierNetwork"`
	CurrentMobileCountryCode string `json:"currentMobileCountryCode"`
	CurrentMobileNetworkCode string `json:"currentMobileNetworkCode"`
	SubscriberCarrierNetwork string `json:"subscriberCarrierNetwork"`
	Eid                      string `json:"eid"`
	Iccid                    string `json:"iccid"`
	Imei                     string `json:"imei"`
	DataPreferred            bool   `json:"dataPreferred"`
	Roaming                  bool   `json:"roaming"`
	VoicePreferred           bool   `json:"voicePreferred"`
	Label                    string `json:"label"`
}

// InventoryMobileDeviceSharedUserV1 represents a shared user on the mobile device.
type InventoryMobileDeviceSharedUserV1 struct {
	ManagedAppleId string `json:"managedAppleId"`
	LoggedIn       bool   `json:"loggedIn"`
	DataToSync     bool   `json:"dataToSync"`
}

// InventoryMobileDeviceLostModeLocationV1 contains lost mode location details for the mobile device.
type InventoryMobileDeviceLostModeLocationV1 struct {
	LastLocationUpdate                       string  `json:"lastLocationUpdate"`
	LostModeLocationHorizontalAccuracyMeters float64 `json:"lostModeLocationHorizontalAccuracyMeters"`
	LostModeLocationVerticalAccuracyMeters   float64 `json:"lostModeLocationVerticalAccuracyMeters"`
	LostModeLocationAltitudeMeters           float64 `json:"lostModeLocationAltitudeMeters"`
	LostModeLocationSpeedMetersPerSecond     float64 `json:"lostModeLocationSpeedMetersPerSecond"`
	LostModeLocationCourseDegrees            float64 `json:"lostModeLocationCourseDegrees"`
	LostModeLocationTimestamp                string  `json:"lostModeLocationTimestamp"`
}

// InventoryMobileDeviceExtensionAttributeV1 represents an extension attribute for the mobile device.
type InventoryMobileDeviceExtensionAttributeV1 struct {
	Id                                  string   `json:"id"`
	Name                                string   `json:"name"`
	Type                                string   `json:"type"`
	Value                               []string `json:"value"`
	ExtensionAttributeCollectionAllowed bool     `json:"extensionAttributeCollectionAllowed"`
	InventoryDisplay                    string   `json:"inventoryDisplay"`
}

// InventoryMobileDeviceEbookV1 represents an ebook assigned to the mobile device.
type InventoryMobileDeviceEbookV1 struct {
	Author          string `json:"author"`
	Title           string `json:"title"`
	Version         string `json:"version"`
	Kind            string `json:"kind"`
	ManagementState string `json:"managementState"`
}

// InventoryMobileDeviceUserProfileV1 represents a user profile on the mobile device.
type InventoryMobileDeviceUserProfileV1 struct {
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	Uuid        string `json:"uuid"`
	Identifier  string `json:"identifier"`
	Removable   bool   `json:"removable"`
}

// InventoryEnrollmentMethodPrestageV1 represents enrollment method prestage information.
type InventoryEnrollmentMethodPrestageV1 struct {
	MobileDevicePrestageId string `json:"mobileDevicePrestageId"`
	ProfileName            string `json:"profileName"`
}

// InventoryJamfSchoolDeviceDetailsV1 contains Jamf School specific device details.
type InventoryJamfSchoolDeviceDetailsV1 struct {
	Udid string `json:"udid"`
}

// GetInventoryMobileDeviceByIDV1 fetches a single mobile device by ID from the Jamf Inventory API.
// sections parameter allows specifying which sections of data to retrieve (e.g., []string{"GENERAL", "HARDWARE"})
func (c *Client) GetInventoryMobileDeviceByIDV1(ctx context.Context, id string, sections []string) (*InventoryMobileDeviceV1, error) {
	params := url.Values{}
	for _, section := range sections {
		params.Add("section", section)
	}

	endpoint := fmt.Sprintf("%s/%s", inventoryMobileDevicesV1Prefix, url.PathEscape(id))
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get mobile device by id: %w", err)
	}
	var result InventoryMobileDeviceV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryMobileDevicesV1 fetches a paginated list of mobile devices with optional sections and pagination.
// sections parameter allows specifying which sections of data to retrieve (e.g., []string{"GENERAL", "HARDWARE"})
func (c *Client) GetInventoryMobileDevicesV1(ctx context.Context, page, pageSize int, sections []string) (*InventoryMobileDeviceSearchResultsV1, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		params.Set("page-size", fmt.Sprintf("%d", pageSize))
	}
	for _, section := range sections {
		params.Add("section", section)
	}

	endpoint := inventoryMobileDevicesV1Prefix
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list mobile devices: %w", err)
	}
	var result InventoryMobileDeviceSearchResultsV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryAllMobileDevicesV1 fetches all mobile devices by automatically handling pagination.
// It starts with page 0 and continues fetching until all devices are retrieved.
// sections parameter allows specifying which sections of data to retrieve (e.g., []string{"GENERAL", "HARDWARE"})
func (c *Client) GetInventoryAllMobileDevicesV1(ctx context.Context, sections []string) ([]InventoryMobileDeviceV1, error) {
	var allDevices []InventoryMobileDeviceV1
	page := 0
	pageSize := 100

	for {
		result, err := c.GetInventoryMobileDevicesV1(ctx, page, pageSize, sections)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch mobile devices page %d: %w", page, err)
		}

		allDevices = append(allDevices, result.Results...)

		if len(allDevices) >= result.TotalCount || len(result.Results) < pageSize {
			break
		}

		page++
	}

	return allDevices, nil
}
