// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package ocsf holds the subset of the Open Cybersecurity Schema Framework that
// cnspec emits, plus the two writers for it: newline-delimited JSON and Apache
// Parquet.
//
// The structs are shared by both writers, which is why every field carries a
// `json` and a `parquet` tag. Two rules keep that workable:
//
//   - No `any`/`interface{}` fields. The parquet schema is derived from the Go
//     types by reflection and cannot type an empty interface. Free-form values
//     (MQL results, assessments) are JSON-encoded into strings or into the
//     `unmapped` map.
//   - Optional columns are written as null when the Go value is its zero value,
//     which lines up with `omitempty` on the JSON side.
//
// The structs cover the union of the supported OCSF versions (see version.go).
// Fields that only exist in a newer version are left empty when an older
// version is selected; in parquet they show up as an all-null column, which
// Glue/Athena and every other reader handle fine.
package ocsf

// Event class identifiers. class_uid = category_uid * 1000 + class id,
// type_uid = class_uid * 100 + activity_id.
const (
	CategoryFindings  = 2
	CategoryDiscovery = 5

	ClassUIDComplianceFinding    = 2003
	ClassUIDVulnerabilityFinding = 2002
	ClassUIDInventoryInfo        = 5001

	// ClassComplianceFinding and friends name the event class in file names and
	// in the writer API.
	ClassComplianceFinding    = "compliance_finding"
	ClassVulnerabilityFinding = "vulnerability_finding"
	ClassInventoryInfo        = "inventory_info"
)

// Finding activity, from the `finding` base class: 1 Create, 2 Update, 3 Close.
const (
	ActivityCreate = 1
)

// Discovery activity, from the `discovery` base class: 1 Log, 2 Collect.
const (
	ActivityCollect = 2
)

// Severity, from the OCSF dictionary.
const (
	SeverityUnknown       = 0
	SeverityInformational = 1
	SeverityLow           = 2
	SeverityMedium        = 3
	SeverityHigh          = 4
	SeverityCritical      = 5
	SeverityFatal         = 6
	SeverityOther         = 99
)

// Finding status, from the `finding` base class.
const (
	StatusUnknown    = 0
	StatusNew        = 1
	StatusInProgress = 2
	StatusSuppressed = 3
	StatusResolved   = 4
	StatusOther      = 99
)

// Compliance status, from the `compliance` object.
const (
	ComplianceStatusUnknown = 0
	ComplianceStatusPass    = 1
	ComplianceStatusWarning = 2
	ComplianceStatusFail    = 3
	ComplianceStatusOther   = 99
)

// Device type, from the `endpoint` object.
const (
	DeviceTypeUnknown = 0
	DeviceTypeServer  = 1
	DeviceTypeVirtual = 6
	DeviceTypeBrowser = 8
	DeviceTypeOther   = 99
)

// OS type, from the `os` object.
const (
	OSTypeUnknown = 0
	OSTypeWindows = 100
	OSTypeLinux   = 200
	OSTypeAndroid = 201
	OSTypeMacOS   = 300
	OSTypeSolaris = 400
	OSTypeAIX     = 401
	OSTypeHPUX    = 402
	OSTypeOther   = 99
)

var severityNames = map[int]string{
	SeverityUnknown:       "Unknown",
	SeverityInformational: "Informational",
	SeverityLow:           "Low",
	SeverityMedium:        "Medium",
	SeverityHigh:          "High",
	SeverityCritical:      "Critical",
	SeverityFatal:         "Fatal",
	SeverityOther:         "Other",
}

var statusNames = map[int]string{
	StatusUnknown:    "Unknown",
	StatusNew:        "New",
	StatusInProgress: "In Progress",
	StatusSuppressed: "Suppressed",
	StatusResolved:   "Resolved",
	StatusOther:      "Other",
}

var complianceStatusNames = map[int]string{
	ComplianceStatusUnknown: "Unknown",
	ComplianceStatusPass:    "Pass",
	ComplianceStatusWarning: "Warning",
	ComplianceStatusFail:    "Fail",
	ComplianceStatusOther:   "Other",
}

