resource "aws_emr_security_configuration" "main" {
  name = "analytics"

  configuration = jsonencode({
    EncryptionConfiguration = {
      EnableAtRestEncryption    = false
      EnableInTransitEncryption = false
    }
  })
}

resource "aws_emr_cluster" "configured" {
  name                   = "configured"
  release_label          = "emr-7.1.0"
  service_role           = "EMR_DefaultRole"
  security_configuration = aws_emr_security_configuration.main.name
}

resource "aws_emr_cluster" "unconfigured" {
  name          = "unconfigured"
  release_label = "emr-7.1.0"
  service_role  = "EMR_DefaultRole"
}
