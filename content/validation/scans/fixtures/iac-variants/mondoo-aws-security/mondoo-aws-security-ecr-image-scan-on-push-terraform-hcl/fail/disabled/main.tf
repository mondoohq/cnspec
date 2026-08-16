# Non-compliant: scanning configuration present but scan_on_push disabled.
resource "aws_ecr_repository" "fail_example" {
  name = "fail-example"

  image_scanning_configuration {
    scan_on_push = false
  }
}
