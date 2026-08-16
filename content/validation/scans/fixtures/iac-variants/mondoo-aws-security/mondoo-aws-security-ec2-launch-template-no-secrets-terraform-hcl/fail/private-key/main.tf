resource "aws_launch_template" "noncompliant" {
  name          = "example"
  image_id      = "ami-12345678"
  instance_type = "t3.micro"
  user_data     = <<-EOT
    -----BEGIN RSA PRIVATE KEY-----
    MIIEpAIBAAKCAQEA1234567890
    -----END RSA PRIVATE KEY-----
  EOT
}
