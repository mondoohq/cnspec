# Non-compliant: MicrosoftAD directory uses the end-of-life SERVER_2012 OS version.
resource "aws_directory_service_directory" "fail_example" {
  name       = "corp.example.com"
  password   = "SuperSecretPassw0rd"
  edition    = "Standard"
  type       = "MicrosoftAD"
  os_version = "SERVER_2012"

  vpc_settings {
    vpc_id     = "vpc-12345678"
    subnet_ids = ["subnet-12345678", "subnet-87654321"]
  }
}
