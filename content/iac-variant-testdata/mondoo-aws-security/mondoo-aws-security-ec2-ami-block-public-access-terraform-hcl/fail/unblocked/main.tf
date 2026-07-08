# Non-compliant: AMI public sharing is unblocked.
resource "aws_ec2_image_block_public_access" "fail_example" {
  state = "unblocked"
}