// SeverityName renders the label that belongs next to a severity_id.
func SeverityName(id int) string { return lookupName(severityNames, id) }

// StatusName renders the label that belongs next to a finding status_id.
func StatusName(id int) string { return lookupName(statusNames, id) }

// ComplianceStatusName renders the label that belongs next to a
// compliance.status_id.
func ComplianceStatusName(id int) string { return lookupName(complianceStatusNames, id) }

func lookupName(names map[int]string, id int) string {
	if name, ok := names[id]; ok {
		return name
	}
	return "Other"
}

// base carries the attributes every OCSF event has, from `base_event` and the
// classification include. It is embedded, which both encoding/json and
// parquet-go flatten into the parent event.
type base struct {
	ActivityID   int    `json:"activity_id" parquet:"activity_id"`
	ActivityName string `json:"activity_name,omitempty" parquet:"activity_name,optional"`
	CategoryUID  int    `json:"category_uid" parquet:"category_uid"`
	CategoryName string `json:"category_name,omitempty" parquet:"category_name,optional"`
	ClassUID     int    `json:"class_uid" parquet:"class_uid"`
	ClassName    string `json:"class_name,omitempty" parquet:"class_name,optional"`
	TypeUID      int    `json:"type_uid" parquet:"type_uid"`
	TypeName     string `json:"type_name,omitempty" parquet:"type_name,optional"`

	// Time is the event time in milliseconds since the epoch.
	Time         int64  `json:"time" parquet:"time"`
	SeverityID   int    `json:"severity_id" parquet:"severity_id"`
	Severity     string `json:"severity,omitempty" parquet:"severity,optional"`
	StatusID     int    `json:"status_id,omitempty" parquet:"status_id,optional"`
	Status       string `json:"status,omitempty" parquet:"status,optional"`
	StatusCode   string `json:"status_code,omitempty" parquet:"status_code,optional"`
	StatusDetail string `json:"status_detail,omitempty" parquet:"status_detail,optional"`
	Message      string `json:"message,omitempty" parquet:"message,optional"`

	Metadata Metadata          `json:"metadata" parquet:"metadata"`
	Unmapped map[string]string `json:"unmapped,omitempty" parquet:"unmapped"`
}

// Metadata describes the event's producer and the schema it follows.
type Metadata struct {
	Version    string   `json:"version" parquet:"version"`
	Product    Product  `json:"product" parquet:"product"`
	LoggedTime int64    `json:"logged_time,omitempty" parquet:"logged_time,optional"`
	Profiles   []string `json:"profiles,omitempty" parquet:"profiles,list"`
	EventCode  string   `json:"event_code,omitempty" parquet:"event_code,optional"`
	Labels     []string `json:"labels,omitempty" parquet:"labels,list"`
}

// Product identifies the tool that produced the event.
type Product struct {
	Name       string `json:"name,omitempty" parquet:"name,optional"`
	VendorName string `json:"vendor_name,omitempty" parquet:"vendor_name,optional"`
	Version    string `json:"version,omitempty" parquet:"version,optional"`
	URLString  string `json:"url_string,omitempty" parquet:"url_string,optional"`
}

// FindingInfo carries the identity and documentation of a finding.
type FindingInfo struct {
	UID           string   `json:"uid" parquet:"uid"`
	Title         string   `json:"title" parquet:"title"`
	Desc          string   `json:"desc,omitempty" parquet:"desc,optional"`
	CreatedTime   int64    `json:"created_time,omitempty" parquet:"created_time,optional"`
	FirstSeenTime int64    `json:"first_seen_time,omitempty" parquet:"first_seen_time,optional"`
	ModifiedTime  int64    `json:"modified_time,omitempty" parquet:"modified_time,optional"`
	ProductUID    string   `json:"product_uid,omitempty" parquet:"product_uid,optional"`
	SrcURL        string   `json:"src_url,omitempty" parquet:"src_url,optional"`
	Types         []string `json:"types,omitempty" parquet:"types,list"`
	DataSources   []string `json:"data_sources,omitempty" parquet:"data_sources,list"`
}

