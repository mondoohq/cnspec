# MQL Patterns for Windows

## Registry Key Value Checks

<Example>
 <MQL>

```mql
value = registrykey.property( path: 'HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows\\WinRM\\Client', name: 'AllowBasic' ).value
switch(value) {
  case _!= empty:
      value == 0
  default:
      false;
}
```

 </MQL>
 <Description>
Pattern for registry keys. Follow this pattern whenever you work with the `registrykey` resource.
 </Description>

</Example>

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\System\\CurrentControlSet\\Control\\Lsa', name: 'LimitBlankPasswordUse').value == 1
```

 </MQL>
 <Description>
Basic registry value comparison.
 </Description>

</Example>

<Example>
 <MQL>

```mql
passwordLength = registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\LAPS', name: 'PasswordLength').value;
switch(passwordLength) {
  case _ != empty:
      passwordLength >= 15;
  default:
      false;
}
```

 </MQL>
 <Description>
Registry value with conditional logic.
 </Description>

</Example>

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon', name: 'PasswordExpiryWarning').value <= 14
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon', name: 'PasswordExpiryWarning').value >= 5
```

 </MQL>
 <Description>
Registry value range check that ensures the password expiry warning is between 5 and 14 days.
 </Description>

</Example>

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\Netlogon\\Parameters', name: 'vulnerablechannelallowlist').exists == false
```

 </MQL>
 <Description>
Registry key existence check.
 </Description>

</Example>

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\LanManServer\\Parameters', name: 'NullSessionPipes').value.downcase == /lsarpc|netlogon|samr/
```

 </MQL>
 <Description>
Registry value pattern matching using a regular expression.
 </Description>

</Example>

## Security Policy (secpol) Checks

<Example>
 <MQL>

```mql
secpol.privilegerights['SeBackupPrivilege'].containsOnly(['S-1-5-32-544'])
```

 </MQL>
 <Description>
User rights assignment check that restricts the SeBackupPrivilege.
 </Description>

</Example>

<Example>
 <MQL>

```mql
secpol.privilegerights['SeTimeZonePrivilege'].contains('S-1-5-32-544')
secpol.privilegerights['SeTimeZonePrivilege'].contains('S-1-5-19')
secpol.privilegerights['SeTimeZonePrivilege'].length == 2
```

 </MQL>
 <Description>
Complex user rights validation ensuring only the expected SIDs hold SeTimeZonePrivilege.
 </Description>

</Example>

<Example>
 <MQL>

```mql
secpol.systemaccess['PasswordHistorySize'] >= 24
secpol.systemaccess['MaximumPasswordAge'] <= 365
secpol.systemaccess['MaximumPasswordAge'] > 0
```

 </MQL>
 <Description>
System access settings for password history and age requirements.
 </Description>

</Example>

<Example>
 <MQL>

```mql
switch (secpol.systemaccess.LockoutDuration.length) {
  case _ != empty:
      secpol.systemaccess['LockoutDuration'] >= 15
  default:
      false;
}
```

 </MQL>
 <Description>
Conditional secpol check that validates lockout duration when the value exists.
 </Description>

</Example>

## Audit Policy Checks

<Example>
 <MQL>

```mql
auditpol.where(subcategoryguid == "0CCE923F-69AE-11D9-BED3-505054503030").list != []
auditpol.where(subcategoryguid == "0CCE923F-69AE-11D9-BED3-505054503030").all(inclusionsetting == props.auditpolSuccessFailure)
```

 </MQL>
 <Description>
Audit subcategory configuration for SMB server audit events.
 </Description>

</Example>

## User Account Checks

<Example>
 <MQL>

```mql
users.where(sid == /S-1-5-21-\d+-\d+-\d+-500/).all(enabled == false)
```

 </MQL>
 <Description>
User account status check that disables built-in administrator accounts.
 </Description>

</Example>

<Example>
 <MQL>

```mql
users.where(sid == /S-1-5-21-\d+-\d+-\d+-500/).all(name != "Administrator")
```

 </MQL>
 <Description>
User account name verification to ensure the default administrator account is renamed.
 </Description>

</Example>

## PowerShell Integration

<Example>
 <MQL>

```mql
validUsers = parse.json(content: powershell("Get-CimInstance -ClassName 'Win32_UserProfile' -Filter 'Special=False' | Select-Object -ExpandProperty SID | ConvertTo-Json").stdout).params.where(value == /^S-1-5-21-[0-9]+-[0-9]+-[0-9]+-[0-9]{3,}$/).map('HKEY_USERS\\' + string(_))
```

 </MQL>
 <Description>
Dynamic user SID collection using PowerShell output.
 </Description>

</Example>

<Example>
 <MQL>

```mql
validUsers = parse.json(content: powershell("Get-CimInstance -ClassName 'Win32_UserProfile' -Filter 'Special=False' | Select-Object -ExpandProperty SID | ConvertTo-Json").stdout).params.where(value == /^S-1-5-21-[0-9]+-[0-9]+-[0-9]+-[0-9]{3,}$/).map('HKEY_USERS\\' + string(_))
validPaths = validUsers.where(registrykey(path = _).exists).map(_ + '\\Software\\Policies\\Microsoft\\Windows\\Control Panel\\Desktop')
validPaths.map(registrykey.property(path: _, name: 'ScreenSaveActive').value == 1)
```

 </MQL>
 <Description>
