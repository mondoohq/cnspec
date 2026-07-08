# Compliant: AMI public sharing is blocked.
resource "aws_ec2_image_block_public_access" "pass_example" {
  state = "block-new-sharing"
}
