# Azure MQL Samples

MQL patterns for Azure resources. Use these as reference when implementing Azure-specific security queries.

## Resource Hierarchy

```
azure.subscription
├── compute
│   ├── vms
│   └── disks
├── storage
│   ├── accounts
│   └── account (single)
├── network
│   ├── interfaces
│   ├── securityGroups
│   └── virtualNetworks
├── web
│   ├── appServices
│   └── appService (single)
├── cloudDefender
│   ├── securityContacts
│   └── monitoringAgentAutoProvision
├── iam
│   └── roles
├── sql
│   ├── servers
│   └── databases
├── postgreSql
│   ├── servers (legacy)
│   └── flexibleServers
└── keyvault
    └── vaults
```

## Platform Filters

```mql
# Full subscription scan
filters: |
  asset.platform == "azure"

# Single resource scans
filters: |
  asset.platform == "azure-storage-account"

filters: |
  asset.platform == "azure-vm"

filters: |
  asset.platform == "azure-sql-server"
```

## Storage Account Patterns

### Check Property Across All Accounts
```mql
# Ensure HTTPS traffic only
azure.subscription.storage.accounts.all(properties.EnableHTTPSTrafficOnly == true)

# Ensure blob public access disabled
azure.subscription.storage.accounts.all(properties.AllowBlobPublicAccess == "false")

# Ensure network rules deny by default
azure.subscription.storage.accounts.all(properties.NetworkRuleSet.defaultAction == "Deny")

# Ensure private endpoints configured
azure.subscription.storage.accounts.all(properties.PrivateEndpointConnections != empty)
```

### Single Storage Account (Granular Scan)
```mql
azure.subscription.storage.account.properties.EnableHTTPSTrafficOnly == true
azure.subscription.storage.account.properties.AllowBlobPublicAccess == "false"
azure.subscription.storage.account.properties.NetworkRuleSet.defaultAction == "Deny"
```

### Data Protection Settings
```mql
azure.subscription.storage.accounts.all(dataProtection.blobSoftDeletionEnabled == true)
azure.subscription.storage.accounts.all(dataProtection.blobRetentionDays > 0)
azure.subscription.storage.accounts.all(dataProtection.containerSoftDeletionEnabled == true)
azure.subscription.storage.accounts.all(dataProtection.containerRetentionDays > 0)
```

### Container Access
```mql
# Filter before checking containers
filters: asset.platform == "azure-storage-account" && azure.subscription.storage.account.containers != empty
mql: azure.subscription.storage.account.containers.all(properties.publicAccess == "None")
```

## VM Patterns

### VM Configuration Checks
```mql
# Check VM storage configuration
azure.subscription.compute.vms.where(properties["storageProfile"]["osDisk"]["managedDisk"] != empty)

# Check all VMs have specific property
azure.subscription.compute.vms.all(properties["osProfile"]["linuxConfiguration"]["disablePasswordAuthentication"] == true)
```

### Network Interface Details
```mql
azure.subscription.network.interfaces {
  name
  location
  properties['nicType']
  properties['macAddress']
  properties['virtualMachine']['id']
}
```

## Cloud Defender / Security Center

```mql
# Auto-provisioning enabled
azure.subscription.cloudDefender.monitoringAgentAutoProvision == true

# Security contacts configured
azure.subscription.cloudDefender.securityContacts != empty
azure.subscription.cloudDefender.securityContacts.all(notificationsByRole.roles.contains('Owner'))
azure.subscription.cloudDefender.securityContacts.all(notificationsByRole.state == 'On')

# Email notification configured
azure.subscription.cloudDefender.securityContacts.all(emails.none(_ == empty))

# Alert severity configuration
azure.subscription.cloudDefender.securityContacts.all(alertNotifications.minimalSeverity == "High")
azure.subscription.cloudDefender.securityContacts.all(alertNotifications.state == "On")
```

## IAM / Role Patterns

```mql
# No custom subscription admin roles
azure.subscription.iam.roles.where(permissions.any(allowedActions.any(_ == "*"))).all(scopes.none(_ == /subscriptions/))
```

## SQL / Database Patterns

### PostgreSQL Servers
```mql
# Check both legacy and flexible servers
azure.subscription.postgreSql.servers
azure.subscription.postgreSql.flexibleServers

# Get firewall rules from both types
azure.subscription.postgreSql.servers { firewallRules }
azure.subscription.postgreSql.flexibleServers { firewallRules }
```

## App Service Patterns

### Web App Configuration
```mql
# Check all app services
azure.subscription.web.appServices.all(properties.httpsOnly == true)
azure.subscription.web.appServices.all(properties.clientCertEnabled == true)

# Check site config
azure.subscription.web.appServices.all(configuration.properties.minTlsVersion == "1.2")
azure.subscription.web.appServices.all(configuration.properties.remoteDebuggingEnabled == false)

# Authentication enabled
azure.subscription.web.appServices.all(authSettings.properties.enabled == true)
```

### Single App Service (Granular Scan)
```mql
azure.subscription.web.appService.properties.httpsOnly == true
azure.subscription.web.appService.configuration.properties.remoteDebuggingEnabled == false
```

## Variants Pattern

Use variants when a check applies to both full subscription scans AND single resource scans:

```yaml
- uid: azure-storage-secure-transfer
  title: Ensure that 'Secure transfer required' is set to 'Enabled'
  impact: 80
  variants:
    - uid: azure-storage-secure-transfer-api
    - uid: azure-storage-secure-transfer-single

- uid: azure-storage-secure-transfer-api
  filters: |
    asset.platform == "azure"
  mql: |
    azure.subscription.storage.accounts.all(properties.EnableHTTPSTrafficOnly == true)

- uid: azure-storage-secure-transfer-single
  filters: |
    asset.platform == "azure-storage-account"
  mql: |
    azure.subscription.storage.account.properties.EnableHTTPSTrafficOnly == true
```

## Common MQL Patterns

### Check All Resources
```mql
azure.subscription.{resource}.all({condition})
```

### Filter Then Check
```mql
azure.subscription.{resource}.where({filter}).all({condition})
```

### Get Specific Properties
```mql
azure.subscription.{resource} { field1 field2 properties['nestedKey'] }
```

### Nested Property Access
```mql
# Bracket notation for dynamic keys
properties["storageProfile"]["osDisk"]["managedDisk"]

# Mixed notation
properties.NetworkRuleSet.defaultAction
```

### Empty Checks
```mql
# Resource exists
azure.subscription.storage.accounts != empty

# Property not empty
properties.PrivateEndpointConnections != empty

# Collection not empty with filter
azure.subscription.storage.account.containers != empty
```

## Resource Type Regex Matching

```mql
# Match Azure resource types in Terraform
type == /^azurerm_/
```
