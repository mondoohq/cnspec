# Compliant: MicrosoftAD directory uses a supported OS version.
resource "aws_directory_service_directory" "pass_example" {
  name       = "corp.example.com"
  password   = "SuperSecretPassw0rd"
  edition    = "Standard"
  type       = "MicrosoftAD"
  os_version = "SERVER_2019"

  vpc_settings {
    vpc_id     = "vpc-12345678"
    subnet_ids = ["subnet-12345678", "subnet-87654321"]
  }
}
