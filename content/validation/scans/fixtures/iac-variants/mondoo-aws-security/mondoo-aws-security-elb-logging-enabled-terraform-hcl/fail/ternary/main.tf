# Non-compliant: access logging toggled by a ternary whose active branch is false.
variable "logging" {
  type    = bool
  default = false
}

resource "aws_lb" "violating" {
  name = "violating-lb"

  access_logs {
    bucket  = "violating-lb-logs"
    enabled = var.logging ? true : false
  }
}
