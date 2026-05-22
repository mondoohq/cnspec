# AWS MQL Patterns

Unique AWS MQL patterns for security policy analysis and implementation.

## 1. **Time-Based Analysis Patterns**

<Example>
 <MQL>

```mql
// Check for unused credentials over 45 days
aws.iam.credentialReport.where(passwordEnabled == true).all(time.now - passwordLastUsed < 45 * time.day)
aws.iam.credentialReport.where(accessKey1Active == true).all(time.now - accessKey1LastUsedDate < 45 * time.day)

// Handle special case where no usage information exists
aws.iam.credentialReport.where(passwordEnabled == true && properties.password_last_used == "no_information").all(time.now - passwordLastChanged < 45 * time.day)
```

 </MQL>
 <Description>
Password age and credential monitoring that flags unused credentials beyond 45 days, including accounts without last-used data.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Ensure access keys are rotated within 90 days, but only for accounts older than 90 days
aws.iam.credentialReport.where(accessKey1Active == true && time.now - userCreationTime > props.maxAccessKeyAge * time.day).all(time.now - accessKey1LastRotated < props.maxAccessKeyAge * time.day)
```

 </MQL>
 <Description>
Key rotation validation that enforces configurable maximum age for active access keys.
 </Description>

</Example>

## 2. **Complex CloudTrail Log Analysis**

<Example>
 <MQL>

```mql
// Ensure CloudTrail captures all management events across regions
trails = aws.cloudtrail.trails.where(isMultiRegionTrail == true && cloudWatchLogsLogGroupArn != empty)
trails.all(status.IsLogging == true)
trails.all(eventSelectors.any(IncludeManagementEvents == true && ReadWriteType == 'All'))
```

 </MQL>
 <Description>
Multi-region CloudTrail validation confirming logging is active with full management event coverage.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Complex chain: CloudTrail -> CloudWatch Logs -> Metric Filters -> Alarms -> SNS -> Subscriptions
trails.all(logGroup.metricsFilters.any(
  filterPattern.contains(/\$\.eventName(\s+)?\=(\s+)?ConsoleLogin/)
    && filterPattern.contains(/\$\.additionalEventData.MFAUsed(\s+)?\!\=(\s+)?\"Yes\"/)
))

trails.all(logGroup.metricsFilters.where(
  filterPattern.contains(/\$\.eventName(\s+)?\=(\s+)?ConsoleLogin/)
).all(
  metrics.all(
    alarms.all(
      actions.where(arn == /^arn:aws:sns:.*/).any(
        subscriptions.any(arn != empty && arn != /PendingConfirmation|Deleted/)
      )
    )
  )
))
```

 </MQL>
 <Description>
Metric filter and alarm chain validation that traces CloudTrail events through CloudWatch and SNS subscriptions.
 </Description>

</Example>

## 3. **IAM Policy Document Analysis**

<Example>
 <MQL>

```mql
// Check for overly permissive policies
aws.iam.attachedPolicies.where(
  defaultVersion.document.Statement.recurse(
    _['Action'].any(_ == /^\*$/) && _['Resource'].any(_ == /^\*$/)
  ).none(Effect == "Allow") != true
).all(attachedUsers == empty)
```

 </MQL>
 <Description>
