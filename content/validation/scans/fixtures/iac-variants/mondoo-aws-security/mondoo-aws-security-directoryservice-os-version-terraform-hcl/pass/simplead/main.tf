# Compliant: a SimpleAD directory has no os_version and is not a MicrosoftAD, so
# the SERVER_2012 check does not apply to it.
resource "aws_directory_service_directory" "pass_example" {
  name     = "corp.example.com"
  password = "SuperSecretPassw0rd"
  size     = "Small"
  type     = "SimpleAD"

  vpc_settings {
    vpc_id     = "vpc-12345678"
    subnet_ids = ["subnet-12345678", "subnet-87654321"]
  }
}