// Compliance ties a finding to the frameworks and controls it evaluates.
type Compliance struct {
	Standards    []string `json:"standards" parquet:"standards,list"`
	Control      string   `json:"control,omitempty" parquet:"control,optional"`
	Requirements []string `json:"requirements,omitempty" parquet:"requirements,list"`
	Status       string   `json:"status,omitempty" parquet:"status,optional"`
	StatusID     int      `json:"status_id" parquet:"status_id"`
	StatusCode   string   `json:"status_code,omitempty" parquet:"status_code,optional"`
	StatusDetail string   `json:"status_detail,omitempty" parquet:"status_detail,optional"`

	// Category and Desc were added in OCSF 1.9 and stay empty on older versions.
	Category string `json:"category,omitempty" parquet:"category,optional"`
	Desc     string `json:"desc,omitempty" parquet:"desc,optional"`
}

// ResourceDetails describes the thing a finding is about.
type ResourceDetails struct {
	UID            string   `json:"uid,omitempty" parquet:"uid,optional"`
	Name           string   `json:"name,omitempty" parquet:"name,optional"`
	Type           string   `json:"type,omitempty" parquet:"type,optional"`
	Labels         []string `json:"labels,omitempty" parquet:"labels,list"`
	CloudPartition string   `json:"cloud_partition,omitempty" parquet:"cloud_partition,optional"`
	Region         string   `json:"region,omitempty" parquet:"region,optional"`
	Namespace      string   `json:"namespace,omitempty" parquet:"namespace,optional"`
	Version        string   `json:"version,omitempty" parquet:"version,optional"`
}

// Remediation describes how to fix a finding.
type Remediation struct {
	Desc       string   `json:"desc" parquet:"desc"`
	References []string `json:"references,omitempty" parquet:"references,list"`
}

// Cloud describes the cloud environment an asset lives in.
type Cloud struct {
	Provider   string   `json:"provider" parquet:"provider"`
	Account    *Account `json:"account,omitempty" parquet:"account,optional"`
	Region     string   `json:"region,omitempty" parquet:"region,optional"`
	Zone       string   `json:"zone,omitempty" parquet:"zone,optional"`
	ProjectUID string   `json:"project_uid,omitempty" parquet:"project_uid,optional"`
}

// Account is the cloud account an asset belongs to.
type Account struct {
	UID  string `json:"uid,omitempty" parquet:"uid,optional"`
	Name string `json:"name,omitempty" parquet:"name,optional"`
	Type string `json:"type,omitempty" parquet:"type,optional"`
}

// Device describes the scanned asset as an endpoint.
type Device struct {
	TypeID   int           `json:"type_id" parquet:"type_id"`
	Type     string        `json:"type,omitempty" parquet:"type,optional"`
	UID      string        `json:"uid,omitempty" parquet:"uid,optional"`
	Name     string        `json:"name,omitempty" parquet:"name,optional"`
	Hostname string        `json:"hostname,omitempty" parquet:"hostname,optional"`
	Domain   string        `json:"domain,omitempty" parquet:"domain,optional"`
	Region   string        `json:"region,omitempty" parquet:"region,optional"`
	OS       *OS           `json:"os,omitempty" parquet:"os,optional"`
	HwInfo   *HardwareInfo `json:"hw_info,omitempty" parquet:"hw_info,optional"`
}

// OS is the operating system of a device.
type OS struct {
	Name    string `json:"name" parquet:"name"`
	TypeID  int    `json:"type_id" parquet:"type_id"`
	Type    string `json:"type,omitempty" parquet:"type,optional"`
	Version string `json:"version,omitempty" parquet:"version,optional"`
	Build   string `json:"build,omitempty" parquet:"build,optional"`
}

// HardwareInfo carries the hardware details cnspec knows about.
type HardwareInfo struct {
	CPUArchitecture string `json:"cpu_architecture,omitempty" parquet:"cpu_architecture,optional"`
}

