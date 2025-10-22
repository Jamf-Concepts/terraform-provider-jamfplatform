// Copyright 2025 Jamf Software LLC.
// https://developer.jamf.com/platform-api/reference/get_v1-computers-id
// https://developer.jamf.com/platform-api/reference/get_v1-computers

package client

import (
	"context"
	"fmt"
	"net/url"
)

// Constants used for the Jamf Inventory API
const (
	inventoryComputersV1Prefix = "/api/devices/v1/computers"
)

// InventoryComputerV1 represents a computer record from the Jamf Inventory API.
type InventoryComputerV1 struct {
	ID                    string                                    `json:"id"`
	UDID                  string                                    `json:"udid"`
	General               InventoryComputerGeneralV1                `json:"general"`
	DiskEncryption        InventoryComputerDiskEncryptionV1         `json:"diskEncryption"`
	Purchasing            InventoryComputerPurchaseV1               `json:"purchasing"`
	Applications          []InventoryComputerApplicationV1          `json:"applications"`
	Storage               InventoryComputerStorageV1                `json:"storage"`
	UserAndLocation       InventoryComputerUserAndLocationV1        `json:"userAndLocation"`
	ConfigurationProfiles []InventoryComputerConfigurationProfileV1 `json:"configurationProfiles"`
	Printers              []InventoryComputerPrinterV1              `json:"printers"`
	Services              []InventoryComputerServiceV1              `json:"services"`
	Hardware              InventoryComputerHardwareV1               `json:"hardware"`
	LocalUserAccounts     []InventoryComputerLocalUserAccountV1     `json:"localUserAccounts"`
	Certificates          []InventoryComputerCertificateV1          `json:"certificates"`
	Attachments           []InventoryComputerAttachmentV1           `json:"attachments"`
	Plugins               []InventoryComputerPluginV1               `json:"plugins"`
	PackageReceipts       InventoryComputerPackageReceiptsV1        `json:"packageReceipts"`
	Fonts                 []InventoryComputerFontV1                 `json:"fonts"`
	Security              InventoryComputerSecurityV1               `json:"security"`
	OperatingSystem       InventoryComputerOperatingSystemV1        `json:"operatingSystem"`
	LicensedSoftware      []InventoryComputerLicensedSoftwareV1     `json:"licensedSoftware"`
	Ibeacons              []InventoryComputerIbeaconV1              `json:"ibeacons"`
	SoftwareUpdates       []InventoryComputerSoftwareUpdateV1       `json:"softwareUpdates"`
	ExtensionAttributes   []InventoryComputerExtensionAttributeV1   `json:"extensionAttributes"`
	ContentCaching        InventoryComputerContentCachingV1         `json:"contentCaching"`
	GroupMemberships      []InventoryGroupMembershipV1              `json:"groupMemberships"`
	ProtectDetails        InventoryComputerProtectDetailsV1         `json:"protectDetails"`
	SchoolDetails         InventorySchoolDeviceDetailsV1            `json:"schoolDetails"`
}

