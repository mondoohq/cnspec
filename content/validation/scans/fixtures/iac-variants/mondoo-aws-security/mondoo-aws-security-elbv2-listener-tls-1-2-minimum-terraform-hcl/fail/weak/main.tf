# Non-compliant: HTTPS listener uses a legacy SSL policy weaker than TLS 1.2.
resource "aws_lb_listener" "fail_example" {
  load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/example/abc"
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
}
