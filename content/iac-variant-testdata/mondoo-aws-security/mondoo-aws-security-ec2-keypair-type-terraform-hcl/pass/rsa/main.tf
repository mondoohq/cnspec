resource "aws_key_pair" "rsa_key" {
  key_name   = "rsa-key"
  key_type   = "rsa"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQexample"
}