// InventoryComputerGeneralV1 contains general information about the computer.
type InventoryComputerGeneralV1 struct {
	Name                 string                              `json:"name"`
	LastIpAddress        string                              `json:"lastIpAddress"`
	LastReportedIp       string                              `json:"lastReportedIp"`
	JamfBinaryVersion    string                              `json:"jamfBinaryVersion"`
	Platform             string                              `json:"platform"`
	Barcode1             string                              `json:"barcode1"`
	Barcode2             string                              `json:"barcode2"`
	AssetTag             string                              `json:"assetTag"`
	RemoteManagement     InventoryComputerRemoteManagementV1 `json:"remoteManagement"`
	Supervised           bool                                `json:"supervised"`
	MdmCapable           InventoryComputerMdmCapabilityV1    `json:"mdmCapable"`
	ReportDate           string                              `json:"reportDate"`
	LastContactTime      string                              `json:"lastContactTime"`
	LastCloudBackupDate  string                              `json:"lastCloudBackupDate"`
	LastEnrolledDate     string                              `json:"lastEnrolledDate"`
	MdmProfileExpiration string                              `json:"mdmProfileExpiration"`
	InitialEntryDate     string                              `json:"initialEntryDate"`
	DistributionPoint    string                              `json:"distributionPoint"`
	EnrollmentMethod     InventoryEnrollmentMethodV1         `json:"enrollmentMethod"`
	Site                 struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"site"`
	ItunesStoreAccountActive             bool                                    `json:"itunesStoreAccountActive"`
	EnrolledViaAutomatedDeviceEnrollment bool                                    `json:"enrolledViaAutomatedDeviceEnrollment"`
	UserApprovedMdm                      bool                                    `json:"userApprovedMdm"`
	DeclarativeDeviceManagementEnabled   bool                                    `json:"declarativeDeviceManagementEnabled"`
	ExtensionAttributes                  []InventoryComputerExtensionAttributeV1 `json:"extensionAttributes"`
	ManagementId                         string                                  `json:"managementId"`
}

// InventoryComputerRemoteManagementV1 contains remote management details for the computer.
type InventoryComputerRemoteManagementV1 struct {
	Managed            bool   `json:"managed"`
	ManagementUsername string `json:"managementUsername"`
}

// InventoryComputerMdmCapabilityV1 describes MDM capability and users for the computer.
type InventoryComputerMdmCapabilityV1 struct {
	Capable      bool     `json:"capable"`
	CapableUsers []string `json:"capableUsers"`
}

// InventoryComputerDiskEncryptionV1 contains disk encryption details for the computer.
type InventoryComputerDiskEncryptionV1 struct {
	BootPartitionEncryptionDetails      InventoryComputerPartitionEncryptionV1 `json:"bootPartitionEncryptionDetails"`
	IndividualRecoveryKeyValidityStatus string                                 `json:"individualRecoveryKeyValidityStatus"`
	InstitutionalRecoveryKeyPresent     bool                                   `json:"institutionalRecoveryKeyPresent"`
	DiskEncryptionConfigurationName     string                                 `json:"diskEncryptionConfigurationName"`
	FileVault2EnabledUserNames          []string                               `json:"fileVault2EnabledUserNames"`
	FileVault2EligibilityMessage        string                                 `json:"fileVault2EligibilityMessage"`
}

// InventoryComputerPartitionEncryptionV1 contains encryption details for a disk partition.
type InventoryComputerPartitionEncryptionV1 struct {
	PartitionName              string `json:"partitionName"`
	PartitionFileVault2State   string `json:"partitionFileVault2State"`
	PartitionFileVault2Percent int    `json:"partitionFileVault2Percent"`
}

// InventoryComputerPurchaseV1 contains purchasing and warranty information for the computer.
type InventoryComputerPurchaseV1 struct {
	Leased              bool                                    `json:"leased"`
	Purchased           bool                                    `json:"purchased"`
	PoNumber            string                                  `json:"poNumber"`
	PoDate              string                                  `json:"poDate"`
	Vendor              string                                  `json:"vendor"`
	WarrantyDate        string                                  `json:"warrantyDate"`
	AppleCareId         string                                  `json:"appleCareId"`
	LeaseDate           string                                  `json:"leaseDate"`
	PurchasePrice       string                                  `json:"purchasePrice"`
	LifeExpectancy      int                                     `json:"lifeExpectancy"`
	PurchasingAccount   string                                  `json:"purchasingAccount"`
	PurchasingContact   string                                  `json:"purchasingContact"`
	ExtensionAttributes []InventoryComputerExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryComputerApplicationV1 represents an application installed on the computer.
type InventoryComputerApplicationV1 struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	Version           string `json:"version"`
	MacAppStore       bool   `json:"macAppStore"`
	SizeMegabytes     int    `json:"sizeMegabytes"`
	BundleId          string `json:"bundleId"`
	UpdateAvailable   bool   `json:"updateAvailable"`
	ExternalVersionId string `json:"externalVersionId"`
}

// InventoryComputerStorageV1 contains storage and disk information for the computer.
type InventoryComputerStorageV1 struct {
	BootDriveAvailableSpaceMegabytes int64                     `json:"bootDriveAvailableSpaceMegabytes"`
	Disks                            []InventoryComputerDiskV1 `json:"disks"`
}

// InventoryComputerDiskV1 represents a physical or logical disk in the computer.
type InventoryComputerDiskV1 struct {
	ID            string                         `json:"id"`
	Device        string                         `json:"device"`
	Model         string                         `json:"model"`
	Revision      string                         `json:"revision"`
	SerialNumber  string                         `json:"serialNumber"`
	SizeMegabytes int                            `json:"sizeMegabytes"`
	SmartStatus   string                         `json:"smartStatus"`
	Type          string                         `json:"type"`
	Partitions    []InventoryComputerPartitionV1 `json:"partitions"`
}

// InventoryComputerPartitionV1 represents a partition on a disk.
type InventoryComputerPartitionV1 struct {
	Name                      string `json:"name"`
	SizeMegabytes             int    `json:"sizeMegabytes"`
	AvailableMegabytes        int    `json:"availableMegabytes"`
	PartitionType             string `json:"partitionType"`
	PercentUsed               int    `json:"percentUsed"`
	FileVault2State           string `json:"fileVault2State"`
	FileVault2ProgressPercent int    `json:"fileVault2ProgressPercent"`
	LvmManaged                bool   `json:"lvmManaged"`
}

// InventoryComputerUserAndLocationV1 contains user and location information for the computer.
type InventoryComputerUserAndLocationV1 struct {
	Username            string                                  `json:"username"`
	Realname            string                                  `json:"realname"`
	Email               string                                  `json:"email"`
	Position            string                                  `json:"position"`
	Phone               string                                  `json:"phone"`
	DepartmentId        string                                  `json:"departmentId"`
	BuildingId          string                                  `json:"buildingId"`
	Room                string                                  `json:"room"`
	ExtensionAttributes []InventoryComputerExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryComputerConfigurationProfileV1 represents a configuration profile installed on the computer.
type InventoryComputerConfigurationProfileV1 struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	LastInstalled     string `json:"lastInstalled"`
	Removable         bool   `json:"removable"`
	DisplayName       string `json:"displayName"`
	ProfileIdentifier string `json:"profileIdentifier"`
}

// InventoryComputerPrinterV1 represents a printer configured on the computer.
type InventoryComputerPrinterV1 struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Uri      string `json:"uri"`
	Location string `json:"location"`
}

// InventoryComputerServiceV1 represents a service running on the computer.
type InventoryComputerServiceV1 struct {
	Name string `json:"name"`
}

// InventoryComputerHardwareV1 contains hardware details for the computer.
type InventoryComputerHardwareV1 struct {
	Make                   string                                  `json:"make"`
	Model                  string                                  `json:"model"`
	ModelIdentifier        string                                  `json:"modelIdentifier"`
	SerialNumber           string                                  `json:"serialNumber"`
	ProcessorSpeedMhz      int                                     `json:"processorSpeedMhz"`
	ProcessorCount         int                                     `json:"processorCount"`
	CoreCount              int                                     `json:"coreCount"`
	ProcessorType          string                                  `json:"processorType"`
	ProcessorArchitecture  string                                  `json:"processorArchitecture"`
	BusSpeedMhz            int                                     `json:"busSpeedMhz"`
	CacheSizeKilobytes     int                                     `json:"cacheSizeKilobytes"`
	NetworkAdapterType     string                                  `json:"networkAdapterType"`
	MacAddress             string                                  `json:"macAddress"`
	AltNetworkAdapterType  string                                  `json:"altNetworkAdapterType"`
	AltMacAddress          string                                  `json:"altMacAddress"`
	TotalRamMegabytes      int                                     `json:"totalRamMegabytes"`
	OpenRamSlots           int                                     `json:"openRamSlots"`
	BatteryCapacityPercent int                                     `json:"batteryCapacityPercent"`
	SmcVersion             string                                  `json:"smcVersion"`
	NicSpeed               string                                  `json:"nicSpeed"`
	OpticalDrive           string                                  `json:"opticalDrive"`
	BootRom                string                                  `json:"bootRom"`
	BleCapable             bool                                    `json:"bleCapable"`
	SupportsIosAppInstalls bool                                    `json:"supportsIosAppInstalls"`
	AppleSilicon           bool                                    `json:"appleSilicon"`
	ProvisioningUdid       string                                  `json:"provisioningUdid"`
	ExtensionAttributes    []InventoryComputerExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryComputerLocalUserAccountV1 represents a local user account on the computer.
type InventoryComputerLocalUserAccountV1 struct {
	Uid                            string `json:"uid"`
	UserGuid                       string `json:"userGuid"`
	Username                       string `json:"username"`
	FullName                       string `json:"fullName"`
	Admin                          bool   `json:"admin"`
	HomeDirectory                  string `json:"homeDirectory"`
	HomeDirectorySizeMb            int64  `json:"homeDirectorySizeMb"`
	FileVault2Enabled              bool   `json:"fileVault2Enabled"`
	UserAccountType                string `json:"userAccountType"`
	PasswordMinLength              int    `json:"passwordMinLength"`
	PasswordMaxAge                 int    `json:"passwordMaxAge"`
	PasswordMinComplexCharacters   int    `json:"passwordMinComplexCharacters"`
	PasswordHistoryDepth           int    `json:"passwordHistoryDepth"`
	PasswordRequireAlphanumeric    bool   `json:"passwordRequireAlphanumeric"`
	ComputerAzureActiveDirectoryId string `json:"computerAzureActiveDirectoryId"`
	UserAzureActiveDirectoryId     string `json:"userAzureActiveDirectoryId"`
	AzureActiveDirectoryId         string `json:"azureActiveDirectoryId"`
}

// InventoryComputerCertificateV1 represents a certificate installed on the computer.
type InventoryComputerCertificateV1 struct {
	CommonName        string `json:"commonName"`
	Identity          bool   `json:"identity"`
	ExpirationDate    string `json:"expirationDate"`
	Username          string `json:"username"`
	LifecycleStatus   string `json:"lifecycleStatus"`
	CertificateStatus string `json:"certificateStatus"`
	SubjectName       string `json:"subjectName"`
	SerialNumber      string `json:"serialNumber"`
	Sha1Fingerprint   string `json:"sha1Fingerprint"`
	IssuedDate        string `json:"issuedDate"`
}

// InventoryComputerAttachmentV1 represents an attachment associated with the computer.
type InventoryComputerAttachmentV1 struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FileType  string `json:"fileType"`
	SizeBytes int64  `json:"sizeBytes"`
}

// InventoryComputerPluginV1 represents a plugin installed on the computer.
type InventoryComputerPluginV1 struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// InventoryComputerPackageReceiptsV1 contains package receipt information for the computer.
type InventoryComputerPackageReceiptsV1 struct {
	InstalledByJamfPro      []string `json:"installedByJamfPro"`
	InstalledByInstallerSwu []string `json:"installedByInstallerSwu"`
	Cached                  []string `json:"cached"`
}

// InventoryComputerFontV1 represents a font installed on the computer.
type InventoryComputerFontV1 struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// InventoryComputerSecurityV1 contains security status and settings for the computer.
type InventoryComputerSecurityV1 struct {
	SipStatus                    string `json:"sipStatus"`
	GatekeeperStatus             string `json:"gatekeeperStatus"`
	XprotectVersion              string `json:"xprotectVersion"`
	AutoLoginDisabled            bool   `json:"autoLoginDisabled"`
	RemoteDesktopEnabled         bool   `json:"remoteDesktopEnabled"`
	ActivationLockEnabled        bool   `json:"activationLockEnabled"`
	RecoveryLockEnabled          bool   `json:"recoveryLockEnabled"`
	SecureBootLevel              string `json:"secureBootLevel"`
	ExternalBootLevel            string `json:"externalBootLevel"`
	BootstrapTokenAllowed        bool   `json:"bootstrapTokenAllowed"`
	BootstrapTokenEscrowedStatus string `json:"bootstrapTokenEscrowedStatus"`
}

// InventoryComputerOperatingSystemV1 contains operating system details for the computer.
type InventoryComputerOperatingSystemV1 struct {
	Name                     string                                  `json:"name"`
	Version                  string                                  `json:"version"`
	Build                    string                                  `json:"build"`
	SupplementalBuildVersion string                                  `json:"supplementalBuildVersion"`
	RapidSecurityResponse    string                                  `json:"rapidSecurityResponse"`
	ActiveDirectoryStatus    string                                  `json:"activeDirectoryStatus"`
	FileVault2Status         string                                  `json:"fileVault2Status"`
	SoftwareUpdateDeviceId   string                                  `json:"softwareUpdateDeviceId"`
	ExtensionAttributes      []InventoryComputerExtensionAttributeV1 `json:"extensionAttributes"`
}

// InventoryComputerLicensedSoftwareV1 represents licensed software assigned to the computer.
type InventoryComputerLicensedSoftwareV1 struct {
	ID string `json:"id"`
}

// InventoryComputerIbeaconV1 represents an iBeacon associated with the computer.
type InventoryComputerIbeaconV1 struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InventoryComputerSoftwareUpdateV1 represents a software update available for the computer.
type InventoryComputerSoftwareUpdateV1 struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	PackageName string `json:"packageName"`
}

// InventoryComputerExtensionAttributeV1 represents an extension attribute for the computer.
type InventoryComputerExtensionAttributeV1 struct {
	DefinitionId string   `json:"definitionId"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Enabled      bool     `json:"enabled"`
	MultiValue   bool     `json:"multiValue"`
	Values       []string `json:"values"`
	DataType     string   `json:"dataType"`
	Options      []string `json:"options"`
	InputType    string   `json:"inputType"`
}

