// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/reference/get_v1-mobile-devices
// https://developer.jamf.com/platform-api/reference/get_v1-mobile-devices-id

package client

import (
	"context"
	"fmt"
	"net/url"
)

// Mobile Device Section constants for the Jamf Inventory API
const (
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

// ValidMobileDeviceSections returns a list of valid section names for mobile device inventory requests
func ValidMobileDeviceSections() []string {
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

// InventoryMobileDevice represents a mobile device record from the Jamf Inventory API.
type InventoryMobileDevice struct {
	MobileDeviceId       string                                      `json:"mobileDeviceId"`
	DeviceType           string                                      `json:"deviceType"`
	General              InventoryMobileDeviceGeneral                `json:"general"`
	Purchasing           InventoryMobileDevicePurchasing             `json:"purchasing"`
	UserAndLocation      InventoryMobileDeviceUserAndLocation        `json:"userAndLocation"`
	Hardware             InventoryMobileDeviceHardware               `json:"hardware"`
	Security             InventoryMobileDeviceSecurity               `json:"security"`
	Applications         []InventoryMobileDeviceApplication          `json:"applications"`
	Profiles             []InventoryMobileDeviceProfile              `json:"profiles"`
	Certificates         []InventoryMobileDeviceCertificate          `json:"certificates"`
	ProvisioningProfiles []InventoryMobileDeviceProvisioningProfile  `json:"provisioningProfiles"`
	ServiceSubscriptions []InventoryMobileDeviceServiceSubscriptions `json:"serviceSubscriptions"`
	SharedUsers          []InventoryMobileDeviceSharedUser           `json:"sharedUsers"`
	ExtensionAttributes  []InventoryMobileDeviceExtensionAttribute   `json:"extensionAttributes"`
	Ebooks               []InventoryMobileDeviceEbook                `json:"ebooks"`
	Network              InventoryMobileDeviceNetwork                `json:"network"`
	UserProfiles         []InventoryMobileDeviceUserProfile          `json:"userProfiles"`
	SchoolDetails        InventoryJamfSchoolDeviceDetails            `json:"schoolDetails"`
}

// InventoryMobileDeviceSearchResults represents a paginated list of inventory mobile devices.
type InventoryMobileDeviceSearchResults struct {
	TotalCount int                     `json:"totalCount"`
	Results    []InventoryMobileDevice `json:"results"`
}

// InventoryMobileDeviceGeneral contains general information about the mobile device.
type InventoryMobileDeviceGeneral struct {
	Udid                               string                                    `json:"udid"`
	DisplayName                        string                                    `json:"displayName"`
	AssetTag                           string                                    `json:"assetTag"`
	SiteId                             string                                    `json:"siteId"`
	LastInventoryUpdateDate            string                                    `json:"lastInventoryUpdateDate"`
	OsVersion                          string                                    `json:"osVersion"`
	OsRapidSecurityResponse            string                                    `json:"osRapidSecurityResponse"`
	OsBuild                            string                                    `json:"osBuild"`
	OsSupplementalBuildVersion         string                                    `json:"osSupplementalBuildVersion"`
	SoftwareUpdateDeviceId             string                                    `json:"softwareUpdateDeviceId"`
	IpAddress                          string                                    `json:"ipAddress"`
	Managed                            bool                                      `json:"managed"`
	Supervised                         bool                                      `json:"supervised"`
	DeviceOwnershipType                string                                    `json:"deviceOwnershipType"`
	EnrollmentMethodPrestage           InventoryEnrollmentMethodPrestage         `json:"enrollmentMethodPrestage"`
	EnrollmentSessionTokenValid        bool                                      `json:"enrollmentSessionTokenValid"`
	LastEnrolledDate                   string                                    `json:"lastEnrolledDate"`
	MdmProfileExpiration               string                                    `json:"mdmProfileExpiration"`
	TimeZone                           string                                    `json:"timeZone"`
	DeclarativeDeviceManagementEnabled bool                                      `json:"declarativeDeviceManagementEnabled"`
	ExtensionAttributes                []InventoryMobileDeviceExtensionAttribute `json:"extensionAttributes"`
}

// InventoryMobileDevicePurchasing contains purchasing and warranty information for the mobile device.
type InventoryMobileDevicePurchasing struct {
	Purchased           bool                                      `json:"purchased"`
	Leased              bool                                      `json:"leased"`
	PoNumber            string                                    `json:"poNumber"`
	Vendor              string                                    `json:"vendor"`
	AppleCareId         string                                    `json:"appleCareId"`
	PurchasePrice       string                                    `json:"purchasePrice"`
	PurchasingAccount   string                                    `json:"purchasingAccount"`
	PoDate              string                                    `json:"poDate"`
	WarrantyExpiresDate string                                    `json:"warrantyExpiresDate"`
	LeaseExpiresDate    string                                    `json:"leaseExpiresDate"`
	LifeExpectancy      int                                       `json:"lifeExpectancy"`
	PurchasingContact   string                                    `json:"purchasingContact"`
	ExtensionAttributes []InventoryMobileDeviceExtensionAttribute `json:"extensionAttributes"`
}

// InventoryMobileDeviceUserAndLocation contains user and location information for the mobile device.
type InventoryMobileDeviceUserAndLocation struct {
	Username            string                                    `json:"username"`
	RealName            string                                    `json:"realName"`
	EmailAddress        string                                    `json:"emailAddress"`
	Position            string                                    `json:"position"`
	PhoneNumber         string                                    `json:"phoneNumber"`
	DepartmentId        string                                    `json:"departmentId"`
	BuildingId          string                                    `json:"buildingId"`
	Room                string                                    `json:"room"`
	Building            string                                    `json:"building"`
	Department          string                                    `json:"department"`
	ExtensionAttributes []InventoryMobileDeviceExtensionAttribute `json:"extensionAttributes"`
}

// InventoryMobileDeviceHardware contains hardware details for the mobile device.
type InventoryMobileDeviceHardware struct {
	CapacityMb                int                                       `json:"capacityMb"`
	AvailableSpaceMb          int                                       `json:"availableSpaceMb"`
	UsedSpacePercentage       int                                       `json:"usedSpacePercentage"`
	BatteryLevel              int                                       `json:"batteryLevel"`
	BatteryHealth             string                                    `json:"batteryHealth"`
	SerialNumber              string                                    `json:"serialNumber"`
	WifiMacAddress            string                                    `json:"wifiMacAddress"`
	BluetoothMacAddress       string                                    `json:"bluetoothMacAddress"`
	ModemFirmwareVersion      string                                    `json:"modemFirmwareVersion"`
	Model                     string                                    `json:"model"`
	ModelIdentifier           string                                    `json:"modelIdentifier"`
	ModelNumber               string                                    `json:"modelNumber"`
	BluetoothLowEnergyCapable bool                                      `json:"bluetoothLowEnergyCapable"`
	DeviceId                  string                                    `json:"deviceId"`
	ExtensionAttributes       []InventoryMobileDeviceExtensionAttribute `json:"extensionAttributes"`
}

// InventoryMobileDeviceNetwork contains network details for the mobile device.
type InventoryMobileDeviceNetwork struct {
	CellularTechnology string `json:"cellularTechnology"`
	Iccid              string `json:"iccid"`
	Carrier            string `json:"carrier"`
	SimPhoneNumber     string `json:"simPhoneNumber"`
	WifiMacAddress     string `json:"wifiMacAddress"`
	BluetoothMac       string `json:"bluetoothMac"`
	EthernetMac        string `json:"ethernetMac"`
}

// InventoryMobileDeviceSecurity contains security status and settings for the mobile device.
type InventoryMobileDeviceSecurity struct {
	DataProtected                          bool                                  `json:"dataProtected"`
	BlockLevelEncryptionCapable            bool                                  `json:"blockLevelEncryptionCapable"`
	FileLevelEncryptionCapable             bool                                  `json:"fileLevelEncryptionCapable"`
	PasscodePresent                        bool                                  `json:"passcodePresent"`
	PasscodeCompliant                      bool                                  `json:"passcodeCompliant"`
	PasscodeCompliantWithProfile           bool                                  `json:"passcodeCompliantWithProfile"`
	HardwareEncryption                     int                                   `json:"hardwareEncryption"`
	ActivationLockEnabled                  bool                                  `json:"activationLockEnabled"`
	JailBreakDetected                      bool                                  `json:"jailBreakDetected"`
	AttestationStatus                      string                                `json:"attestationStatus"`
	LastAttestationAttemptDate             string                                `json:"lastAttestationAttemptDate"`
	LastSuccessfulAttestationDate          string                                `json:"lastSuccessfulAttestationDate"`
	PasscodeLockGracePeriodEnforcedSeconds int                                   `json:"passcodeLockGracePeriodEnforcedSeconds"`
	PersonalDeviceProfileCurrent           bool                                  `json:"personalDeviceProfileCurrent"`
	LostModeEnabled                        bool                                  `json:"lostModeEnabled"`
	LostModeMessage                        string                                `json:"lostModeMessage"`
	LostModePhoneNumber                    string                                `json:"lostModePhoneNumber"`
	LostModeFootnote                       string                                `json:"lostModeFootnote"`
	LostModeLocation                       InventoryMobileDeviceLostModeLocation `json:"lostModeLocation"`
}

// InventoryMobileDeviceApplication represents an application installed on the mobile device.
type InventoryMobileDeviceApplication struct {
	Identifier       string `json:"identifier"`
	Name             string `json:"name"`
	Version          string `json:"version"`
	ShortVersion     string `json:"shortVersion"`
	ManagementStatus string `json:"managementStatus"`
	ValidationStatus bool   `json:"validationStatus"`
	BundleSize       string `json:"bundleSize"`
	DynamicSize      string `json:"dynamicSize"`
}

// InventoryMobileDeviceProfile represents a configuration profile installed on the mobile device.
type InventoryMobileDeviceProfile struct {
	DisplayName   string `json:"displayName"`
	Version       string `json:"version"`
	Uuid          string `json:"uuid"`
	Identifier    string `json:"identifier"`
	Removable     bool   `json:"removable"`
	LastInstalled string `json:"lastInstalled"`
	Username      string `json:"username"`
}

// InventoryMobileDeviceCertificate represents a certificate installed on the mobile device.
type InventoryMobileDeviceCertificate struct {
	CommonName     string `json:"commonName"`
	Identity       bool   `json:"identity"`
	ExpirationDate string `json:"expirationDate"`
}

// InventoryMobileDeviceProvisioningProfile represents a provisioning profile on the mobile device.
type InventoryMobileDeviceProvisioningProfile struct {
	DisplayName    string `json:"displayName"`
	Uuid           string `json:"uuid"`
	ExpirationDate string `json:"expirationDate"`
}

// InventoryMobileDeviceServiceSubscriptions contains service subscription details for the mobile device.
type InventoryMobileDeviceServiceSubscriptions struct {
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

// InventoryMobileDeviceSharedUser represents a shared user on the mobile device.
type InventoryMobileDeviceSharedUser struct {
	ManagedAppleId string `json:"managedAppleId"`
	LoggedIn       bool   `json:"loggedIn"`
	DataToSync     bool   `json:"dataToSync"`
}

// InventoryMobileDeviceLostModeLocation contains lost mode location details for the mobile device.
type InventoryMobileDeviceLostModeLocation struct {
	LastLocationUpdate                       string  `json:"lastLocationUpdate"`
	LostModeLocationHorizontalAccuracyMeters float64 `json:"lostModeLocationHorizontalAccuracyMeters"`
	LostModeLocationVerticalAccuracyMeters   float64 `json:"lostModeLocationVerticalAccuracyMeters"`
	LostModeLocationAltitudeMeters           float64 `json:"lostModeLocationAltitudeMeters"`
	LostModeLocationSpeedMetersPerSecond     float64 `json:"lostModeLocationSpeedMetersPerSecond"`
	LostModeLocationCourseDegrees            float64 `json:"lostModeLocationCourseDegrees"`
	LostModeLocationTimestamp                string  `json:"lostModeLocationTimestamp"`
}

// InventoryMobileDeviceExtensionAttribute represents an extension attribute for the mobile device.
type InventoryMobileDeviceExtensionAttribute struct {
	Id                                  string   `json:"id"`
	Name                                string   `json:"name"`
	Type                                string   `json:"type"`
	Value                               []string `json:"value"`
	ExtensionAttributeCollectionAllowed bool     `json:"extensionAttributeCollectionAllowed"`
	InventoryDisplay                    string   `json:"inventoryDisplay"`
}

// InventoryMobileDeviceEbook represents an ebook assigned to the mobile device.
type InventoryMobileDeviceEbook struct {
	Author          string `json:"author"`
	Title           string `json:"title"`
	Version         string `json:"version"`
	Kind            string `json:"kind"`
	ManagementState string `json:"managementState"`
}

// InventoryMobileDeviceUserProfile represents a user profile on the mobile device.
type InventoryMobileDeviceUserProfile struct {
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	Uuid        string `json:"uuid"`
	Identifier  string `json:"identifier"`
	Removable   bool   `json:"removable"`
}

// InventoryEnrollmentMethodPrestage represents enrollment method prestage information.
type InventoryEnrollmentMethodPrestage struct {
	MobileDevicePrestageId string `json:"mobileDevicePrestageId"`
	ProfileName            string `json:"profileName"`
}

// InventoryJamfSchoolDeviceDetails contains Jamf School specific device details.
type InventoryJamfSchoolDeviceDetails struct {
	Udid string `json:"udid"`
}

// GetInventoryMobileDeviceByID fetches a single mobile device by ID from the Jamf Inventory API.
// sections parameter allows specifying which sections of data to retrieve (e.g., []string{"GENERAL", "HARDWARE"})
func (c *Client) GetInventoryMobileDeviceByID(ctx context.Context, id string, sections []string) (*InventoryMobileDevice, error) {
	params := url.Values{}
	for _, section := range sections {
		params.Add("section", section)
	}

	endpoint := fmt.Sprintf("/v1/mobile-devices/%s", url.PathEscape(id))
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get mobile device by id: %w", err)
	}
	var result InventoryMobileDevice
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryMobileDevices fetches a paginated list of mobile devices with optional sections and pagination.
// sections parameter allows specifying which sections of data to retrieve (e.g., []string{"GENERAL", "HARDWARE"})
func (c *Client) GetInventoryMobileDevices(ctx context.Context, page, pageSize int, sections []string) (*InventoryMobileDeviceSearchResults, error) {
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

	endpoint := "/v1/mobile-devices"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list mobile devices: %w", err)
	}
	var result InventoryMobileDeviceSearchResults
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryAllMobileDevices fetches all mobile devices by automatically handling pagination.
// It starts with page 0 and continues fetching until all devices are retrieved.
// sections parameter allows specifying which sections of data to retrieve (e.g., []string{"GENERAL", "HARDWARE"})
func (c *Client) GetInventoryAllMobileDevices(ctx context.Context, sections []string) ([]InventoryMobileDevice, error) {
	var allDevices []InventoryMobileDevice
	page := 0
	pageSize := 100

	for {
		result, err := c.GetInventoryMobileDevices(ctx, page, pageSize, sections)
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
