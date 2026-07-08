# Compliant intent: scan-on-push toggled by a feature flag; default is true.
variable "enable_scan" {
  type    = bool
  default = true
}

resource "aws_ecr_repository" "ternary" {
  name = "ternary"
  image_scanning_configuration {
    scan_on_push = var.enable_scan ? true : false
  }
}