// InventoryComputerContentCachingV1 contains content caching information for the computer.
type InventoryComputerContentCachingV1 struct {
	ComputerContentCachingInformationId string                                              `json:"computerContentCachingInformationId"`
	Parents                             []InventoryComputerContentCachingParentV1           `json:"parents"`
	Alerts                              []InventoryComputerContentCachingAlertV1            `json:"alerts"`
	Activated                           bool                                                `json:"activated"`
	Active                              bool                                                `json:"active"`
	ActualCacheBytesUsed                int64                                               `json:"actualCacheBytesUsed"`
	CacheDetails                        []InventoryComputerContentCachingCacheDetailV1      `json:"cacheDetails"`
	CacheBytesFree                      int64                                               `json:"cacheBytesFree"`
	CacheBytesLimit                     int64                                               `json:"cacheBytesLimit"`
	CacheStatus                         string                                              `json:"cacheStatus"`
	CacheBytesUsed                      int64                                               `json:"cacheBytesUsed"`
	DataMigrationCompleted              bool                                                `json:"dataMigrationCompleted"`
	DataMigrationProgressPercentage     int                                                 `json:"dataMigrationProgressPercentage"`
	DataMigrationError                  InventoryComputerContentCachingDataMigrationErrorV1 `json:"dataMigrationError"`
	MaxCachePressureLast1HourPercentage int                                                 `json:"maxCachePressureLast1HourPercentage"`
	PersonalCacheBytesFree              int64                                               `json:"personalCacheBytesFree"`
	PersonalCacheBytesLimit             int64                                               `json:"personalCacheBytesLimit"`
	PersonalCacheBytesUsed              int64                                               `json:"personalCacheBytesUsed"`
	Port                                int                                                 `json:"port"`
	PublicAddress                       string                                              `json:"publicAddress"`
	RegistrationError                   string                                              `json:"registrationError"`
	RegistrationResponseCode            int                                                 `json:"registrationResponseCode"`
	RegistrationStarted                 string                                              `json:"registrationStarted"`
	RegistrationStatus                  string                                              `json:"registrationStatus"`
	RestrictedMedia                     bool                                                `json:"restrictedMedia"`
	ServerGuid                          string                                              `json:"serverGuid"`
}

