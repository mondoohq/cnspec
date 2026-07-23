# Compliant: security configuration includes LocalDiskEncryptionConfiguration.
resource "aws_emr_security_configuration" "pass_example" {
  name = "pass-config"

  configuration = <<JSON
{
  "EncryptionConfiguration": {
    "EnableInTransitEncryption": true,
    "EnableAtRestEncryption": true,
    "LocalDiskEncryptionConfiguration": {
      "EncryptionKeyProviderType": "AwsKms",
      "AwsKmsKey": "arn:aws:kms:us-east-1:123456789012:key/abc"
    }
  }
}
JSON
}
