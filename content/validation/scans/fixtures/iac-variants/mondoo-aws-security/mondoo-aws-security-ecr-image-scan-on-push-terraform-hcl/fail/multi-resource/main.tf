# Two repositories; the second disables scan-on-push.
resource "aws_ecr_repository" "compliant" {
  name = "compliant"
  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_repository" "violating" {
  name = "violating"
  image_scanning_configuration {
    scan_on_push = false
  }
}
