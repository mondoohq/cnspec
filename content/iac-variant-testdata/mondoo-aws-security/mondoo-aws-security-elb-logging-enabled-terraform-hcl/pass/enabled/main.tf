resource "aws_lb" "pass" {
  name = "example-lb"

  access_logs {
    bucket  = "example-lb-logs"
    enabled = true
  }
}
