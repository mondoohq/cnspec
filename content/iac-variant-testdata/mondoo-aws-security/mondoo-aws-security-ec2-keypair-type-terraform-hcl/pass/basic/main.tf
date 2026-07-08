resource "aws_key_pair" "pass_example" {
  key_name   = "pass-key"
  key_type   = "ed25519"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAexample"
}
