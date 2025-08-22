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

// InventoryComputer represents a computer record from the Jamf Inventory API.
type InventoryComputer struct {
	ID                    string                                  `json:"id"`
	UDID                  string                                  `json:"udid"`
	General               InventoryComputerGeneral                `json:"general"`
	DiskEncryption        InventoryComputerDiskEncryption         `json:"diskEncryption"`
	Purchasing            InventoryComputerPurchase               `json:"purchasing"`
	Applications          []InventoryComputerApplication          `json:"applications"`
	Storage               InventoryComputerStorage                `json:"storage"`
	UserAndLocation       InventoryComputerUserAndLocation        `json:"userAndLocation"`
	ConfigurationProfiles []InventoryComputerConfigurationProfile `json:"configurationProfiles"`
	Printers              []InventoryComputerPrinter              `json:"printers"`
	Services              []InventoryComputerService              `json:"services"`
	Hardware              InventoryComputerHardware               `json:"hardware"`
	LocalUserAccounts     []InventoryComputerLocalUserAccount     `json:"localUserAccounts"`
	Certificates          []InventoryComputerCertificate          `json:"certificates"`
	Attachments           []InventoryComputerAttachment           `json:"attachments"`
	Plugins               []InventoryComputerPlugin               `json:"plugins"`
	PackageReceipts       InventoryComputerPackageReceipts        `json:"packageReceipts"`
	Fonts                 []InventoryComputerFont                 `json:"fonts"`
	Security              InventoryComputerSecurity               `json:"security"`
	OperatingSystem       InventoryComputerOperatingSystem        `json:"operatingSystem"`
	LicensedSoftware      []InventoryComputerLicensedSoftware     `json:"licensedSoftware"`
	Ibeacons              []InventoryComputerIbeacon              `json:"ibeacons"`
	SoftwareUpdates       []InventoryComputerSoftwareUpdate       `json:"softwareUpdates"`
	ExtensionAttributes   []InventoryComputerExtensionAttribute   `json:"extensionAttributes"`
	ContentCaching        InventoryComputerContentCaching         `json:"contentCaching"`
	GroupMemberships      []InventoryGroupMembership              `json:"groupMemberships"`
	ProtectDetails        InventoryComputerProtectDetails         `json:"protectDetails"`
	SchoolDetails         InventorySchoolDeviceDetails            `json:"schoolDetails"`
}

