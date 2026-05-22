# MQL (Mondoo Query Language) Development Context

## Overview
MQL is Mondoo's domain-specific language for security and compliance queries. This context provides patterns, best practices, and examples for writing effective MQL policies and queries.

## Core MQL Concepts

### Lexer and Token Structure

MQL uses the following lexer regex pattern:
```regex
/(\s+)|(?P<Ident>[a-zA-Z$_][a-zA-Z0-9_]*)|(?P<Float>[-+]?\d*\.\d+([eE][-+]?\d+)?)|(?P<Int>[-+]?\d+([eE][-+]?\d+)?)|(?P<String>'[^']*'|"[^"]*")|(?P<Comment>(//|#)[^\n]*(\n|\z))|(?P<Regex>/([^\\/]+|\\.)+/[msi]*)|(?P<Op>[-+*/%,:.=<>!|&~;])|(?P<Call>[(){}\[\]])/
```

**Token Types Breakdown:**

1. **Whitespace**: `(\s+)` - Spaces, tabs, newlines
2. **Identifiers**: `[a-zA-Z$_][a-zA-Z0-9_]*` - Variable names, resource names, properties
   - Must start with letter, `$`, or `_`
   - Can contain letters, numbers, and underscores
   - Examples: `file`, `user_name`, `$variable`, `_private`

3. **Float Numbers**: `[-+]?\d*\.\d+([eE][-+]?\d+)?`
   - Optional sign, optional digits before decimal, required decimal point and digits after
   - Optional scientific notation
   - Examples: `3.14`, `-0.5`, `1.23e-4`, `+2.5E+3`

4. **Integer Numbers**: `[-+]?\d+([eE][-+]?\d+)?`
   - Optional sign, required digits
   - Optional scientific notation
   - Examples: `42`, `-10`, `+5`, `1e3`

5. **Strings**: `'[^']*'|"[^"]*"`
   - Single or double quoted
   - Examples: `"hello"`, `'world'`, `"/etc/passwd"`

6. **Comments**: `(//|#)[^\n]*(\n|\z)`
   - Line comments starting with `//` or `#`
   - Examples: `// This is a comment`, `# Another comment`

7. **Regular Expressions**: `/([^\\/]+|\\.)+/[msi]*`
   - Enclosed in forward slashes
   - Can contain escaped characters
   - Optional modifiers: `m` (multiline), `s` (single line), `i` (case insensitive)
   - Examples: `/pattern/`, `/case-insensitive/i`, `/multi.*line/m`

8. **Operators**: `[-+*/%,:.=<>!|&~;]`
   - Arithmetic: `+`, `-`, `*`, `/`, `%`
   - Comparison: `=`, `<`, `>`, `!`, `==`, `!=`, `<=`, `>=`
   - Logical: `|`, `&`, `!`

9. **Delimiters**: `[(){}\[\]]`
   - Parentheses: `(`, `)`
   - Braces: `{`, `}`
   - Brackets: `[`, `]`

**Important Lexer Rules:**
- Identifiers cannot start with numbers
- Regular expressions must be properly escaped
- Strings can use either single or double quotes
- Comments extend to end of line
- Operators can be single or multi-character (like `==`, `!=`)

### Query Structure
```mql
# Basic structure
resource.property == value

# With filtering
resources.where(condition).all(assertion)

# With data blocks
resource {
  property1
  property2 == expected_value
}
```

### Variables and Basic Syntax
```mql
# Variable definition
v = 23

# Null value
value = null

# Regular expression matching (NOT =~)
string == /pattern/

# Block notation for resource fields
sshd.config { ciphers content kexs macs }

# Empty checks
value == empty
value != empty
```

### Data Types and Operators

#### Empty Values
A value is considered empty when it is:
- `null`
- An empty array `[]`
- An empty map `{}`
- An empty string `""`

#### Dict Type
The `dict` type represents any JSON-compatible value:
- `null`
- Boolean (`true`/`false`)
- Number (int or float)
- String
- Array
- Map

#### String Membership
```mql
# Check if string exists in array
string.in(array)
```

#### Platform Access
```mql
# Use asset.platform instead of deprecated platform
asset.platform == "ubuntu"
```

### List Operations

#### Filtering with `.where()`
```mql
# Basic filtering
array.where(condition)

# Map filtering with key/value access
sshd.config.params.where(key == /pam/i)
sshd.config.params.where(value == "yes")
```

#### List Assertions
```mql
# All entries must match
array.all(condition)

# At least one entry matches
array.contains(condition)

# No entries match
array.none(condition)

# Exactly one entry matches
array.one(condition)
```

