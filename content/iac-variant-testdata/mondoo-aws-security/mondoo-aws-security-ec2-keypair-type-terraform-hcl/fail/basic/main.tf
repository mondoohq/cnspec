resource "aws_key_pair" "fail_example" {
  key_name   = "fail-key"
  key_type   = "dsa"
  public_key = "ssh-dss AAAAB3NzaC1kc3MAAACBAexample"
}
