# MQL (Mondoo Query Language) Samples

This document contains useful MQL patterns and examples extracted from various query packs to help learn MQL syntax and capabilities.

## Basic Queries

<Example>
 <MQL>

```mql
# Get OS uptime
os.uptime

# Get all installed packages
packages

# Get DNS parameters
dns.params
```

 </MQL>
 <Description>
Simple resource access patterns that return entire objects or their primary properties.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Get specific asset information
asset { platform title version arch }

# Get kernel version specifically
kernel.info["version"]

# Get Terraform state version
terraform.state.terraformVersion
```

 </MQL>
 <Description>
Accessing object properties using projection braces and dictionary-style lookups for targeted data retrieval.
 </Description>

</Example>

## Asset and Platform Information

<Example>
 <MQL>

```mql
# Check if platform is Windows
asset.platform == "windows"

# Check if platform is macOS
asset.platform == "macos"

# Check if asset family contains Linux
asset.family.contains("linux")

# Multiple platform conditions
asset.platform == "okta" || asset.platform == "okta-org"
```

 </MQL>
 <Description>
Platform detection queries used to scope evaluations to compatible operating systems or services.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Check if system supports running commands
mondoo.capabilities.contains("run-command")

# Conditional execution based on capabilities
if ( mondoo.capabilities.contains('run-command') ) {
  ports.listening {
    protocol
    address
    port
  }
}
```

 </MQL>
 <Description>
Capability checking that guards command execution behind feature detection.
 </Description>

</Example>

## Filtering and Conditions

<Example>
 <MQL>

```mql
# Filter running services
services.where(running == true)

# Filter users excluding system accounts
users.where( name != /^_/ && shell != /\/usr\/bin\/false/ )

# Filter packages by SSL-related names
packages.where(name == /ssl/)

# Filter Terraform resources by type
terraform.state.resources.any( type == /^aws_/ )
```

 </MQL>
 <Description>
Where clauses that filter collections with equality and regex comparisons.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Multiple conditions with logical operators
services.where(running == true) { name running enabled masked type }

# Regex matching for exclusion
users.where( name != /^_/ && shell != /\/usr\/bin\/false/ )

# Check if any item matches condition
terraform.state.resources.any( type == /^google_/ )
```

 </MQL>
 <Description>
Complex filtering using logical operators, projections, and `any()` membership checks.
 </Description>

</Example>

## Data Structures and Field Selection

<Example>
 <MQL>

```mql
# Select specific fields from processes
processes.list { pid command }

# Select fields from mount points
mount.list { path fstype device options }

# Select fields from kernel modules
kernel.modules { name loaded }

# Select fields with nested access
terraform.state.resources { type providerName values['arn'] values['owner_id'] }
```

 </MQL>
 <Description>
Field projection examples that limit returned data to only the properties you need.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Access nested dictionary values
values['arn']
values['owner_id']
values['project']

# Access array elements
dns.params.A.rData.first

# Parse and access plist data
parse.plist('/Library/Preferences/com.apple.SoftwareUpdate.plist').params['RecommendedUpdates']
```

 </MQL>
 <Description>
Nested object access techniques using bracket notation, dot notation, and external data parsing.
 </Description>

</Example>

## String Manipulation and Processing

<Example>
 <MQL>

```mql
# Split strings and access elements
dns.params.A.rData.first.split(".")[3]
dns.params.A.rData.first.split(".")[2]
dns.params.A.rData.first.split(".")[1]
dns.params.A.rData.first.split(".")[0]

# String concatenation for reverse DNS
reverseDNSDomain =
  dns.params.A.rData.first.split(".")[3] + "."
    + dns.params.A.rData.first.split(".")[2] + "."
    +  dns.params.A.rData.first.split(".")[1] + "."
    +  dns.params.A.rData.first.split(".")[0]
    + ".in-addr.arpa"
```

 </MQL>
 <Description>
String operations that split hostnames into components and recombine them into reverse DNS entries.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Match patterns at beginning of string
name != /^_/

# Match file extensions or patterns
type == /^aws_/
type == /^google_/
type == /^azurerm_/