#### List Iterators
```mql
# Current item reference with underscore
array.where(_.contains("pattern"))
array.all(_.permissions.user_readable == true)

# Accessing properties of current item
files.where(_.exists).all(_.size > 0)
```

## MQL Best Practices & Style Guide

### 1. Avoid Unnecessary Variables
**Bad:**
```mql
proper = ["/etc/kubernetes/kubelet.conf", "/etc/kubernetes/kubeleta.conf"]
proper {
  if (file(_).exists) {
    files.find(from: _, type: 'file').all(permissions { other_readable == true })
  }
}
```

**Good:**
```mql
["/etc/kubernetes/kubelet.conf", "/etc/kubernetes/kubeleta.conf"]
  .where(file(_).exists)
  .all(file(_).permissions.other_readable == true)
```

### 2. Use `.where()` Instead of `if` Conditions
**Bad:** `if (condition) { ... }`
**Good:** `.where(condition)`

### 3. Avoid Blocks in `.all()` Expressions
**Bad:** `.all(permissions { other_readable == true })`
**Good:** `.all(permissions.other_readable == true)`

### 4. Direct File Access for Single Files
**Bad:** `files.find(from: path, type: 'file').all(...)`
**Good:** `file(path).property` (when checking single files)

### 5. Remove Unnecessary Spacing
**Bad:** `.where( file( _ ).exists )`
**Good:** `.where(file(_).exists)`

### 6. Atomic Logic for Actionable Output
**Bad:**
```mql
webhooks.all(config["url"] == /https/ || config["insecure_ssls"] == 0)
```

**Good:**
```mql
webhooks.all(config["url"] == /https/) ||
webhooks.all(config["insecure_ssls"] == 0)
```

### 7. Proper Null Handling
**Bad:** Assuming values are always present
**Good:** Check for null or empty values
```mql
# Check if value exists and meets condition
config.params.where(key == "setting" && value != null).all(value == "expected")

# Handle potentially null values
user.shell != null && user.shell != "/bin/false"
```

### 8. Use Correct Regular Expression Syntax
**Bad:** `string =~ /pattern/`
**Good:** `string == /pattern/`

### 9. Platform Detection
**Bad:** `platform == "ubuntu"`
**Good:** `asset.platform == "ubuntu"`

## Essential Design Patterns

### Pattern A: Multiple File Paths Check
```mql
props:
  - uid: configPaths
    title: Possible configuration file locations
    mql: |
      return [
        "/etc/config/app.conf",
        "/opt/app/config.conf",
        "/var/lib/app/app.conf"
      ]

mql: |
  props.configPaths.where(file(_).exists).all(
    file(_).permissions.other_readable == false &&
    file(_).user.name == "root"
  )
```

### Pattern B: Service Status Validation
```mql
props:
  - uid: restrictedServices
    title: Services that should be disabled
    mql: |
      return ["telnet", "rsh", "ftp"]

mql: |
  props.restrictedServices.where(package(_)).all(
    service(_).running == false &&
    service(_).enabled == false
  )
```

### Pattern C: File Existence with Conditional Checks
```mql
# Only check permissions if file exists
["/etc/cron.allow"].where(file(_).exists) {
  file(_) {
    user.name == 'root'
    group.name == 'root'
    permissions.user_readable == true
    permissions.user_writeable == true
    permissions.group_readable == false
    permissions.other_readable == false
  }
}
```

### Pattern D: Configuration File Parsing
```mql
# Preferred approach using plist relationships
parse.plist("/Library/Preferences/app.plist") {
  file.exists
  params['setting'] == expected_value
}

# Alternative with variable for multiple checks
config = parse.ini("/etc/app.conf")
config {
  file.exists
  params["section"]["key"] == "value"
}
```

### Pattern E: SSH Configuration Blocks
```mql
props:
  - uid: excludedMatchBlocks
    title: SSH match blocks to exclude from testing
    mql: |
      return ["User ansible", "User deploy"]
  - uid: checkDefaultBlock
    title: Whether to check default SSH block
    mql: return true

mql: |
  # Check user-defined blocks
  sshd.config.blocks
    .where(criteria != "" && criteria.contains(props.excludedMatchBlocks) == false)
    .all(params.PasswordAuthentication == 'no')

  # Check default block if enabled
  if (props.checkDefaultBlock) {
    sshd.config.blocks.where(criteria == "").all(params.PasswordAuthentication == 'no')
  }
```

