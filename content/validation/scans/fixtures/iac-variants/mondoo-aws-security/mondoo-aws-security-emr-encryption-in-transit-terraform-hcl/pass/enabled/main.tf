# Compliant: security configuration includes InTransitEncryptionConfiguration.
resource "aws_emr_security_configuration" "pass_example" {
  name = "pass-config"

  configuration = <<JSON
{
  "EncryptionConfiguration": {
    "EnableInTransitEncryption": true,
    "EnableAtRestEncryption": true,
    "InTransitEncryptionConfiguration": {
      "EncryptionKeyProviderType": "AwsKms",
      "AwsKmsKey": "arn:aws:kms:us-east-1:123456789012:key/abc"
    }
  }
}
JSON
}