// InventoryComputerContentCachingParentV1 represents a parent in the content caching hierarchy.
type InventoryComputerContentCachingParentV1 struct {
	ContentCachingParentId string                                         `json:"contentCachingParentId"`
	Address                string                                         `json:"address"`
	Alerts                 InventoryComputerContentCachingParentAlertV1   `json:"alerts"`
	Details                InventoryComputerContentCachingParentDetailsV1 `json:"details"`
	Guid                   string                                         `json:"guid"`
	Healthy                bool                                           `json:"healthy"`
	Port                   int                                            `json:"port"`
	Version                string                                         `json:"version"`
}

// InventoryComputerContentCachingParentAlertV1 represents an alert for a content caching parent.
type InventoryComputerContentCachingParentAlertV1 struct {
	ContentCachingParentAlertId string   `json:"contentCachingParentAlertId"`
	Addresses                   []string `json:"addresses"`
	ClassName                   string   `json:"className"`
	PostDate                    string   `json:"postDate"`
}

// InventoryComputerContentCachingParentDetailsV1 contains details about a content caching parent.
type InventoryComputerContentCachingParentDetailsV1 struct {
	ContentCachingParentDetailsId string                                                `json:"contentCachingParentDetailsId"`
	AcPower                       bool                                                  `json:"acPower"`
	CacheSizeBytes                int64                                                 `json:"cacheSizeBytes"`
	Capabilities                  InventoryComputerContentCachingParentCapabilitiesV1   `json:"capabilities"`
	Portable                      bool                                                  `json:"portable"`
	LocalNetwork                  []InventoryComputerContentCachingParentLocalNetworkV1 `json:"localNetwork"`
}

