# Compliant: image tags are immutable.
resource "aws_ecr_repository" "pass_example" {
  name                 = "pass-example"
  image_tag_mutability = "IMMUTABLE"
}