# Pattern matching for SSL packages
packages.where(name == /ssl/)
```

 </MQL>
 <Description>
Regular expression usage for pattern matching across asset names, resource types, and package lists.
 </Description>

</Example>

## Network and DNS Queries

<Example>
 <MQL>

```mql
# Basic DNS parameter access
dns.params

# Access specific record types
dns.params.MX.name
dns.mx { domainName preference }

# Query specific DNS entries
dns("_dmarc."+domainName.fqdn).params.TXT
dns(reverseDNSDomain).params.PTR

# Access record data
dns.params.TXT
dns.params.A.rData.first
```

 </MQL>
 <Description>
DNS query examples accessing multiple record types and dynamically constructed lookups.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Get listening ports
ports.listening

# Get ports with details
ports.listening {
  protocol
  address
  port
}
```

 </MQL>
 <Description>
Network information queries that enumerate listening sockets with optional field projections.
 </Description>

</Example>

## Properties and Variables

<Example>
 <MQL>

```mql
props:
  - uid: mondooEmailSecurityDkimSelectors
    title: Define a list of valid DKIM selectors
    mql: |
      [
        "google",
        "selector1",
        "selector2",
        "k1",
        "dkim",
        "mx",
        "mailjet"
      ]
```

 </MQL>
 <Description>
Property definition that introduces reusable configuration values through props.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Reference property in query
props.mondooEmailSecurityDkimSelectors{ dns(_+"._domainkey."+domainName.fqdn).params['TXT'] }
```

 </MQL>
 <Description>
Using properties during iteration, with `_` representing the current selector value.
 </Description>

</Example>

## Advanced Patterns

<Example>
 <MQL>

```mql
mql: |
  asset {
    platform
    version
    arch
  }
```

 </MQL>
 <Description>
Multi-line query definition using YAML block scalars for readability.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Basic variable assignment
mql: |
  reverseDNSDomain =
    dns.params.A.rData.first.split(".")[3] + "."
      + dns.params.A.rData.first.split(".")[2] + "."
      +  dns.params.A.rData.first.split(".")[1] + "."
      +  dns.params.A.rData.first.split(".")[0]
      + ".in-addr.arpa"
  dns(reverseDNSDomain).params.PTR

# Advanced variable usage in filters
aws_account = aws.account.id
aws.iam.policies.where( arn.contains(aws_account)).length
```

 </MQL>
 <Description>
Variable assignment and reuse that avoid repeated API calls and improve readability.
 </Description>

</Example>

<Example>
 <MQL>

```mql
mql: |
  if ( mondoo.capabilities.contains('run-command') ) {
    ports.listening {
      protocol
      address
      port
    }
  }
```

 </MQL>
 <Description>
Conditional execution pattern that runs nested queries when capabilities permit.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Iterate over property arrays
props.mondooEmailSecurityDkimSelectors{ dns(_+"._domainkey."+domainName.fqdn).params['TXT'] }
```

 </MQL>
 <Description>
Collection iteration leveraging property arrays with `_` as the iterator variable.
 </Description>

</Example>

## Cloud-Specific Advanced Patterns

<Example>
 <MQL>

```mql
# Azure VM storage configuration
azure.subscription.compute.vms.where( properties["storageProfile"]["osDisk"]["managedDisk"] != empty )

# Mixed property access styles
azure.subscription.network.interfaces{
  name location
  properties['nicType']
  properties['macAddress']
  properties['virtualMachine']['id']
}
```

 </MQL>
 <Description>
Deep nested property access examples that combine direct fields with bracket access.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Multiple OR conditions
aws.iam.credentialReport.where( accessKey1Active || accessKey2Active )

# Case-insensitive regex with additional conditions
aws.iam.policies.where( name == /FullAccess/i && attachmentCount != 0)

# Exclusion filtering for accurate counts
aws.autoscaling.groups.where( name != "mondoo-scanning-asg" ).length
```

 </MQL>
 <Description>
Complex boolean logic in filters leveraging OR, case-insensitive regex, and exclusions.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Query both legacy and flexible PostgreSQL servers
azure.subscription.postgreSql.servers
azure.subscription.postgreSql.flexibleServers