### Pattern F: Map Key/Value Filtering
```mql
# Find SSH config parameters with case-insensitive key matching
sshd.config.params.where(key == /pam/i) {
  value == "yes"
}

# Check environment variables
env.where(key == /proxy/i).all(value != empty)

# Audit log configuration
auditd.config.params.where(key == "log_file").all(value == "/var/log/audit/audit.log")
```

### Pattern G: Empty and Null Value Handling
```mql
# Ensure required fields are not empty
users.where(name != "root").all(
  shell != null &&
  shell != empty &&
  shell != "/bin/false"
)

# Handle optional configuration
config = parse.ini("/etc/app.conf")
if (config.file.exists && config.params["section"] != null) {
  config.params["section"]["key"] == "value"
}
```

### Pattern H: Query Variants for Multiple Configurations
```yaml
# Parent query with variants for different architectures
queries:
  - uid: audit-time-change-events
    title: Audit time-change events
    variants:
      - uid: audit-time-change-32bit
      - uid: audit-time-change-64bit
      - uid: audit-time-change-comprehensive

  # 32-bit specific rules
  - uid: audit-time-change-32bit
    filters: |
      asset.family.contains("linux") &&
      (kernel.info['machine'] == 'i386' || kernel.info['machine'] == 'i686')
    mql: |
      auditd.rules.files.where(path == '/etc/localtime' && keyname == 'time-change').length > 0 &&
      auditd.rules.syscalls.where(
        fields.any(key == 'arch' && value == 'b32') &&
        syscalls.contains('adjtimex') &&
        keyname == 'time-change'
      ).length > 0

  # 64-bit specific rules
  - uid: audit-time-change-64bit
    filters: |
      asset.family.contains("linux") &&
      (kernel.info['machine'] == 'x86_64' || kernel.info['machine'].contains('64'))
    mql: |
      auditd.rules.files.where(path == '/etc/localtime' && keyname == 'time-change').length > 0 &&
      auditd.rules.syscalls.where(
        fields.any(key == 'arch' && value == 'b64') &&
        syscalls.contains('adjtimex') &&
        keyname == 'time-change'
      ).length > 0 &&
      auditd.rules.syscalls.where(
        fields.any(key == 'arch' && value == 'b32') &&
        syscalls.contains('stime') &&
        keyname == 'time-change'
      ).length > 0

  # Comprehensive check (if architecture detection is complex)
  - uid: audit-time-change-comprehensive
    filters: asset.family.contains("linux")
    mql: |
      machine_arch = kernel.info['machine']
      is_64bit = machine_arch == 'x86_64' || machine_arch.contains('64')

      localtime_rule = auditd.rules.files.where(
        path == '/etc/localtime' && keyname == 'time-change'
      ).length > 0

      if (is_64bit) {
        localtime_rule &&
        auditd.rules.syscalls.where(
          fields.any(key == 'arch' && value == 'b64') &&
          keyname == 'time-change'
        ).length > 0 &&
        auditd.rules.syscalls.where(
          fields.any(key == 'arch' && value == 'b32') &&
          keyname == 'time-change'
        ).length > 0
      } else {
        localtime_rule &&
        auditd.rules.syscalls.where(
          fields.any(key == 'arch' && value == 'b32') &&
          keyname == 'time-change'
        ).length > 0
      }
```

**When to Use Query Variants:**
- Different system architectures (32-bit vs 64-bit)
- Different OS distributions (Ubuntu vs CentOS vs RHEL)
- Different software versions (Apache 2.2 vs 2.4)
- Different cloud providers (AWS vs Azure vs GCP)
- Different deployment types (container vs VM vs bare metal)

**Benefits of Query Variants:**
1. **Cleaner Logic**: Each variant focuses on a specific scenario
2. **Better Performance**: Filters prevent execution on irrelevant systems
3. **Easier Maintenance**: Changes to one configuration don't affect others
4. **Clearer Reporting**: Results are categorized by system type
5. **Reduced Complexity**: Avoids nested conditional logic in MQL

### Pattern I: Time Service Configuration Check
```mql
# Modern approach - exactly one time service should be enabled and running
timeServices = [
  "chrony",
  "chronyd",
  "systemd-timesyncd"
]

timeServices.one(
  service(_).enabled &&
  service(_).running
)

# Platform-specific variants for older systems (Ubuntu 18, Debian 10)
timeServicesExtended = [
  "chrony",
  "chronyd",
  "ntp",
  "ntpd",
  "systemd-timesyncd"
]

timeServicesExtended.one(
  service(_).enabled &&
  service(_).running
)

# Alternative counter-based approach (if .one() doesn't work as expected)
active_count = 0
if (services.where(name == 'chrony' && enabled == true).length > 0) {
  active_count = active_count + 1
}
if (services.where(name == 'systemd-timesyncd' && enabled == true).length > 0) {
  active_count = active_count + 1
}
active_count == 1
```

