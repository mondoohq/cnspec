# Non-compliant: no image_scanning_configuration block, so scan-on-push is off.
resource "aws_ecr_repository" "fail_example" {
  name = "fail-example"
}
