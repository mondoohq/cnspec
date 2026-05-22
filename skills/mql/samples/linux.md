# MQL Samples for Linux

## 1. File and Directory Operations

<Example>
 <MQL>

```mql
file("/etc/passwd").exists
file("/etc/shadow") {
  user.name == 'root'
  group.name == 'root'
  permissions.user_readable == true
  permissions.group_writeable == false
}
```

 </MQL>
 <Description>
Basic file existence and permission validation for critical system files.
 </Description>

</Example>

<Example>
 <MQL>

```mql
file("/etc/issue").content.downcase.contains(asset.platform) == false
file("/etc/login.defs").content.lines.where(_ == /^[^#]/).any(_ == /^ENCRYPT_METHOD\s+yescrypt/)
```

 </MQL>
 <Description>
File content analysis that filters comments and checks for hardened configuration values.
 </Description>

</Example>

<Example>
 <MQL>

```mql
files.find(from: "/etc/audit/", type: "file") {
  permissions.group_writeable == false
  permissions.other_writeable == false
}
```

 </MQL>
 <Description>
Directory traversal with filters to locate files that violate permission requirements.
 </Description>

</Example>

<Example>
 <MQL>

```mql
command("df --local -P | awk '{if (NR!=1) print $6}' | xargs -I '{}' find '{}' -xdev -type f -perm -0002").stdout == ""
```

 </MQL>
 <Description>
Complex file system search combining shell commands to detect world-writable files.
 </Description>

</Example>

## 2. Service and Package Management

<Example>
 <MQL>

```mql
service("auditd").enabled && service("auditd").running
service("bluetooth").enabled == false && service("bluetooth").masked == true
```

 </MQL>
 <Description>
Service state verification ensuring required daemons run while unnecessary ones remain disabled.
 </Description>

</Example>

<Example>
 <MQL>

```mql
package("aide").installed
package("rsync").installed == false || service("rsync").masked
```

 </MQL>
 <Description>
Package installation checks with logical fallbacks tied to related service states.
 </Description>

</Example>

<Example>
 <MQL>

```mql
package("pam").version >= semver("1.3.1-24")
```

 </MQL>
 <Description>
Version comparison using semantic version parsing to enforce minimum package versions.
 </Description>

</Example>

## 3. System Configuration Analysis

<Example>
 <MQL>

```mql
kernel.parameters['net.ipv4.ip_forward'] != 1
kernel.module("usb_storage").loaded == false
```

 </MQL>
 <Description>
Kernel parameter and module checks that verify forwarding is disabled and removable storage modules are unloaded.
 </Description>

</Example>

<Example>
 <MQL>

```mql
mount.where(path == "/tmp").list {
  options["noexec"] != null
  options["noexec"] == ''
}
```

 </MQL>
 <Description>
Mount point analysis validating that `/tmp` includes the expected `noexec` option.
 </Description>

</Example>

<Example>
 <MQL>

```mql
parse.ini("/etc/selinux/config").params["SELINUX"] == "enforcing"
parse.json(content: command("apparmor_status --json").stdout).params.profiles
```

 </MQL>
 <Description>
Configuration file parsing across INI and JSON formats to confirm mandatory security settings.
 </Description>

</Example>

## 4. User and Group Management

<Example>
 <MQL>

```mql
users.where(name == "root").list.all(gid == 0)
users.where(uid == 0).all(name == "root")
```

 </MQL>
 <Description>
User property verification that ensures UID and GID 0 are exclusively assigned to root.
 </Description>

</Example>

<Example>
 <MQL>

```mql
shadow.where(password != "!*" && password != "!" && password != "!!" && password != "*").all(maxdays <= 365)
```

 </MQL>
 <Description>
Shadow file analysis enforcing maximum password age where usable hashes are present.
 </Description>

</Example>

<Example>
 <MQL>

```mql
users.where(shell.contains("nologin") == false).list {
  file(home).permissions.other_writeable == false
}
```

 </MQL>
 <Description>
Home directory checks that cross-reference interactive users with directory permissions.
 </Description>

</Example>

## 5. Network and Security Configuration

<Example>
 <MQL>

```mql
ports.listening.where(protocol == "tcp4" || protocol == "udp4").where(address != /^127\.\d{1,3}\.\d{1,3}\.\d{1,3}(\:\d+)?$/).map(port)
```

 </MQL>
 <Description>
Port and network analysis filtering out loopback bindings to surface externally reachable ports.
 </Description>

</Example>

<Example>
 <MQL>

```mql
iptables.input.contains(target == "ACCEPT" && protocol == /all|0/ && source == "0.0.0.0/0" && in == "lo")
```

 </MQL>
 <Description>
Firewall rule verification looking for overly permissive loopback allowances.
 </Description>

</Example>

<Example>
 <MQL>

```mql
defaultBlock = sshd.config.blocks.where(criteria.in([""]) == props.checkDefaultMatchBlock && criteria == "");
userBlocks = sshd.config.blocks.where(criteria.contains(props.excludedMatchBlocks) == false && criteria != "");
userBlocks.all(params.PermitRootLogin == 'no')
```

 </MQL>
 <Description>