**Key Points:**
- Use `.one()` to ensure exactly one service is active (prevents conflicts)
- Different service arrays for different platforms/requirements
- Avoid complex if-else logic in favor of array-based approaches
- Check both `enabled` and `running` states for services

## Common Resource Patterns

### File Operations
```mql
# File existence
file("/path").exists

# File content
file("/path").content.contains("pattern")

# File permissions (preferred over permissions.string)
file("/path").permissions {
  user_readable == true
  user_writeable == false
  group_readable == false
  other_readable == false
}

# Find files
files.find(from: "/path", type: "file", regex: "*.conf")

# Handle potentially missing files
["/etc/optional.conf"].where(file(_).exists).all(
  file(_).content == /required_pattern/
)
```

### User & Group Management
```mql
# User properties with null checking
users.where(name != "root" && shell != null).all(shell == "/bin/false")

# Group membership
groups.where(name == "sudo").all(members.none(name == "guest"))

# Shadow file
shadow.where(user.name == "admin").all(passwordChangeRequired == true)

# Check for disabled accounts
users.where(shell != null && shell.in(["/bin/false", "/usr/sbin/nologin"]))
```

### Package & Service Management
```mql
# Package installation
package("nginx").installed == true
package("apache2").version >= semver("2.4.0")

# Service status
service("ssh").running == true
service("telnet").enabled == false
service("rsh").masked == true

# Handle services that may not exist
["telnet", "rsh"].where(service(_) != null).all(
  service(_).enabled == false
)
```

### System Configuration
```mql
# Kernel parameters
kernel.parameters['net.ipv4.ip_forward'] == 0

# Kernel modules
kernel.module("dccp").loaded == false

# Mount points
mount.where(path == "/tmp").all(options.contains("noexec"))

# Process information
processes.where(executable == /nginx/).all(user.name == "www-data")
```

### Network & Security
```mql
# Listening ports
ports.listening.where(protocol == "tcp").all(port != 23)

# Firewall rules
iptables.input.contains("-p tcp --dport 22 -j ACCEPT")

# Audit configuration
command("auditctl -l").stdout.lines.any(contains("-w /etc/passwd"))
```

### Cloud Resources (AWS Example)
```mql
# S3 buckets
aws.s3.buckets.all(acl.private == true)

# EC2 instances
aws.ec2.instances.where(state == "running").all(
  securityGroups.all(
    rules.none(protocol == "tcp" && fromPort <= 22 && toPort >= 22 && cidr == "0.0.0.0/0")
  )
)

# IAM policies
aws.iam.policies.where(attachmentCount > 0).all(
  document.Statement.none(Effect == "Allow" && Action == "*" && Resource == "*")
)
```

### Kubernetes Resources
```mql
# Namespaces
k8s.namespaces.all(
  manifest['metadata']['labels']['pod-security.kubernetes.io/enforce'] == "restricted"
)

# Pods
k8s.pods.where(namespace != "kube-system").all(
  manifest['spec']['securityContext']['runAsNonRoot'] == true
)

# RBAC
k8s.rbac.clusterRoles.all(
  rules.none(verbs.contains("*") && resources.contains("*"))
)
```

## Anti-Patterns to Avoid

### Don't Use permissions.string
```mql
# Bad
file("/etc/passwd").permissions.string == "-rw-r--r--"

# Good
file("/etc/passwd").permissions {
  user_readable == true
  user_writeable == true
  user_executable == false
  group_readable == true
  group_writeable == false
  group_executable == false
  other_readable == true
  other_writeable == false
  other_executable == false
}
```

### Don't Use =~ for Regular Expressions
```mql
# Bad
string =~ /pattern/

# Good
string == /pattern/
```

### Don't Use Deprecated platform
```mql
# Bad
platform == "ubuntu"

# Good
asset.platform == "ubuntu"
```

### Don't Nest .where() Clauses
```mql
# Bad
events.where(parameters.where(_['name'] == "NEW_VALUE"))

# Good
events.where(parameters.any(_['name'] == "NEW_VALUE"))
```

### Don't Use Variables for Single Use
```mql
# Bad
filePath = "/etc/passwd"
file(filePath).exists

# Good
file("/etc/passwd").exists
```

### Don't Ignore Null Values
```mql
# Bad
users.all(shell == "/bin/bash")

# Good
users.where(shell != null).all(shell == "/bin/bash")
```