// InventoryComputerGeneral contains general information about the computer.
type InventoryComputerGeneral struct {
	Name                 string                            `json:"name"`
	LastIpAddress        string                            `json:"lastIpAddress"`
	LastReportedIp       string                            `json:"lastReportedIp"`
	JamfBinaryVersion    string                            `json:"jamfBinaryVersion"`
	Platform             string                            `json:"platform"`
	Barcode1             string                            `json:"barcode1"`
	Barcode2             string                            `json:"barcode2"`
	AssetTag             string                            `json:"assetTag"`
	RemoteManagement     InventoryComputerRemoteManagement `json:"remoteManagement"`
	Supervised           bool                              `json:"supervised"`
	MdmCapable           InventoryComputerMdmCapability    `json:"mdmCapable"`
	ReportDate           string                            `json:"reportDate"`
	LastContactTime      string                            `json:"lastContactTime"`
	LastCloudBackupDate  string                            `json:"lastCloudBackupDate"`
	LastEnrolledDate     string                            `json:"lastEnrolledDate"`
	MdmProfileExpiration string                            `json:"mdmProfileExpiration"`
	InitialEntryDate     string                            `json:"initialEntryDate"`
	DistributionPoint    string                            `json:"distributionPoint"`
	EnrollmentMethod     InventoryEnrollmentMethod         `json:"enrollmentMethod"`
	Site                 struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"site"`
	ItunesStoreAccountActive             bool                                  `json:"itunesStoreAccountActive"`
	EnrolledViaAutomatedDeviceEnrollment bool                                  `json:"enrolledViaAutomatedDeviceEnrollment"`
	UserApprovedMdm                      bool                                  `json:"userApprovedMdm"`
	DeclarativeDeviceManagementEnabled   bool                                  `json:"declarativeDeviceManagementEnabled"`
	ExtensionAttributes                  []InventoryComputerExtensionAttribute `json:"extensionAttributes"`
	ManagementId                         string                                `json:"managementId"`
}

// InventoryComputerRemoteManagement contains remote management details for the computer.
type InventoryComputerRemoteManagement struct {
	Managed            bool   `json:"managed"`
	ManagementUsername string `json:"managementUsername"`
}

// InventoryComputerMdmCapability describes MDM capability and users for the computer.
type InventoryComputerMdmCapability struct {
	Capable      bool     `json:"capable"`
	CapableUsers []string `json:"capableUsers"`
}

// InventoryComputerDiskEncryption contains disk encryption details for the computer.
type InventoryComputerDiskEncryption struct {
	BootPartitionEncryptionDetails      InventoryComputerPartitionEncryption `json:"bootPartitionEncryptionDetails"`
	IndividualRecoveryKeyValidityStatus string                               `json:"individualRecoveryKeyValidityStatus"`
	InstitutionalRecoveryKeyPresent     bool                                 `json:"institutionalRecoveryKeyPresent"`
	DiskEncryptionConfigurationName     string                               `json:"diskEncryptionConfigurationName"`
	FileVault2EnabledUserNames          []string                             `json:"fileVault2EnabledUserNames"`
	FileVault2EligibilityMessage        string                               `json:"fileVault2EligibilityMessage"`
}

// InventoryComputerPartitionEncryption contains encryption details for a disk partition.
type InventoryComputerPartitionEncryption struct {
	PartitionName              string `json:"partitionName"`
	PartitionFileVault2State   string `json:"partitionFileVault2State"`
	PartitionFileVault2Percent int    `json:"partitionFileVault2Percent"`
}

// InventoryComputerPurchase contains purchasing and warranty information for the computer.
type InventoryComputerPurchase struct {
	Leased              bool                                  `json:"leased"`
	Purchased           bool                                  `json:"purchased"`
	PoNumber            string                                `json:"poNumber"`
	PoDate              string                                `json:"poDate"`
	Vendor              string                                `json:"vendor"`
	WarrantyDate        string                                `json:"warrantyDate"`
	AppleCareId         string                                `json:"appleCareId"`
	LeaseDate           string                                `json:"leaseDate"`
	PurchasePrice       string                                `json:"purchasePrice"`
	LifeExpectancy      int                                   `json:"lifeExpectancy"`
	PurchasingAccount   string                                `json:"purchasingAccount"`
	PurchasingContact   string                                `json:"purchasingContact"`
	ExtensionAttributes []InventoryComputerExtensionAttribute `json:"extensionAttributes"`
}

// InventoryComputerApplication represents an application installed on the computer.
type InventoryComputerApplication struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	Version           string `json:"version"`
	MacAppStore       bool   `json:"macAppStore"`
	SizeMegabytes     int    `json:"sizeMegabytes"`
	BundleId          string `json:"bundleId"`
	UpdateAvailable   bool   `json:"updateAvailable"`
	ExternalVersionId string `json:"externalVersionId"`
}

// InventoryComputerStorage contains storage and disk information for the computer.
type InventoryComputerStorage struct {
	BootDriveAvailableSpaceMegabytes int64                   `json:"bootDriveAvailableSpaceMegabytes"`
	Disks                            []InventoryComputerDisk `json:"disks"`
}

// InventoryComputerDisk represents a physical or logical disk in the computer.
type InventoryComputerDisk struct {
	ID            string                       `json:"id"`
	Device        string                       `json:"device"`
	Model         string                       `json:"model"`
	Revision      string                       `json:"revision"`
	SerialNumber  string                       `json:"serialNumber"`
	SizeMegabytes int                          `json:"sizeMegabytes"`
	SmartStatus   string                       `json:"smartStatus"`
	Type          string                       `json:"type"`
	Partitions    []InventoryComputerPartition `json:"partitions"`
}

// InventoryComputerPartition represents a partition on a disk.
type InventoryComputerPartition struct {
	Name                      string `json:"name"`
	SizeMegabytes             int    `json:"sizeMegabytes"`
	AvailableMegabytes        int    `json:"availableMegabytes"`
	PartitionType             string `json:"partitionType"`
	PercentUsed               int    `json:"percentUsed"`
	FileVault2State           string `json:"fileVault2State"`
	FileVault2ProgressPercent int    `json:"fileVault2ProgressPercent"`
	LvmManaged                bool   `json:"lvmManaged"`
}

// InventoryComputerUserAndLocation contains user and location information for the computer.
type InventoryComputerUserAndLocation struct {
	Username            string                                `json:"username"`
	Realname            string                                `json:"realname"`
	Email               string                                `json:"email"`
	Position            string                                `json:"position"`
	Phone               string                                `json:"phone"`
	DepartmentId        string                                `json:"departmentId"`
	BuildingId          string                                `json:"buildingId"`
	Room                string                                `json:"room"`
	ExtensionAttributes []InventoryComputerExtensionAttribute `json:"extensionAttributes"`
}

// InventoryComputerConfigurationProfile represents a configuration profile installed on the computer.
type InventoryComputerConfigurationProfile struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	LastInstalled     string `json:"lastInstalled"`
	Removable         bool   `json:"removable"`
	DisplayName       string `json:"displayName"`
	ProfileIdentifier string `json:"profileIdentifier"`
}

// InventoryComputerPrinter represents a printer configured on the computer.
type InventoryComputerPrinter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Uri      string `json:"uri"`
	Location string `json:"location"`
}

// InventoryComputerService represents a service running on the computer.
type InventoryComputerService struct {
	Name string `json:"name"`
}

// InventoryComputerHardware contains hardware details for the computer.
type InventoryComputerHardware struct {
	Make                   string                                `json:"make"`
	Model                  string                                `json:"model"`
	ModelIdentifier        string                                `json:"modelIdentifier"`
	SerialNumber           string                                `json:"serialNumber"`
	ProcessorSpeedMhz      int                                   `json:"processorSpeedMhz"`
	ProcessorCount         int                                   `json:"processorCount"`
	CoreCount              int                                   `json:"coreCount"`
	ProcessorType          string                                `json:"processorType"`
	ProcessorArchitecture  string                                `json:"processorArchitecture"`
	BusSpeedMhz            int                                   `json:"busSpeedMhz"`
	CacheSizeKilobytes     int                                   `json:"cacheSizeKilobytes"`
	NetworkAdapterType     string                                `json:"networkAdapterType"`
	MacAddress             string                                `json:"macAddress"`
	AltNetworkAdapterType  string                                `json:"altNetworkAdapterType"`
	AltMacAddress          string                                `json:"altMacAddress"`
	TotalRamMegabytes      int                                   `json:"totalRamMegabytes"`
	OpenRamSlots           int                                   `json:"openRamSlots"`
	BatteryCapacityPercent int                                   `json:"batteryCapacityPercent"`
	SmcVersion             string                                `json:"smcVersion"`
	NicSpeed               string                                `json:"nicSpeed"`
	OpticalDrive           string                                `json:"opticalDrive"`
	BootRom                string                                `json:"bootRom"`
	BleCapable             bool                                  `json:"bleCapable"`
	SupportsIosAppInstalls bool                                  `json:"supportsIosAppInstalls"`
	AppleSilicon           bool                                  `json:"appleSilicon"`
	ProvisioningUdid       string                                `json:"provisioningUdid"`
	ExtensionAttributes    []InventoryComputerExtensionAttribute `json:"extensionAttributes"`
}

// InventoryComputerLocalUserAccount represents a local user account on the computer.
type InventoryComputerLocalUserAccount struct {
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

// InventoryComputerCertificate represents a certificate installed on the computer.
type InventoryComputerCertificate struct {
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

// InventoryComputerAttachment represents an attachment associated with the computer.
type InventoryComputerAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FileType  string `json:"fileType"`
	SizeBytes int64  `json:"sizeBytes"`
}

// InventoryComputerPlugin represents a plugin installed on the computer.
type InventoryComputerPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// InventoryComputerPackageReceipts contains package receipt information for the computer.
type InventoryComputerPackageReceipts struct {
	InstalledByJamfPro      []string `json:"installedByJamfPro"`
	InstalledByInstallerSwu []string `json:"installedByInstallerSwu"`
	Cached                  []string `json:"cached"`
}

// InventoryComputerFont represents a font installed on the computer.
type InventoryComputerFont struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// InventoryComputerSecurity contains security status and settings for the computer.
type InventoryComputerSecurity struct {
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

// InventoryComputerOperatingSystem contains operating system details for the computer.
type InventoryComputerOperatingSystem struct {
	Name                     string                                `json:"name"`
	Version                  string                                `json:"version"`
	Build                    string                                `json:"build"`
	SupplementalBuildVersion string                                `json:"supplementalBuildVersion"`
	RapidSecurityResponse    string                                `json:"rapidSecurityResponse"`
	ActiveDirectoryStatus    string                                `json:"activeDirectoryStatus"`
	FileVault2Status         string                                `json:"fileVault2Status"`
	SoftwareUpdateDeviceId   string                                `json:"softwareUpdateDeviceId"`
	ExtensionAttributes      []InventoryComputerExtensionAttribute `json:"extensionAttributes"`
}

// InventoryComputerLicensedSoftware represents licensed software assigned to the computer.
type InventoryComputerLicensedSoftware struct {
	ID string `json:"id"`
}

// InventoryComputerIbeacon represents an iBeacon associated with the computer.
type InventoryComputerIbeacon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InventoryComputerSoftwareUpdate represents a software update available for the computer.
type InventoryComputerSoftwareUpdate struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	PackageName string `json:"packageName"`
}

// InventoryComputerExtensionAttribute represents an extension attribute for the computer.
type InventoryComputerExtensionAttribute struct {
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

// InventoryComputerContentCaching contains content caching information for the computer.
type InventoryComputerContentCaching struct {
	ComputerContentCachingInformationId string                                            `json:"computerContentCachingInformationId"`
	Parents                             []InventoryComputerContentCachingParent           `json:"parents"`
	Alerts                              []InventoryComputerContentCachingAlert            `json:"alerts"`
	Activated                           bool                                              `json:"activated"`
	Active                              bool                                              `json:"active"`
	ActualCacheBytesUsed                int64                                             `json:"actualCacheBytesUsed"`
	CacheDetails                        []InventoryComputerContentCachingCacheDetail      `json:"cacheDetails"`
	CacheBytesFree                      int64                                             `json:"cacheBytesFree"`
	CacheBytesLimit                     int64                                             `json:"cacheBytesLimit"`
	CacheStatus                         string                                            `json:"cacheStatus"`
	CacheBytesUsed                      int64                                             `json:"cacheBytesUsed"`
	DataMigrationCompleted              bool                                              `json:"dataMigrationCompleted"`
	DataMigrationProgressPercentage     int                                               `json:"dataMigrationProgressPercentage"`
	DataMigrationError                  InventoryComputerContentCachingDataMigrationError `json:"dataMigrationError"`
	MaxCachePressureLast1HourPercentage int                                               `json:"maxCachePressureLast1HourPercentage"`
	PersonalCacheBytesFree              int64                                             `json:"personalCacheBytesFree"`
	PersonalCacheBytesLimit             int64                                             `json:"personalCacheBytesLimit"`
	PersonalCacheBytesUsed              int64                                             `json:"personalCacheBytesUsed"`
	Port                                int                                               `json:"port"`
	PublicAddress                       string                                            `json:"publicAddress"`
	RegistrationError                   string                                            `json:"registrationError"`
	RegistrationResponseCode            int                                               `json:"registrationResponseCode"`
	RegistrationStarted                 string                                            `json:"registrationStarted"`
	RegistrationStatus                  string                                            `json:"registrationStatus"`
	RestrictedMedia                     bool                                              `json:"restrictedMedia"`
	ServerGuid                          string                                            `json:"serverGuid"`
}

// InventoryComputerContentCachingParent represents a parent in the content caching hierarchy.
type InventoryComputerContentCachingParent struct {
	ContentCachingParentId string                                       `json:"contentCachingParentId"`
	Address                string                                       `json:"address"`
	Alerts                 InventoryComputerContentCachingParentAlert   `json:"alerts"`
	Details                InventoryComputerContentCachingParentDetails `json:"details"`
	Guid                   string                                       `json:"guid"`
	Healthy                bool                                         `json:"healthy"`
	Port                   int                                          `json:"port"`
	Version                string                                       `json:"version"`
}

// InventoryComputerContentCachingParentAlert represents an alert for a content caching parent.
type InventoryComputerContentCachingParentAlert struct {
	ContentCachingParentAlertId string   `json:"contentCachingParentAlertId"`
	Addresses                   []string `json:"addresses"`
	ClassName                   string   `json:"className"`
	PostDate                    string   `json:"postDate"`
}

// InventoryComputerContentCachingParentDetails contains details about a content caching parent.
type InventoryComputerContentCachingParentDetails struct {
	ContentCachingParentDetailsId string                                              `json:"contentCachingParentDetailsId"`
	AcPower                       bool                                                `json:"acPower"`
	CacheSizeBytes                int64                                               `json:"cacheSizeBytes"`
	Capabilities                  InventoryComputerContentCachingParentCapabilities   `json:"capabilities"`
	Portable                      bool                                                `json:"portable"`
	LocalNetwork                  []InventoryComputerContentCachingParentLocalNetwork `json:"localNetwork"`
}

// InventoryComputerContentCachingParentCapabilities describes capabilities of a content caching parent.
type InventoryComputerContentCachingParentCapabilities struct {
	ContentCachingParentCapabilitiesId string `json:"contentCachingParentCapabilitiesId"`
	Imports                            bool   `json:"imports"`
	Namespaces                         bool   `json:"namespaces"`
	PersonalContent                    bool   `json:"personalContent"`
	QueryParameters                    bool   `json:"queryParameters"`
	SharedContent                      bool   `json:"sharedContent"`
	Prioritization                     bool   `json:"prioritization"`
}

// InventoryComputerContentCachingParentLocalNetwork represents a local network for a content caching parent.
type InventoryComputerContentCachingParentLocalNetwork struct {
	ContentCachingParentLocalNetworkId string `json:"contentCachingParentLocalNetworkId"`
	Speed                              int    `json:"speed"`
	Wired                              bool   `json:"wired"`
}

// InventoryComputerContentCachingAlert represents a content caching alert for the computer.
type InventoryComputerContentCachingAlert struct {
	CacheBytesLimit int `json:"cacheBytesLimit"`
}

// InventoryComputerContentCachingCacheDetail contains cache details for content caching.
type InventoryComputerContentCachingCacheDetail struct {
	ComputerContentCachingCacheDetailsId string `json:"computerContentCachingCacheDetailsId"`
}

// InventoryComputerContentCachingDataMigrationError represents a data migration error in content caching.
type InventoryComputerContentCachingDataMigrationError struct {
	Code     int                                                         `json:"code"`
	Domain   string                                                      `json:"domain"`
	UserInfo []InventoryComputerContentCachingDataMigrationErrorUserInfo `json:"userInfo"`
}

// InventoryComputerContentCachingDataMigrationErrorUserInfo contains user info for a data migration error.
type InventoryComputerContentCachingDataMigrationErrorUserInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// InventoryGroupMembership represents a group membership for the computer.
type InventoryGroupMembership struct {
	GroupId string `json:"groupId"`
}

// InventoryComputerProtectDetails contains Jamf Protect details for the computer.
type InventoryComputerProtectDetails struct {
	Uuid string `json:"uuid"`
}

// InventorySchoolDeviceDetails contains school-related details for the computer.
type InventorySchoolDeviceDetails struct {
	Udid string `json:"udid"`
}

// InventoryEnrollmentMethod describes the enrollment method for the computer.
type InventoryEnrollmentMethod struct {
	ID string `json:"id"`
}

// InventoryComputerSearchResults represents a paginated list of inventory computers.
type InventoryComputerSearchResults struct {
	TotalCount int                 `json:"totalCount"`
	Results    []InventoryComputer `json:"results"`
}

// GetInventoryComputerByID fetches a single computer by ID
func (c *Client) GetInventoryComputerByID(ctx context.Context, id string) (*InventoryComputer, error) {
	endpoint := fmt.Sprintf("%s/%s", inventoryComputersV1Prefix, url.PathEscape(id))
	resp, err := c.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get computer by id: %w", err)
	}
	var result InventoryComputer
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryComputers fetches a paginated list of computers, with optional filter
func (c *Client) GetInventoryComputers(ctx context.Context, page, pageSize int, filter string) (*InventoryComputerSearchResults, error) {
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
	var result InventoryComputerSearchResults
	if err := c.handleAPIResponse(resp, 200, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInventoryAllComputers fetches all computers by automatically handling pagination.
// It starts with page 0 and continues fetching until all computers are retrieved.
func (c *Client) GetInventoryAllComputers(ctx context.Context, filter string) ([]InventoryComputer, error) {
	var allComputers []InventoryComputer
	page := 0
	pageSize := 100

	for {
		result, err := c.GetInventoryComputers(ctx, page, pageSize, filter)
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
