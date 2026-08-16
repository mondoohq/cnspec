resource "aws_lb" "fail" {
  name = "example-lb"

  access_logs {
    bucket  = "example-lb-logs"
    enabled = false
  }
}
