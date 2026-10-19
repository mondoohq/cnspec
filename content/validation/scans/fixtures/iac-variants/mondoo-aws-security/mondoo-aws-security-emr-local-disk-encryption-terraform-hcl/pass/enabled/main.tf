# Compliant: at-rest encryption is enabled and includes LocalDiskEncryptionConfiguration.
resource "aws_emr_security_configuration" "pass_example" {
  name = "pass-config"

  configuration = <<JSON
{
  "EncryptionConfiguration": {
    "EnableInTransitEncryption": false,
    "EnableAtRestEncryption": true,
    "AtRestEncryptionConfiguration": {
      "LocalDiskEncryptionConfiguration": {
        "EncryptionKeyProviderType": "AwsKms",
        "AwsKmsKey": "arn:aws:kms:us-east-1:123456789012:key/abc"
      }
    }
  }
}
JSON
}
