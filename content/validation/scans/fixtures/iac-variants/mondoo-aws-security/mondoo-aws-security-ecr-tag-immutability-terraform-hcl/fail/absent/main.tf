# Non-compliant: image_tag_mutability omitted, so it defaults to MUTABLE.
resource "aws_ecr_repository" "fail_example" {
  name = "fail-example"
}