SSH configuration analysis that separates default and user-specific `Match` blocks before enforcing settings.
 </Description>

</Example>

## 6. Audit System Configuration

<Example>
 <MQL>

```mql
props.auditFiles.any(_.contains(/^(\s+)?\-w\s+\/etc\/passwd\s+\-p\s+wa\s+\-k\s+identity(\s+)?$/))
```

 </MQL>
 <Description>
Audit rules verification using regex to confirm presence of identity monitoring rules.
 </Description>

</Example>

<Example>
 <MQL>

```mql
props.auditFiles.flat.unique.where(_ == /chmod|fchmod|fchmodat/).all(
  split("-").containsAll(["a always,exit "])
  && split("-").containsAll(["F arch=b64 "])
  && split("-").containsAll(["F key=perm_mod"])
)
```

 </MQL>
 <Description>
Complex audit checks that parse rule components to ensure syscall monitoring coverage.
 </Description>

</Example>

## 7. Conditional Logic and Platform Detection

<Example>
 <MQL>

```mql
switch {
  case asset.platform == "amazonlinux" && asset.version == /2023|2017|2018/:
    ["/etc/audit/auditd.conf"].where(file(_).exists) {
      parse.ini(_).params["space_left_action"].downcase == /email|exec|single|halt/
    };
  case asset.platform == "debian":
    service("cron").enabled;
  default:
    service("crond").enabled;
}
```

 </MQL>
 <Description>
Asset-based conditionals employing switch statements for platform-specific validation paths.
 </Description>

</Example>

<Example>
 <MQL>

```mql
if(asset.family.contains('debian')) {
  package("apparmor").installed
} else {
  package("libselinux").installed
}
```

 </MQL>
 <Description>
Family-based checks that select the correct mandatory security package for each OS family.
 </Description>

</Example>

## 8. Advanced Data Processing

<Example>
 <MQL>

```mql
userHome = users.where(shell.contains("nologin") == false).map(home)
userHome {
  allDotFiles = files.find(from: _, type: "file").where(path == /\/\./)
  allDotFiles.all(permissions.other_writeable == false)
}
```

 </MQL>
 <Description>
Complex filtering and mapping workflow that gathers user homes then validates dotfile permissions.
 </Description>

</Example>

<Example>
 <MQL>

```mql
props.dconfDbPaths.where(parse.ini(_).sections['org/gnome/login-screen']['banner-message-enable'] != empty)
  .all(parse.ini(_).sections['org/gnome/login-screen']['banner-message-enable'] == true)
```

 </MQL>
 <Description>
Array and collection operations that filter dconf databases before enforcing GNOME login banner settings.
 </Description>

</Example>

<Example>
 <MQL>

```mql
command('find / -perm /6000 -type f').stdout.trim.lines {
  spPath = _
  privUseCollectedOnDisk = props.auditFiles.flat.unique.any(_.contains(spPath))
  spPath
}
```

 </MQL>
 <Description>
Command output processing iterating over setuid files and correlating them with collected audit data.
 </Description>

</Example>

## 9. Error Handling and Validation

<Example>
 <MQL>

```mql
file("/etc/cron.deny").exists == false
file("/etc/cron.allow").exists == true;
["/etc/cron.allow"].where(file(_).exists) {
  file(_).permissions.user_readable == true
}
```

 </MQL>
 <Description>
Existence checks with fallbacks that validate cron permission files before applying permission logic.
 </Description>

</Example>

<Example>
 <MQL>

```mql
sshd.config.ciphers != null
sshd.config.ciphers != []
sshd.config.ciphers.containsOnly(props.sshdCiphers)
```

 </MQL>
 <Description>
Null and empty checks preceding enforcement of approved SSH cipher lists.
 </Description>

</Example>

## 10. Time and Date Operations

<Example>
 <MQL>

```mql
command('date -u --date="$(curl -v install.mondoo.com 2>&1 | grep Date: | cut -d" " -f3-9)"').stdout.trim <= command('date --date "1 min" -u').stdout.trim
```

 </MQL>
 <Description>
Time synchronization check comparing local time to an external HTTPS response.
 </Description>

</Example>

<Example>
 <MQL>

```mql
shadow.where(password != "!*").all(lastchanged < time.now)
```

 </MQL>
 <Description>
Age-based validation ensuring password change timestamps precede the current time.
 </Description>

</Example>

## Key MQL Concepts Demonstrated

1. **Chaining Operations**: Using `.where()`, `.all()`, `.any()`, `.map()` in sequence
2. **Regex Patterns**: Extensive use of regex for content matching
3. **Property Access**: Deep object property access with dot notation
4. **Conditional Logic**: Complex if/else and switch statements
5. **Data Transformation**: Converting and processing data through multiple steps
6. **Resource Integration**: Combining multiple system resources (files, users, services)
7. **String Processing**: Advanced string manipulation and parsing
8. **Collection Operations**: Working with arrays and collections efficiently
9. **Platform Abstraction**: Writing cross-platform compatible checks
10. **Error Prevention**: Defensive programming with existence checks

These patterns form the foundation for writing robust security compliance checks in MQL.