Deep policy structure inspection that detects wildcard allow statements attached to users.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Validate AWS Support role configuration
aws.iam.policies.where(name == "AWSSupportAccess").all(
  attachedRoles.any(
    assumeRolePolicyDocument.Statement.any(
      _['Action'] == "sts:AssumeRole" &&
      _['Principal']["AWS"] == /arn:aws:iam::.*:user\/*/
    )
  )
)
```

 </MQL>
 <Description>
Role assumption policy validation for the AWS Support Access policy.
 </Description>

</Example>

## 4. **Network Security Deep Analysis**

<Example>
 <MQL>

```mql
// Check if security groups allow dangerous ports from anywhere
props.disallowedPublicPortsIpv4 {
  disallowedPort = _
  aws.ec2.securityGroups.where(ipPermissions.any(
    ipRanges.contains('0.0.0.0/0'))).all(
      ipPermissions.none(fromPort <= disallowedPort && toPort >= disallowedPort)
  )
}
```

 </MQL>
 <Description>
Security group port range validation using a configurable list of disallowed public ports.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Ensure VPC flow logs are properly configured
aws.vpcs.all(
  flowLogs.any(
    status == "ACTIVE" &&
    destination != empty &&
    destinationType == "cloud-watch-logs" &&
    deliverLogsStatus == "SUCCESS" &&
    (trafficType == "REJECT" || trafficType == "ALL")
  )
)
```

 </MQL>
 <Description>
VPC flow log configuration validation that checks logging status, destination, and traffic type.
 </Description>

</Example>

## 5. **S3 Security Policy Analysis**

<Example>
 <MQL>

```mql
// Ensure S3 buckets deny HTTP requests
aws.s3.buckets.all(
  policy.statements.any(
    _["Condition"]["Bool"]["aws:SecureTransport"] == false &&
    _["Action"].contains(/^s3:.*$/) &&
    _["Principal"]["AWS"].contains(/^\*$/) &&
    _["Effect"] == "Deny"
  )
)
```

 </MQL>
 <Description>
S3 bucket policy statement validation that enforces HTTPS-only access.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Comprehensive S3 public access validation
aws.s3.buckets.all(publicAccessBlock != empty)
aws.s3.buckets.all(publicAccessBlock.BlockPublicAcls == true)
aws.s3.buckets.all(publicAccessBlock.BlockPublicPolicy == true)
aws.s3.buckets.all(publicAccessBlock.IgnorePublicAcls == true)
aws.s3.buckets.all(publicAccessBlock.RestrictPublicBuckets == true)
```

 </MQL>
 <Description>
S3 public access block configuration check covering all required flags.
 </Description>

</Example>

## 6. **Multi-Resource Cross-Validation**

<Example>
 <MQL>

```mql
// Ensure AWS Config is enabled across all regions
aws.config.recorders.all(allSupported == true)
aws.config.recorders.any(includeGlobalResourceTypes == true)
aws.config.recorders.where(allSupported == true).all(recording == true)
aws.config.deliveryChannels.where(s3KeyPrefix == "config").all(s3BucketName != empty)
```

 </MQL>
 <Description>
Config service regional validation confirming recording status and delivery channel setup.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Check Security Hub across all regions
aws.regions {
  region = _
  aws.securityhub.hubs.one(arn.contains(region))
}
```

 </MQL>
 <Description>
Regional resource validation pattern that iterates through regions to confirm Security Hub presence.
 </Description>

</Example>

## 7. **Conditional Logic with Null Handling**

<Example>
 <MQL>

```mql
// Handle potentially null password usage
aws.iam.credentialReport.where(passwordEnabled == true).all(passwordLastUsed == Never) ||
aws.iam.credentialReport.where(passwordEnabled == true).all(time.now - passwordLastUsed > 14 * time.day) ||
aws.iam.credentialReport.where(passwordEnabled == true).all(time.now - userCreationTime < 14 * time.day)
```

 </MQL>
 <Description>
Smart null and empty validation for password usage data within the credential report.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Only check if file exists, then validate contents
["/etc/optional.conf"].where(file(_).exists).all(
  file(_).content == /required_pattern/
)
```

 </MQL>
 <Description>
Conditional file existence validation prior to enforcing configuration content.
 </Description>

</Example>

## 8. **Advanced Regular Expression Patterns**

<Example>
 <MQL>

```mql
// CloudTrail filter pattern validation with complex regex
trails.all(logGroup.metricsFilters.any(
  filterPattern.contains(/\(\$\.eventName(\s+)?\=(\s+)?ConsoleLogin\)/)
    && filterPattern.contains(/\(\$\.errorMessage(\s+)?\=(\s+)?\"Failed authentication\"\)/)
))
```

 </MQL>
 <Description>
Complex filter pattern matching that validates CloudTrail metric filter regular expressions.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Validate SNS topic ARN format and active subscriptions
actions.where(arn == /^arn:aws:sns:.*/).any(
  subscriptions.any(arn != empty && arn != /PendingConfirmation|Deleted/)
)
```

 </MQL>
 <Description>
ARN pattern matching that ensures SNS topics have confirmed subscriptions.
 </Description>

</Example>

## 9. **Certificate and Encryption Validation**

<Example>
 <MQL>

```mql
// Check for expired SSL certificates
aws.iam.serverCertificates == empty ||
aws.iam.serverCertificates.all(parse.date(Expiration) > time.now)
```

 </MQL>
 <Description>
Certificate expiration check that validates IAM server certificates are current.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Validate customer-managed symmetric key rotation
aws.kms.keys
  .where(metadata.KeyState == "Enabled" && metadata.KeyManager == "CUSTOMER")
  .where(metadata.KeySpec == "SYMMETRIC_DEFAULT")
  .all(keyRotationEnabled == true)
```

 </MQL>
 <Description>
KMS key management validation for enabled customer-managed symmetric keys with rotation.
 </Description>

</Example>

## 10. **Database and Storage Analysis**

<Example>
 <MQL>

```mql
// Comprehensive RDS security check
aws.rds.instances.all(storageEncrypted == true)
aws.rds.instances.all(autoMinorVersionUpgrade == true)
aws.rds.instances.all(multiAZ == true)
aws.rds.instances.all(publiclyAccessible == false)
```

 </MQL>
 <Description>
RDS multi-AZ and encryption validation covering availability, encryption, and network exposure.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Ensure EFS file systems are encrypted
aws.efs.filesystems.all(encrypted == true)
```

 </MQL>
 <Description>
EFS encryption validation ensuring all file systems enforce encryption at rest.
 </Description>

</Example>

## 11. **Props-Based Dynamic Validation**

<Example>
 <MQL>

```mql
props:
  - uid: disallowedPublicPortsIpv4
    mql: return [22, 3389]

// Use in validation
props.disallowedPublicPortsIpv4 {
  disallowedPort = _
  aws.ec2.securityGroups.where(ipPermissions.any(
    ipRanges.contains('0.0.0.0/0'))).all(
      ipPermissions.none(fromPort <= disallowedPort && toPort >= disallowedPort)
  )
}
```

 </MQL>
 <Description>
Props-based dynamic validation that defines reusable disallowed port lists for security group checks.
 </Description>

</Example>

## 12. **Account-Level Security Settings**

<Example>
 <MQL>

```mql
// Comprehensive password policy checking
aws.iam.accountPasswordPolicy.MinimumPasswordLength != empty
aws.iam.accountPasswordPolicy.where(MinimumPasswordLength != empty).all(MinimumPasswordLength >= 14)
aws.iam.accountPasswordPolicy.PasswordReusePrevention != empty
aws.iam.accountPasswordPolicy.where(PasswordReusePrevention != empty).all(PasswordReusePrevention >= 24)
```

 </MQL>
 <Description>
Password policy validation that enforces minimum length and reuse prevention requirements.
 </Description>

</Example>

<Example>
 <MQL>

```mql
// Check for hardware MFA (not virtual)
aws.iam.accountSummary.AccountMFAEnabled == 1
aws.iam.virtualMfaDevices.none(serialNumber == /arn:aws:iam::.*:mfa\/root-account-mfa-device/)
```

 </MQL>
 <Description>
MFA device validation that requires hardware MFA for the root account and confirms MFA enablement.
 </Description>

</Example>

## Key Patterns Summary

1. **Time-based validations** with smart null handling
2. **Complex chaining** across multiple AWS services
3. **Deep JSON/policy document** traversal
4. **Regular expression** matching for ARNs and patterns
5. **Props-based configuration** for flexible validation
6. **Multi-resource correlation** (CloudTrail -> CloudWatch -> SNS)
7. **Regional iteration** patterns
8. **Conditional existence** checking before validation
9. **Array operations** with iterator patterns
10. **Certificate and encryption** state validation
