# Non-compliant: image tags are mutable.
resource "aws_ecr_repository" "fail_example" {
  name                 = "fail-example"
  image_tag_mutability = "MUTABLE"
}
