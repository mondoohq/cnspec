resource "aws_emr_security_configuration" "main" {
  name = "analytics"

  configuration = jsonencode({
    EncryptionConfiguration = {
      EnableAtRestEncryption    = true
      EnableInTransitEncryption = false
      AtRestEncryptionConfiguration = {
        LocalDiskEncryptionConfiguration = {
          EncryptionKeyProviderType = "AwsKms"
          AwsKmsKey                 = "arn:aws:kms:us-east-1:111122223333:key/example"
        }
      }
    }
  })
}

resource "aws_emr_cluster" "analytics" {
  name          = "analytics"
  release_label = "emr-7.1.0"
  service_role  = "EMR_DefaultRole"

  security_configuration = aws_emr_security_configuration.main.name
}
