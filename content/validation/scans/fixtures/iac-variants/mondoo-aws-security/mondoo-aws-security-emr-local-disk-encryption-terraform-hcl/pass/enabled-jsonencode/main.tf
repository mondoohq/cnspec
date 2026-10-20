# Compliant: at-rest encryption is enabled and includes LocalDiskEncryptionConfiguration,
# written with jsonencode() instead of a heredoc.
resource "aws_emr_security_configuration" "pass_example" {
  name = "pass-config"

  configuration = jsonencode({
    EncryptionConfiguration = {
      EnableInTransitEncryption = false
      EnableAtRestEncryption    = true
      AtRestEncryptionConfiguration = {
        LocalDiskEncryptionConfiguration = {
          EncryptionKeyProviderType = "AwsKms"
          AwsKmsKey                 = "arn:aws:kms:us-east-1:123456789012:key/abc"
        }
      }
    }
  })
}
