# Non-compliant: a counted load balancer disables access logs.
resource "aws_lb" "violating" {
  count = 3
  name  = "violating-lb-${count.index}"

  access_logs {
    bucket  = "violating-lb-logs"
    enabled = false
  }
}
