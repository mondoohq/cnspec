# Two load balancers; the second disables access logs, so .all() must fail.
resource "aws_lb" "compliant" {
  name = "compliant-lb"

  access_logs {
    bucket  = "compliant-lb-logs"
    enabled = true
  }
}

resource "aws_lb" "violating" {
  name = "violating-lb"

  access_logs {
    bucket  = "violating-lb-logs"
    enabled = false
  }
}