// Vulnerability describes one advisory affecting an asset.
type Vulnerability struct {
	Title            string            `json:"title,omitempty" parquet:"title,optional"`
	Desc             string            `json:"desc,omitempty" parquet:"desc,optional"`
	Severity         string            `json:"severity,omitempty" parquet:"severity,optional"`
	CVE              *CVE              `json:"cve,omitempty" parquet:"cve,optional"`
	AffectedPackages []AffectedPackage `json:"affected_packages,omitempty" parquet:"affected_packages,list"`
	References       []string          `json:"references,omitempty" parquet:"references,list"`
	IsFixAvailable   bool              `json:"is_fix_available" parquet:"is_fix_available"`
	FirstSeenTime    int64             `json:"first_seen_time,omitempty" parquet:"first_seen_time,optional"`
	LastSeenTime     int64             `json:"last_seen_time,omitempty" parquet:"last_seen_time,optional"`
	VendorName       string            `json:"vendor_name,omitempty" parquet:"vendor_name,optional"`
	Remediation      *Remediation      `json:"remediation,omitempty" parquet:"remediation,optional"`
}

// CVE is a single CVE record of a vulnerability.
type CVE struct {
	UID          string   `json:"uid" parquet:"uid"`
	Title        string   `json:"title,omitempty" parquet:"title,optional"`
	Desc         string   `json:"desc,omitempty" parquet:"desc,optional"`
	CreatedTime  int64    `json:"created_time,omitempty" parquet:"created_time,optional"`
	ModifiedTime int64    `json:"modified_time,omitempty" parquet:"modified_time,optional"`
	CVSS         []CVSS   `json:"cvss,omitempty" parquet:"cvss,list"`
	References   []string `json:"references,omitempty" parquet:"references,list"`
}

// CVSS is a CVSS score of a CVE.
type CVSS struct {
	Version      string  `json:"version" parquet:"version"`
	BaseScore    float64 `json:"base_score" parquet:"base_score"`
	Severity     string  `json:"severity,omitempty" parquet:"severity,optional"`
	VectorString string  `json:"vector_string,omitempty" parquet:"vector_string,optional"`
}

// AffectedPackage is a package that a vulnerability applies to.
type AffectedPackage struct {
	Name           string `json:"name" parquet:"name"`
	Version        string `json:"version" parquet:"version"`
	Architecture   string `json:"architecture,omitempty" parquet:"architecture,optional"`
	PURL           string `json:"purl,omitempty" parquet:"purl,optional"`
	FixedInVersion string `json:"fixed_in_version,omitempty" parquet:"fixed_in_version,optional"`
	PackageManager string `json:"package_manager,omitempty" parquet:"package_manager,optional"`
}

// ComplianceFinding is OCSF class 2003: the outcome of one cnspec check on one
// asset.
type ComplianceFinding struct {
	base
	Compliance  Compliance        `json:"compliance" parquet:"compliance"`
	FindingInfo FindingInfo       `json:"finding_info" parquet:"finding_info"`
	Remediation *Remediation      `json:"remediation,omitempty" parquet:"remediation,optional"`
	Resources   []ResourceDetails `json:"resources,omitempty" parquet:"resources,list"`
	Device      *Device           `json:"device,omitempty" parquet:"device,optional"`
	Cloud       *Cloud            `json:"cloud,omitempty" parquet:"cloud,optional"`
}

// VulnerabilityFinding is OCSF class 2002: one advisory affecting one asset.
type VulnerabilityFinding struct {
	base
	FindingInfo     FindingInfo       `json:"finding_info" parquet:"finding_info"`
	Vulnerabilities []Vulnerability   `json:"vulnerabilities" parquet:"vulnerabilities,list"`
	Resources       []ResourceDetails `json:"resources,omitempty" parquet:"resources,list"`
	Device          *Device           `json:"device,omitempty" parquet:"device,optional"`
	Cloud           *Cloud            `json:"cloud,omitempty" parquet:"cloud,optional"`
}

// InventoryInfo is OCSF class 5001: the asset cnspec scanned, plus whatever
// data-only queries collected about it.
type InventoryInfo struct {
	base
	Device Device `json:"device" parquet:"device"`
	Cloud  *Cloud `json:"cloud,omitempty" parquet:"cloud,optional"`
}
