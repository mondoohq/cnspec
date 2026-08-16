# Compliant: AMI shared with an AWS Organization, not publicly and not with a
# specific account. Sharing scoped to an organization is not public sharing.
resource "aws_ami_launch_permission" "pass_example" {
  image_id         = "ami-12345678"
  organization_arn = "arn:aws:organizations::111122223333:organization/o-abc123def4"
}