// InventoryComputerContentCachingParentCapabilitiesV1 describes capabilities of a content caching parent.
type InventoryComputerContentCachingParentCapabilitiesV1 struct {
	ContentCachingParentCapabilitiesId string `json:"contentCachingParentCapabilitiesId"`
	Imports                            bool   `json:"imports"`
	Namespaces                         bool   `json:"namespaces"`
	PersonalContent                    bool   `json:"personalContent"`
	QueryParameters                    bool   `json:"queryParameters"`
	SharedContent                      bool   `json:"sharedContent"`
	Prioritization                     bool   `json:"prioritization"`
}

// InventoryComputerContentCachingParentLocalNetworkV1 represents a local network for a content caching parent.
type InventoryComputerContentCachingParentLocalNetworkV1 struct {
	ContentCachingParentLocalNetworkId string `json:"contentCachingParentLocalNetworkId"`
	Speed                              int    `json:"speed"`
	Wired                              bool   `json:"wired"`
}

// InventoryComputerContentCachingAlertV1 represents a content caching alert for the computer.
type InventoryComputerContentCachingAlertV1 struct {
	CacheBytesLimit int `json:"cacheBytesLimit"`
}

// InventoryComputerContentCachingCacheDetailV1 contains cache details for content caching.
type InventoryComputerContentCachingCacheDetailV1 struct {
	ComputerContentCachingCacheDetailsId string `json:"computerContentCachingCacheDetailsId"`
}