## Error Handling Patterns

### Graceful File Existence Checks
```mql
# Skip checks if file doesn't exist
["/optional/config"].where(file(_).exists) {
  parse.ini(_).params["section"]["key"] == "value"
}

# Require file to exist
file("/required/config").exists == true
parse.ini("/required/config").params["section"]["key"] == "value"
```

### Multiple Condition Checks
```mql
# Use OR for alternative conditions
file("/etc/config1").exists == true ||
file("/etc/config2").exists == true

# Use AND for required conditions
service("app").running == true &&
service("app").enabled == true
```

### Null and Empty Value Handling
```mql
# Check for null before accessing properties
config = parse.ini("/etc/app.conf")
if (config.file.exists && config.params != null) {
  config.params.where(key == "setting").all(value != empty)
}

# Handle arrays that might be empty
users.where(groups != empty).all(
  groups.none(name == "admin")
)
```

## Common Use Cases

### 1. Password Policy Validation
```mql
props:
  - uid: minPasswordLength
    mql: return 8
  - uid: maxPasswordAge
    mql: return 90

mql: |
  loginDefs = parse.ini("/etc/login.defs")
  loginDefs.file.exists
  loginDefs.params["PASS_MIN_LEN"] != null
  loginDefs.params["PASS_MIN_LEN"] >= props.minPasswordLength
  loginDefs.params["PASS_MAX_DAYS"] != null
  loginDefs.params["PASS_MAX_DAYS"] <= props.maxPasswordAge
```

### 2. Certificate Validation
```mql
# Check certificate files
files.find(from: "/etc/ssl/certs", type: "file", regex: "*.crt").all(
  permissions.other_readable == false &&
  permissions.group_writeable == false
)

# Check certificate expiration
certificate("/etc/ssl/certs/server.crt").expiresIn > time.days(30)
```

### 3. Log Configuration
```mql
# Rsyslog configuration
rsyslog.conf.settings.contains("*.info /var/log/messages")

# Log file permissions
file("/var/log/secure").permissions {
  user_readable == true
  user_writeable == true
  group_readable == false
  other_readable == false
}
```

### 4. Network Security
```mql
# Check for insecure protocols
ports.listening.none(port == 23)  # No telnet
ports.listening.none(port == 21)  # No FTP
ports.listening.none(port == 512) # No rsh

# SSH configuration with null checking
sshd.config.params.where(key == "PasswordAuthentication").all(value == "no")
sshd.config.params.where(key == "PermitRootLogin").all(value == "no")
sshd.config.params.where(key == "Protocol").all(value == "2")
```

### 5. Environment Variable Validation
```mql
# Check proxy settings
env.where(key == /proxy/i).all(value != empty && value == /^https?:\/\//)

# Validate PATH variable
env.where(key == "PATH").all(
  value.contains("/usr/bin") &&
  value.contains("/bin") &&
  !value.contains(".")
)
```

## Tips for Effective MQL Development

1. **Start Simple**: Begin with basic assertions and add complexity gradually
2. **Test Incrementally**: Validate each part of complex queries separately
3. **Leverage Relationships**: Use resource relationships instead of manual lookups
4. **Handle Edge Cases**: Consider what happens when files don't exist or services aren't installed
5. **Be Atomic**: Write granular checks for better actionable output
6. **Use Meaningful Names**: Choose descriptive names for variables and properties
7. **Check for Null**: Always consider that values can be null or empty
8. **Use Correct Operators**: Remember that MQL uses `==` for regex matching, not `=~`
9. **Platform Detection**: Use `asset.platform` instead of deprecated `platform`
10. **Props**: Use props only when the user "might" add something to a list of variables, otherwise do not use props. Do not use props for constants.
11. **Query Variants**: When checking for multiple conditions or different system configurations (like 32-bit vs 64-bit), use query variants instead of complex conditional logic in a single query. Variants provide cleaner, more maintainable checks.
12. **MQL Patterns**: Use the context of already existing MQL patterns, look for the relevant sample files.

## Debugging Tips

1. **Check Resource Availability**: Ensure resources exist before accessing properties
2. **Validate Syntax**: Use MQL compiler to check query syntax
3. **Test with Sample Data**: Run queries against known good/bad configurations
4. **Use Incremental Building**: Build complex queries step by step
5. **Check Error Messages**: MQL provides detailed error context for debugging
6. **Handle Null Values**: Add null checks when debugging unexpected results
7. **Use Block Notation**: Use `{ }` blocks to inspect resource fields during development

---

This context should be used when working with MQL policies, security checks, or any Mondoo-related development tasks.
