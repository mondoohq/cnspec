# Non-compliant: security configuration does not include LocalDiskEncryptionConfiguration.
resource "aws_emr_security_configuration" "fail_example" {
  name = "fail-config"

  configuration = <<JSON
{
  "EncryptionConfiguration": {
    "EnableInTransitEncryption": false,
    "EnableAtRestEncryption": false
  }
}
JSON
}
