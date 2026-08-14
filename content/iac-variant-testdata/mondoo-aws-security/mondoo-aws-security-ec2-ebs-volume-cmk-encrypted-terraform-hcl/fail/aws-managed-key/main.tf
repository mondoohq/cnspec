# Encrypted with the default aws/ebs key, which the account cannot rotate on
# its own schedule or deny access to independently.
resource "aws_ebs_volume" "data" {
  availability_zone = "us-east-1a"
  size              = 500
  type              = "gp3"
  encrypted         = true
}