// InventoryComputerContentCachingDataMigrationErrorV1 represents a data migration error in content caching.
type InventoryComputerContentCachingDataMigrationErrorV1 struct {
	Code     int                                                           `json:"code"`
	Domain   string                                                        `json:"domain"`
	UserInfo []InventoryComputerContentCachingDataMigrationErrorUserInfoV1 `json:"userInfo"`
}

// InventoryComputerContentCachingDataMigrationErrorUserInfoV1 contains user info for a data migration error.
type InventoryComputerContentCachingDataMigrationErrorUserInfoV1 struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// InventoryGroupMembershipV1 represents a group membership for the computer.
type InventoryGroupMembershipV1 struct {
	GroupId string `json:"groupId"`
}

// InventoryComputerProtectDetailsV1 contains Jamf Protect details for the computer.
type InventoryComputerProtectDetailsV1 struct {
	Uuid string `json:"uuid"`
}

// InventorySchoolDeviceDetailsV1 contains school-related details for the computer.
type InventorySchoolDeviceDetailsV1 struct {
	Udid string `json:"udid"`
}

// InventoryEnrollmentMethodV1 describes the enrollment method for the computer.
type InventoryEnrollmentMethodV1 struct {
	ID string `json:"id"`
}

// InventoryComputerSearchResultsV1 represents a paginated list of inventory computers.
type InventoryComputerSearchResultsV1 struct {
	TotalCount int                   `json:"totalCount"`
	Results    []InventoryComputerV1 `json:"results"`
}

// GetInventoryComputerByIDV1 fetches a single computer by ID
func (c *Client) GetInventoryComputerByIDV1(ctx context.Context, id string) (*InventoryComputerV1, error) {
	endpoint := fmt.Sprintf("%s/%s", inventoryComputersV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get computer by id: %w", err)
	}
	var result InventoryComputerV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryComputersV1 fetches a paginated list of computers, with optional filter
func (c *Client) GetInventoryComputersV1(ctx context.Context, page, pageSize int, filter string) (*InventoryComputerSearchResultsV1, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		params.Set("page-size", fmt.Sprintf("%d", pageSize))
	}
	if filter != "" {
		params.Set("filter", filter)
	}
	endpoint := inventoryComputersV1Prefix
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list computers: %w", err)
	}
	var result InventoryComputerSearchResultsV1
	if err := c.handleAPIResponse(ctx, resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryAllComputersV1 fetches all computers by automatically handling pagination.
// It starts with page 0 and continues fetching until all computers are retrieved.
func (c *Client) GetInventoryAllComputersV1(ctx context.Context, filter string) ([]InventoryComputerV1, error) {
	var allComputers []InventoryComputerV1
	page := 0
	pageSize := 100

	for {
		result, err := c.GetInventoryComputersV1(ctx, page, pageSize, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch computers page %d: %w", page, err)
		}

		allComputers = append(allComputers, result.Results...)

		if len(allComputers) >= result.TotalCount || len(result.Results) < pageSize {
			break
		}

		page++
	}

	return allComputers, nil
}
