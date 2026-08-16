resource "aws_glue_security_configuration" "example" {
  name = "example-security-config"

  encryption_configuration {
    cloudwatch_encryption {
      cloudwatch_encryption_mode = "DISABLED"
    }
  }
}