# Get firewall rules from both types
azure.subscription.postgreSql.servers { firewallRules }
azure.subscription.postgreSql.flexibleServers { firewallRules }
```

 </MQL>
 <Description>
Multi-resource type queries targeting legacy and flexible PostgreSQL offerings in Azure.
 </Description>

</Example>

<Example>
 <MQL>

```mql
filters: |
  asset.platform == "aws-iam-user"
  aws.iam.attachedPolicies
    .where(arn == "arn:aws:iam::aws:policy/AdministratorAccess")
    .any(attachedUsers
      .contains(
        arn.in(asset.ids)
      )
    )
```

 </MQL>
 <Description>
Complex asset filtering that combines platform filters with relationship traversal and containment checks.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Complex nested selection with multiple relationships
aws.ec2.instances.where( publicIp != '' ) {
  arn
  instanceId
  region
  state
  vpc.id
  keypair {
    name
  }
  securityGroups {
    name
    description
    ipPermissions
  }
  tags
}
```

 </MQL>
 <Description>
Advanced object traversal selecting instance metadata, network relationships, and nested collections.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# PowerShell integration with JSON parsing
parse.json(content: powershell('$time = (Get-Date).Adddays(-(180));Get-ADComputer -Filter {LastLogonTimeStamp -ge $time} -properties * | select Name,Enabled,OperatingSystem,OperatingSystemVersion,LastLogonDate | ConvertTo-Json').stdout).params
```

 </MQL>
 <Description>
External system integration combining PowerShell execution with JSON parsing for Active Directory data.
 </Description>

</Example>

<Example>
 <MQL>

```mql
# Filter with empty checks before execution
filters: asset.platform == "azure-storage-account" && azure.subscription.storage.account.containers != empty
mql: azure.subscription.storage.account.containers
```

 </MQL>
 <Description>
Conditional filtering with state checks to avoid running queries on empty container lists.
 </Description>

</Example>

## Query Pack Structure

<Example>
 <MQL>

```yaml
packs:
  - uid: unique-pack-identifier
    name: Human Readable Pack Name
    version: 1.0.0
    license: BUSL-1.1
    authors:
      - name: Author Name
        email: author@example.com
    tags:
      mondoo.com/platform: target-platform
      mondoo.com/category: category
    filters:
      - asset.platform == "target-platform"
    queries:
      - uid: query-identifier
        title: Query Description
        mql: query-expression
```

 </MQL>
 <Description>
Basic query pack structure showing metadata, filters, and query definitions.
 </Description>

</Example>

<Example>
 <MQL>

```yaml
queries:
  - uid: parent-query
    title: Main Query
    variants:
      - uid: variant-1
      - uid: variant-2
  - uid: variant-1
    filters: specific-condition-1
    mql: variant-specific-query-1
  - uid: variant-2
    filters: specific-condition-2
    mql: variant-specific-query-2
```

 </MQL>
 <Description>
Query variant structure illustrating specialized implementations for different conditions.
 </Description>

</Example>

## Best Practices

### Performance and Optimization
1. **Always use filters**: Target queries to appropriate platforms using filters
2. **Check capabilities**: Use capability checks for system-dependent operations
3. **Project fields**: Select only needed fields to optimize performance
4. **Check for empty data**: Use `!= empty` checks in filters to prevent unnecessary query execution
5. **Use variables**: Store intermediate results in variables to avoid repeated API calls

### Pattern Matching and Filtering
6. **Use regex carefully**: Regular expressions should be specific to avoid false matches
7. **Use case-insensitive matching**: Use `/pattern/i` for flexible text matching
8. **Combine conditions effectively**: Use `&&` and `||` for complex boolean logic
9. **Exclude system resources**: Filter out monitoring/system resources for accurate counts

### Query Organization
10. **Document queries**: Include meaningful titles and descriptions for all queries
11. **Use variants**: Create specialized query versions for different conditions
12. **Group related queries**: Organize queries by function or resource type
13. **Handle multiple resource types**: Query both legacy and current versions of services
