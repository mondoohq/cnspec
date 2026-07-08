# Compliant: image scanning on push enabled.
resource "aws_ecr_repository" "pass_example" {
  name = "pass-example"

  image_scanning_configuration {
    scan_on_push = true
  }
}
