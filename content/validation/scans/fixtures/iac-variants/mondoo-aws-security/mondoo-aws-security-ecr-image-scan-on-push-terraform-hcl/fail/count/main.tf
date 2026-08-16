# Non-compliant: counted repositories with scan-on-push disabled.
resource "aws_ecr_repository" "counted" {
  count = 2
  name  = "counted-${count.index}"
  image_scanning_configuration {
    scan_on_push = false
  }
}