Per-user registry check ensuring ScreenSaveActive is enabled for each valid user profile.
 </Description>

</Example>

<Example>
 <MQL>

```mql
parse.json(content: powershell('Get-WinEvent @{logname="Microsoft-Windows-SMBServer/Audit"; starttime = "$((Get-Date).adddays(-1))" ; id="3000" } | ConvertTo-Json -Compress;').stdout).params == empty
```

 </MQL>
 <Description>
Event log analysis that confirms no SMB server audit events with ID 3000 occurred in the last day.
 </Description>

</Example>

## Conditional Logic Based on OS Type

<Example>
 <MQL>

```mql
if(windows.computerInfo['OsProductType'] == 2) {
  secpol.privilegerights['SeCreateSymbolicLinkPrivilege'].containsOnly(['S-1-5-32-544'])
}
else {
  secpol.privilegerights['SeCreateSymbolicLinkPrivilege'].containsOnly(['S-1-5-32-544', 'S-1-5-83-0'])
}
```

 </MQL>
 <Description>
Server versus workstation logic that adjusts symbolic link privileges based on OS product type.
 </Description>

</Example>

<Example>
 <MQL>

```mql
if(windows.computerInfo['OsProductType'] == 3) {
  users.where(sid == /S-1-5-21-\d+-\d+-\d+-501/).all(enabled == false)
}
```

 </MQL>
 <Description>
Domain controller specific check that disables the built-in guest account.
 </Description>

</Example>

## Asset Build Version Checks

<Example>
 <MQL>

```mql
asset.build >= "4252" ||
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Winlogon\\GPExtensions\\{D76B9641-3288-4f75-942D-087DE603E3EA}', name: 'DllName').value.downcase == 'c:\\program files\\laps\\cse\\admpwd.dll'
```

 </MQL>
 <Description>
Version-based conditional logic that accepts a minimum asset build or a specific LAPS DLL configuration.
 </Description>

</Example>

## Windows Features Check

<Example>
 <MQL>

```mql
props.deniedWindowsFeatures {
  windows.feature(_) {
    name
    installed == false
  }
}
```

 </MQL>
 <Description>
Disabled Windows features listing that confirms specified features are not installed.
 </Description>

</Example>

<Example>
 <MQL>

```mql
windows.computerInfo['WindowsInstallationType'].downcase.contains('core')
```

 </MQL>
 <Description>
OS installation type check that verifies the system runs a core installation.
 </Description>

</Example>

## Complex Value Validation

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\LAPS', name: 'PasswordExpirationProtectionEnabled').value == 1 ||
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\LAPS', name: 'PwdExpirationProtectionEnabled').value == 1
```

 </MQL>
 <Description>
Multiple registry OR conditions for LAPS password expiration protection.
 </Description>

</Example>

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\System\\CurrentControlSet\\Control\\Lsa\\MSV1_0', name: 'RestrictSendingNTLMTraffic').value.in(["1","2"])
```

 </MQL>
 <Description>
Value-in-set check confirming NTLM traffic restrictions are configured.
 </Description>

</Example>

<Example>
 <MQL>

```mql
secpol.privilegerights['SeImpersonatePrivilege'] { _ == /S-1-5-(32-544|19|20|6|32-568)/ }
```

 </MQL>
 <Description>
Regex pattern with SID validation for impersonate privilege assignments.
 </Description>

</Example>

## Service Configuration Checks

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\Spooler', name: 'Start').value == 4
```

 </MQL>
 <Description>
Service start type verification ensuring the Print Spooler service is disabled.
 </Description>

</Example>

<Example>
 <MQL>

```mql
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\LxssManager', name: 'Start').exists == false ||
registrykey.property(path: 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\LxssManager', name: 'Start').value == 4
```

 </MQL>
 <Description>
Service existence check with fallback to confirm WSL service is absent or disabled.
 </Description>

</Example>

## Advanced User Registry Iteration

<Example>
 <MQL>

```mql
users.where(sid == /^S-1-5-21-[0-9]+-[0-9]+-[0-9]+-[0-9]{3,}$/ && enabled == true).
  where(registrykey(path: 'HKEY_USERS\\' + sid).exists) {
    registrykey.property(path: 'HKEY_USERS\\' + sid + '\\Software\\Policies\\Microsoft\\Windows\\Control Panel\\Desktop', name: 'ScreenSaveTimeOut').value == "" ||
    registrykey.property(path: 'HKEY_USERS\\' + sid + '\\Software\\Policies\\Microsoft\\Windows\\Control Panel\\Desktop', name: 'ScreenSaveTimeOut').value <= 900 ||
    registrykey.property(path: 'HKEY_USERS\\' + sid + '\\Software\\Policies\\Microsoft\\Windows\\Control Panel\\Desktop', name: 'ScreenSaveTimeOut').value != "0"
}
```

 </MQL>
 <Description>
Complex user-specific registry iteration that enforces screen saver timeout policies.
 </Description>

</Example>
